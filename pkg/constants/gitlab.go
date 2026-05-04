package constants

// GitLab API Endpoint.
const (
	// GitLabAPIEndpoint is the default GitLab API endpoint for gitlab.com.
	// For self-managed GitLab instances, override via SetGitlabEndpoint().
	GitLabAPIEndpoint = "https://gitlab.com/api/v4"

	// GitLabBaseURL is the default GitLab base URL without API version.
	GitLabBaseURL = "https://gitlab.com"
)

// GitLab API Rate Limits
//
// These limits are based on GitLab's documented API rate limits per user:
// - Repository Files API: 5 requests/minute (for downloads)
// - Project Import/Export API: 6 requests/minute (for exports and imports)
//
// ⚠️  WARNING: DO NOT increase these values above GitLab's documented limits.
// Exceeding limits results in HTTP 429 errors and potential account restrictions.
//
// References:
//   - Import/Export Limits: https://docs.gitlab.com/ee/administration/settings/import_export_rate_limits.html
//   - Repository Files Limits: https://docs.gitlab.com/ee/administration/settings/files_api_rate_limits.html
//   - General Rate Limits: https://docs.gitlab.com/security/rate_limits/
const (
	// DownloadRateLimitIntervalSeconds is the time window for download rate limiting (60 seconds = 1 minute).
	DownloadRateLimitIntervalSeconds = 60

	// DownloadRateLimitBurst is the maximum number of download requests allowed per interval.
	// GitLab repository files API limit: 5 requests per minute per user.
	DownloadRateLimitBurst = 5

	// ExportRateLimitIntervalSeconds is the time window for export rate limiting (60 seconds = 1 minute).
	ExportRateLimitIntervalSeconds = 60

	// ExportRateLimitBurst is the maximum number of export requests allowed per interval.
	// GitLab project import/export API limit: 6 requests per minute per user.
	ExportRateLimitBurst = 6

	// ImportRateLimitIntervalSeconds is the time window for import rate limiting (60 seconds = 1 minute).
	ImportRateLimitIntervalSeconds = 60

	// ImportRateLimitBurst is the maximum number of import requests allowed per interval.
	// GitLab project import/export API limit: 6 requests per minute per user.
	ImportRateLimitBurst = 6
)

// Export Operation Constants
//
// These control the behavior of project export polling and retry logic.
const (
	// ExportCheckIntervalSeconds is the delay between export status checks when polling GitLab.
	// Lower values provide more responsive feedback but increase API load.
	// Default: 5 seconds (balanced responsiveness/load).
	ExportCheckIntervalSeconds = 5

	// MaxExportRetries is the maximum number of consecutive "none" status responses
	// before giving up on export. Prevents infinite loops for stale exports.
	// Default: 5 retries = 25 seconds of "none" responses before timeout.
	MaxExportRetries = 5

	// DefaultExportTimeoutMins is the default timeout for export operations in minutes.
	// Large projects may take longer. This can be overridden via configuration.
	// Default: 1440 minutes (24 hours).
	DefaultExportTimeoutMins = 1440
)

// API Retry Constants
//
// These control retry behavior for transient GitLab API failures (HTTP 5xx, 429, network errors).
// Retry uses exponential backoff with jitter to avoid thundering herd effects.
const (
	// RetryMaxAttempts is the number of retry attempts after an initial failure.
	// Total attempts = 1 (initial) + RetryMaxAttempts (retries) = 4.
	RetryMaxAttempts = 3

	// RetryBaseDelayMs is the base delay in milliseconds for exponential backoff.
	// Delays: 1s, 2s, 4s (before jitter).
	RetryBaseDelayMs = 1000

	// RetryMaxDelayMs is the maximum delay in milliseconds between retries.
	// Caps the exponential growth to prevent excessive waits.
	RetryMaxDelayMs = 30000
)

// Import Operation Constants
//
// These control the behavior of project import polling logic.
const (
	// DefaultImportTimeoutMins is the default maximum time to wait for an import
	// operation to complete. Large projects (>1GB) routinely need longer than the
	// historical 10-minute default. Configurable via importTimeoutMins / IMPORT_TIMEOUT_MIN.
	// Default: 60 minutes.
	DefaultImportTimeoutMins = 60

	// ImportPollSeconds is the interval between import status checks when polling GitLab.
	// Lower values provide more responsive feedback but increase API load.
	// Default: 5 seconds (matches export polling interval).
	ImportPollSeconds = 5

	// ImportRateLimitGracePeriodSeconds is the slack used inside the import polling
	// loop to discriminate a real rate-limit failure from a deadline-exceeded
	// situation. golang.org/x/time/rate.Wait returns "would exceed context deadline"
	// proactively, before ctx.Err() is set; if the import context's remaining time
	// is within this window, we treat the failure as a timeout instead.
	ImportRateLimitGracePeriodSeconds = 15
)
