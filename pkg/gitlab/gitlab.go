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
// See "Rate Limiting" documentation block (around line 60) for comprehensive details on
// enforcement mechanisms, consequences, tuning guidance, and recovery procedures.
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

	"github.com/sgaunet/gitlab-backup/pkg/constants"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/time/rate"
)

var (
	// ErrHTTPStatusCode is returned when an unexpected HTTP status code is received.
	ErrHTTPStatusCode = errors.New("unexpected HTTP status code")
)

// Rate Limiting
//
// This package enforces GitLab API rate limits using token bucket rate limiters
// (golang.org/x/time/rate) to prevent API throttling and potential account
// restrictions. Each API endpoint has independent rate limiting configured
// according to GitLab's documented limits.
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ Enforcement Mechanism: Token Bucket Algorithm                       │
// ├─────────────────────────────────────────────────────────────────────┤
// │                                                                       │
// │  Each rate limiter maintains a "bucket" of tokens:                   │
// │    - Bucket capacity = Burst value (max tokens)                      │
// │    - Refill rate = Interval / Burst (tokens per second)              │
// │    - Each API call consumes one token                                │
// │    - Wait() blocks until a token is available                        │
// │                                                                       │
// │  Example (Download API):                                             │
// │    - Capacity: 5 tokens                                              │
// │    - Refill: 1 token every 12 seconds (60s / 5 = 12s)               │
// │    - Result: Maximum 5 requests per minute                           │
// │                                                                       │
// └─────────────────────────────────────────────────────────────────────┘
//
// Default Rate Limits (per user):
//
//	┌───────────────────┬──────────┬───────┬─────────────────────────────┐
//	│ API Endpoint      │ Interval │ Burst │ Effective Limit             │
//	├───────────────────┼──────────┼───────┼─────────────────────────────┤
//	│ Download API      │ 60s      │ 5     │ 5 requests/minute           │
//	│ Export API        │ 60s      │ 6     │ 6 requests/minute           │
//	│ Import API        │ 60s      │ 6     │ 6 requests/minute           │
//	└───────────────────┴──────────┴───────┴─────────────────────────────┘
//
// Consequences of Exceeding Limits:
//
//  1. HTTP 429 Response: GitLab returns "429 Too Many Requests" with:
//     - Plain text body: "Retry later"
//     - Retry-After header: Seconds to wait before retry
//     - Rate limit headers (on some endpoints):
//       - X-RateLimit-Limit: Maximum requests allowed
//       - X-RateLimit-Remaining: Requests remaining in current window
//       - X-RateLimit-Reset: Unix timestamp when limit resets
//
//  2. Request Rejection: The request is not processed by GitLab
//
//  3. Cooldown Period: Must wait for rate limit window to reset (up to 60 seconds)
//
//  4. Logging: GitLab instances log rate limit violations in auth.log
//
//  5. Potential Account Restrictions: Repeated violations on GitLab.com may result
//     in account review or temporary restrictions (rare, but documented in ToS)
//
// Tuning Guidance:
//
// ⚠️  WARNING: DO NOT modify rate limit constants unless absolutely necessary.
//
//	The default values are based on GitLab's documented API limits and should
//	work for all GitLab instances (gitlab.com and self-managed).
//
// When to Adjust (rare cases):
//
//	1. Self-managed GitLab with custom rate limits:
//	   - Verify actual limits: Admin > Settings > Network > Import and export rate limits
//	   - Reduce constants to match or stay below server limits
//	   - NEVER increase above server limits
//
//	2. Aggressive rate limit enforcement:
//	   - Reduce Burst values to be more conservative
//	   - Increase Interval values to slow down requests
//
// How to Adjust:
//
//	Option 1: Modify constants in this file (requires recompilation):
//	  const DownloadRateLimitBurst = 3  // Reduce from 5 to 3
//
//	Option 2: Create custom Service with modified rate limiters (programmatic):
//	  service := gitlab.NewGitlabService()
//	  service.rateLimitDownloadAPI = rate.NewLimiter(
//	      rate.Every(60*time.Second),
//	      3,  // Custom burst
//	  )
//
// DO NOT DO:
//
//	❌ Increase burst values above GitLab's documented limits
//	❌ Decrease interval values (speeds up requests)
//	❌ Set limits to 0 or disable rate limiting (will cause HTTP 429 errors)
//	❌ Modify rate limiters for performance without understanding consequences
//
// Recovery from Rate Limit Errors:
//
//  1. Immediate: Client receives context.DeadlineExceeded or ErrRateLimit
//  2. Automatic: Rate limiters will automatically respect the 60-second window
//  3. Retry: Wait for Retry-After duration from HTTP 429 response (if manual retry)
//  4. Investigation: Check GitLab instance rate limit configuration if persistent
//
// Monitoring Rate Limit Health:
//
//	1. Enable debug logging to see rate limit Wait() calls:
//	   gitlab.SetLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
//
//	2. Monitor for ErrRateLimit in application error logs
//
//	3. Check response time increases (rate limiting adds latency)
//
//	4. Track HTTP 429 responses in GitLab instance logs (self-managed only)
//
// GitLab API Rate Limit Documentation:
//   - Import/Export Limits: https://docs.gitlab.com/ee/administration/settings/import_export_rate_limits.html
//   - Repository Files Limits: https://docs.gitlab.com/ee/administration/settings/files_api_rate_limits.html
//   - General Rate Limits: https://docs.gitlab.com/security/rate_limits/
//   - API Best Practices: https://docs.gitlab.com/ee/api/rest/#pagination
//
// Implementation: See NewGitlabServiceWithTimeout() for rate limiter initialization.

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
	return NewGitlabServiceWithTimeout(constants.DefaultExportTimeoutMins)
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
		gitlabAPIEndpoint: constants.GitLabAPIEndpoint,
		token:             token,
		exportTimeoutDuration: time.Duration(timeoutMins) * time.Minute,
		// implement rate limiting https://docs.gitlab.com/ee/administration/settings/import_export_rate_limits.html
		rateLimitDownloadAPI: rate.NewLimiter(
			rate.Every(constants.DownloadRateLimitIntervalSeconds*time.Second),
			constants.DownloadRateLimitBurst,
		),
		rateLimitExportAPI: rate.NewLimiter(
			rate.Every(constants.ExportRateLimitIntervalSeconds*time.Second),
			constants.ExportRateLimitBurst,
		),
		rateLimitImportAPI: rate.NewLimiter(
			rate.Every(constants.ImportRateLimitIntervalSeconds*time.Second),
			constants.ImportRateLimitBurst,
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
	if r.gitlabAPIEndpoint != constants.GitLabAPIEndpoint {
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
