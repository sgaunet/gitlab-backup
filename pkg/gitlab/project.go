package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	// ErrExportTimeout is returned when timeout occurs waiting for GitLab to start project export.
	ErrExportTimeout = errors.New("timeout waiting for gitlab to start the export project")
	// ErrRateLimit is returned when rate limit is exceeded.
	ErrRateLimit = errors.New("rate limit error")
)

const (
	// ExportCheckIntervalSeconds defines the interval between export status checks.
	ExportCheckIntervalSeconds = 5
	// MaxExportRetries defines the maximum number of export retries.
	MaxExportRetries = 5
)

// Project represents a Gitlab project
// https://docs.gitlab.com/ee/api/projects.html
// struct fields are not exhaustive - most of them won't be used.
type Project struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Archived     bool   `json:"archived"`
	ExportStatus string `json:"export_status"`
}

// askExport requests GitLab to schedule a project export via the Export API.
//
// This function initiates the export workflow by calling GitLab's
// ProjectImportExport.ScheduleExport endpoint. It does not wait for export
// completion - use waitForExport for polling the export status.
//
// The function checks the HTTP status code to determine if GitLab accepted
// the export request (HTTP 202 Accepted). If GitLab is already processing
// an export for this project, it may return a different status code.
//
// Context cancellation is respected - if ctx is cancelled, the API call
// will be interrupted and return an error.
//
// Parameters:
//   - ctx: Request context for cancellation and timeout control
//   - projectID: GitLab project ID to export
//
// Returns:
//   - bool: true if GitLab accepted the export request (HTTP 202), false otherwise
//   - error: API communication errors or context cancellation
//
// This function is rate-limited by ExportRateLimitBurst (6 requests/minute).
// The rate limiting is enforced by the caller (ExportProject).
//
// GitLab API Reference:
// https://docs.gitlab.com/ee/api/project_import_export.html#schedule-an-export
func (s *Service) askExport(ctx context.Context, projectID int64) (bool, error) {
	resp, err := s.client.ProjectImportExport().ScheduleExport(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("failed to make export request: %w", err)
	}

	// 202 means that gitlab has accepted request
	return resp.StatusCode == http.StatusAccepted, nil
}

// waitForExport polls GitLab's export status until the export completes or fails.
//
// This function implements the polling loop for export workflow. It checks the
// export status every ExportCheckIntervalSeconds (5 seconds) until one of:
//   - Export reaches "finished" status (success)
//   - Context timeout expires (returns ErrExportTimeout)
//   - Context is cancelled (returns context error)
//   - Maximum retries exceeded for "none" status (returns ErrExportTimeout)
//
// The function creates an internal timeout context based on s.exportTimeoutDuration
// (default: 10 minutes) to prevent indefinite waiting. This is in addition to any
// timeout set by the caller.
//
// Export Status Values:
//   - "none": No export in progress (counted against MaxExportRetries)
//   - "finished": Export completed successfully (workflow continues)
//   - "queued", "started", "regeneration_in_progress": In progress (keep polling)
//   - Other statuses: Continue polling
//
// Context Cancellation:
// The function respects both the caller's context and the internal timeout context.
// If either context is cancelled or times out, the function returns immediately
// with an appropriate error message including the project ID.
//
// Parameters:
//   - ctx: Caller's context for cancellation control
//   - projectID: GitLab project ID being exported
//
// Returns:
//   - error: nil on success, ErrExportTimeout on timeout, context error on cancellation
//
// Related Functions:
//   - getStatusExport: Retrieves current export status from GitLab
//   - sleepWithContext: Context-aware sleep between polling attempts
//
// Used by: ExportProject
//
// GitLab API Reference:
// https://docs.gitlab.com/ee/api/project_import_export.html#export-status
func (s *Service) waitForExport(ctx context.Context, projectID int64) error {
	// Create a context with timeout to avoid waiting forever
	timeoutCtx, cancel := context.WithTimeout(ctx, s.exportTimeoutDuration)
	defer cancel()

	nbTries := 0
loop:
	for nbTries < MaxExportRetries {
		exportStatus, err := s.getStatusExport(timeoutCtx, projectID)
		if err != nil {
			return err
		}
		switch exportStatus {
		case "none":
			nbTries++
			log.Warn("no export in progress", "projectID", projectID)
		case "finished":
			break loop
		default:
			log.Info("wait after gitlab to get the archive", "projectID", projectID)
		}

		// Sleep with context awareness
		if err := s.sleepWithContext(timeoutCtx, projectID, ExportCheckIntervalSeconds*time.Second); err != nil {
			return err
		}
	}
	if nbTries == MaxExportRetries {
		return fmt.Errorf("%w %d", ErrExportTimeout, projectID)
	}
	return nil
}

