package gitlab

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
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
	// Based on GitLab repository files API limit: 5 requests per minute per user
	DownloadRateLimitIntervalSeconds = 60
	// DownloadRateLimitBurst defines the burst limit for download API calls.
	DownloadRateLimitBurst = 5
	// ExportRateLimitIntervalSeconds defines the rate limit interval for export API calls.
	// Based on GitLab project import/export API limit: 6 requests per minute per user
	ExportRateLimitIntervalSeconds = 60
	// ExportRateLimitBurst defines the burst limit for export API calls.
	ExportRateLimitBurst = 6
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
	client               GitLabClient
	gitlabAPIEndpoint    string
	token                string
	rateLimitDownloadAPI *rate.Limiter
	rateLimitExportAPI   *rate.Limiter
}

func init() {
	log = slog.New(slog.DiscardHandler)
}

// NewGitlabService returns a new Service.
func NewGitlabService() *Service {
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
		// implement rate limiting https://docs.gitlab.com/ee/administration/settings/import_export_rate_limits.html
		rateLimitDownloadAPI: rate.NewLimiter(
			rate.Every(DownloadRateLimitIntervalSeconds*time.Second),
			DownloadRateLimitBurst,
		),
		rateLimitExportAPI: rate.NewLimiter(
			rate.Every(ExportRateLimitIntervalSeconds*time.Second),
			ExportRateLimitBurst,
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
func (r *Service) GetGroup(ctx context.Context, groupID int) (Group, error) {
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
func (r *Service) GetProject(ctx context.Context, projectID int) (Project, error) {
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
