// Package gitlab provides GitLab API client functionality with rate limiting.
//
// The package implements a Service layer that wraps the official GitLab Go client
// with additional features:
//   - Per-endpoint rate limiting (download: 5/min, export: 6/min, import: 6/min)
//   - Project and group operations (list, get, create)
//   - Export/import workflow with polling and status tracking
//   - Project emptiness validation for safe restore
//
// Rate Limiting:
// The Service enforces GitLab API rate limits per endpoint using token buckets:
//   - Download API: 5 requests/minute (repository files)
//   - Export API: 6 requests/minute (project export)
//   - Import API: 6 requests/minute (project import)
//
// Key Interfaces:
//   - Client: GitLab API operations (implemented by Service)
//   - RateLimiter: Token bucket rate limiter
//
// Architecture:
//
//	Service
//	    ├─> gitlab.Client (official client)
//	    ├─> downloadLimiter (5 req/min)
//	    ├─> exportLimiter (6 req/min)
//	    └─> importLimiter (6 req/min)
//
// Example usage:
//
//	service := gitlab.NewService(baseURL, token, logger)
//
//	// Export project
//	exportID, err := service.ExportProject(ctx, projectID)
//	status, err := service.WaitForExportCompletion(ctx, projectID, exportID, timeout)
//
//	// Import project
//	importID, err := service.ImportProject(ctx, namespace, projectPath, archivePath)
//	status, err := service.WaitForImportCompletion(ctx, projectID, timeout)
package gitlab

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/time/rate"
)

var (
	// ErrHTTPStatusCode is returned when an unexpected HTTP status code is received.
	ErrHTTPStatusCode = errors.New("unexpected HTTP status code")
)

const (
	// GitlabAPIEndpoint is the default GitLab API endpoint.
	GitlabAPIEndpoint = "https://gitlab.com/api/v4"

	// DownloadRateLimitIntervalSeconds defines the rate limit interval for download API calls.
	// Based on GitLab repository files API limit: 5 requests per minute per user.
	DownloadRateLimitIntervalSeconds = 60
	// DownloadRateLimitBurst defines the burst limit for download API calls.
	DownloadRateLimitBurst = 5
	// ExportRateLimitIntervalSeconds defines the rate limit interval for export API calls.
	// Based on GitLab project import/export API limit: 6 requests per minute per user.
	ExportRateLimitIntervalSeconds = 60
	// ExportRateLimitBurst defines the burst limit for export API calls.
	ExportRateLimitBurst = 6
	// ImportRateLimitIntervalSeconds defines the rate limit interval for import API calls.
	// Based on GitLab project import/export API limit: 6 requests per minute per user.
	ImportRateLimitIntervalSeconds = 60
	// ImportRateLimitBurst defines the burst limit for import API calls.
	ImportRateLimitBurst = 6
	// DefaultExportTimeoutMins defines the default export timeout in minutes.
	DefaultExportTimeoutMins = 10
)

var log Logger

// Logger interface defines the logging methods used by GitLab service.
type Logger interface {
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
}

// Service provides methods to interact with GitLab API.
type Service struct {
	mu                    sync.RWMutex
	client                GitLabClient
	gitlabAPIEndpoint     string
	token                 string
	rateLimitDownloadAPI  *rate.Limiter
	rateLimitExportAPI    *rate.Limiter
	rateLimitImportAPI    *rate.Limiter
	exportTimeoutDuration time.Duration
}

func init() {
	log = slog.New(slog.DiscardHandler)
}

// NewGitlabService returns a new Service with default timeout.
func NewGitlabService() *Service {
	return NewGitlabServiceWithTimeout(DefaultExportTimeoutMins)
}

// NewGitlabServiceWithTimeout returns a new Service with configurable timeout.
func NewGitlabServiceWithTimeout(timeoutMins int) *Service {
	token := os.Getenv("GITLAB_TOKEN")
	glClient, err := gitlab.NewClient(token)
	if err != nil {
		log.Error("failed to create GitLab client", "error", err)
		return nil
	}

	client := NewGitLabClientWrapper(glClient)

	gs := &Service{
		client:            client,
		gitlabAPIEndpoint: GitlabAPIEndpoint,
		token:             token,
		exportTimeoutDuration: time.Duration(timeoutMins) * time.Minute,
		// implement rate limiting https://docs.gitlab.com/ee/administration/settings/import_export_rate_limits.html
		rateLimitDownloadAPI: rate.NewLimiter(
			rate.Every(DownloadRateLimitIntervalSeconds*time.Second),
			DownloadRateLimitBurst,
		),
		rateLimitExportAPI: rate.NewLimiter(
			rate.Every(ExportRateLimitIntervalSeconds*time.Second),
			ExportRateLimitBurst,
		),
		rateLimitImportAPI: rate.NewLimiter(
			rate.Every(ImportRateLimitIntervalSeconds*time.Second),
			ImportRateLimitBurst,
		),
	}
	return gs
}

// SetLogger sets the logger.
func SetLogger(l Logger) {
	if l != nil {
		log = l
	}
}

// SetGitlabEndpoint sets the Gitlab API endpoint
// default: https://gitlab.com/v4/api/
func (r *Service) SetGitlabEndpoint(gitlabAPIEndpoint string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.gitlabAPIEndpoint = gitlabAPIEndpoint
	// Create a new client with the custom base URL
	glClient, err := gitlab.NewClient(r.token, gitlab.WithBaseURL(gitlabAPIEndpoint))
	if err != nil {
		log.Error("failed to create GitLab client with custom base URL", "error", err, "url", gitlabAPIEndpoint)
		return
	}
	r.client = NewGitLabClientWrapper(glClient)
}

// SetToken sets the Gitlab API token
// default: GITLAB_TOKEN env variable
func (r *Service) SetToken(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if token == "" {
		log.Warn("no token provided")
	}
	r.token = token
	// Create a new client with the new token
	var glClient *gitlab.Client
	var err error
	if r.gitlabAPIEndpoint != GitlabAPIEndpoint {
		glClient, err = gitlab.NewClient(token, gitlab.WithBaseURL(r.gitlabAPIEndpoint))
	} else {
		glClient, err = gitlab.NewClient(token)
	}
	if err != nil {
		log.Error("failed to create GitLab client with new token", "error", err)
		return
	}
	r.client = NewGitLabClientWrapper(glClient)
}

// GetGroup returns the gitlab group from the given ID.
func (r *Service) GetGroup(ctx context.Context, groupID int64) (Group, error) {
	group, _, err := r.client.Groups().GetGroup(groupID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return Group{}, fmt.Errorf("error retrieving group: %w", err)
	}

	return Group{
		ID:   group.ID,
		Name: group.Name,
	}, nil
}

// GetProject returns informations of the project that matches the given ID.
func (r *Service) GetProject(ctx context.Context, projectID int64) (Project, error) {
	project, _, err := r.client.Projects().GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return Project{}, fmt.Errorf("error retrieving project: %w", err)
	}

	return Project{
		ID:           project.ID,
		Name:         project.Name,
		Archived:     project.Archived,
		ExportStatus: "", // ExportStatus not available in project struct, will be fetched separately when needed
	}, nil
}

// Client returns the underlying GitLab client for advanced operations.
//nolint:ireturn // Returning interface is intentional for abstraction
func (r *Service) Client() GitLabClient {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

// RateLimitImportAPI returns the import API rate limiter.
func (r *Service) RateLimitImportAPI() *rate.Limiter {
	return r.rateLimitImportAPI
}