// sleepWithContext implements a context-aware sleep that can be interrupted.
//
// This is a utility function for polling loops that need to respect context
// cancellation. It blocks for the specified duration unless the context is
// cancelled or times out, in which case it returns immediately with an error.
//
// Unlike time.Sleep(), this function does not ignore context cancellation,
// making it suitable for long-running polling operations where graceful
// shutdown is important.
//
// Error Messages:
//   - Timeout: Includes project ID and timeout duration for debugging
//   - Cancellation: Includes project ID and generic cancellation message
//
// Parameters:
//   - ctx: Context to monitor for cancellation/timeout
//   - projectID: Project ID for error messages (not used functionally)
//   - duration: How long to sleep if context remains active
//
// Returns:
//   - error: nil if sleep completed, wrapped context error if cancelled/timeout
//
// Implementation Note:
// Uses select with two cases: context Done channel and time.After timer.
// The context error is wrapped with fmt.Errorf to preserve error chain.
//
// Used by: waitForExport (for polling delay between status checks).
func (s *Service) sleepWithContext(ctx context.Context, projectID int64, duration time.Duration) error {
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("export timeout after %v for project %d: %w",
				s.exportTimeoutDuration, projectID, context.DeadlineExceeded)
		}
		return fmt.Errorf("export cancelled for project %d: %w", projectID, ctx.Err())
	case <-time.After(duration):
		return nil
	}
}

// getStatusExport returns the status of the export.
func (s *Service) getStatusExport(ctx context.Context, projectID int64) (string, error) {
	exportStatus, _, err := s.client.ProjectImportExport().ExportStatus(projectID, gitlab.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to get export status: %w", err)
	}
	return exportStatus.ExportStatus, nil
}

// ExportProject exports the project to the given archive file path.
func (s *Service) ExportProject(ctx context.Context, project *Project, archiveFilePath string) error {
	var gitlabAcceptedRequest bool
	if project.Archived {
		log.Warn("SaveProject", "project name", project.Name, "is archived, skip it")
		return nil
	}
	err := s.rateLimitExportAPI.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRateLimit, err)
	}
	for !gitlabAcceptedRequest {
		gitlabAcceptedRequest, err = s.askExport(ctx, project.ID)
		if err != nil {
			return err
		}
	}
	log.Info("SaveProject (gitlab is creating the archive)", "project name", project.Name)
	err = s.waitForExport(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("failed to export project %s: %w", project.Name, err)
	}
	log.Info("SaveProject (gitlab has created the archive, download is beginning)", "project name", project.Name)
	err = s.downloadProject(ctx, project.ID, archiveFilePath)
	if err != nil {
		return err
	}
	return nil
}

// downloadProject downloads the project and save the archive to the given path.
func (s *Service) downloadProject(ctx context.Context, projectID int64, tmpFilePath string) error {
	err := s.rateLimitDownloadAPI.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRateLimit, err)
	}

	tmpFile := tmpFilePath + ".tmp"

	// Download the export using the official client
	data, _, err := s.client.ProjectImportExport().ExportDownload(projectID, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to download export: %w", err)
	}

	log.Debug("downloadProject", "tmpFile", tmpFile)
	log.Debug("downloadProject", "tmpFilePath", tmpFilePath)
	log.Debug("downloadProject", "projectID", projectID)
	log.Debug("downloadProject", "dataSize", len(data))

	//nolint:gosec,mnd // G304: File creation is intentional for download functionality
	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write temporary file %s: %w", tmpFile, err)
	}

	if err = os.Rename(tmpFile, tmpFilePath); err != nil {
		return fmt.Errorf("failed to rename temporary file %s to %s: %w", tmpFile, tmpFilePath, err)
	}
	return nil
}
