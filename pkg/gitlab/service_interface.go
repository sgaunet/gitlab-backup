package gitlab

import (
	"context"

	"golang.org/x/time/rate"
)

//go:generate go tool github.com/matryer/moq -out mocks/service.go -pkg mocks . GitLabService
//go:generate go tool github.com/matryer/moq -out mocks/backup_service.go -pkg mocks . BackupService

// BackupService is the GitLab surface the backup application layer (pkg/app)
// depends on. It exists so the application can be driven with a mock in
// black-box tests instead of a live GitLab connection. *Service satisfies it.
type BackupService interface {
	// SetToken sets the GitLab API token used for authentication.
	SetToken(token string)
	// SetGitlabEndpoint sets the GitLab API endpoint.
	SetGitlabEndpoint(endpoint string)
	// GetProject returns the project identified by projectID.
	GetProject(ctx context.Context, projectID int64) (Project, error)
	// GetProjectsOfGroup returns every non-archived project of the group and its subgroups.
	GetProjectsOfGroup(ctx context.Context, groupID int64) ([]Project, error)
	// ExportProject exports project to archiveFilePath.
	ExportProject(ctx context.Context, project *Project, archiveFilePath string) error
}

// Compile-time guarantee that *Service satisfies BackupService.
var _ BackupService = (*Service)(nil)

// GitLabService defines the GitLab service operations needed for restore workflows.
// This interface enables testing of the restore orchestrator without requiring a
// real GitLab API connection.
//
// The interface exposes only the minimal surface area required by the restore
// orchestrator, following the Interface Segregation Principle.
//
//nolint:revive // Service interface naming is intentionally explicit
type GitLabService interface {
	// Client returns the underlying GitLab client for API operations.
	// The client provides access to all GitLab API services (Projects, Commits,
	// Issues, Labels, ProjectImportExport, etc.) through their respective
	// service interfaces.
	Client() GitLabClient

	// RateLimitImportAPI returns the import API rate limiter.
	// This limiter enforces GitLab's 6 requests/minute limit for import operations
	// to prevent API throttling and potential account restrictions.
	RateLimitImportAPI() *rate.Limiter
}
