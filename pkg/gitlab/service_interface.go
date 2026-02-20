package gitlab

import "golang.org/x/time/rate"

//go:generate go tool github.com/matryer/moq -out mocks/service.go -pkg mocks . GitLabService

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
