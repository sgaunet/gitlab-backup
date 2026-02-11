package gitlab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/time/rate"
	gitlabapi "gitlab.com/gitlab-org/api/client-go"
)

const (
	importTimeoutMinutes = 10
	importPollSeconds    = 5
)

var (
	// ErrImportFailed is returned when GitLab import fails.
	ErrImportFailed = errors.New("import failed")
	// ErrUnexpectedImportStatus is returned when import reaches an unexpected status.
	ErrUnexpectedImportStatus = errors.New("unexpected import status")
)

// ImportService provides GitLab project import functionality.
type ImportService struct {
	importExportService ProjectImportExportService
	rateLimiterImport   *rate.Limiter
}

// NewImportService creates a new import service instance.
func NewImportService(importExportService ProjectImportExportService) *ImportService {
	return &ImportService{
		importExportService: importExportService,
		rateLimiterImport: rate.NewLimiter(
			rate.Every(ImportRateLimitIntervalSeconds*time.Second),
			ImportRateLimitBurst,
		),
	}
}

// NewImportServiceWithRateLimiters creates an import service with custom rate limiters.
func NewImportServiceWithRateLimiters(
	importExportService ProjectImportExportService,
	rateLimiterImport *rate.Limiter,
) *ImportService {
	return &ImportService{
		importExportService: importExportService,
		rateLimiterImport:   rateLimiterImport,
	}
}

// ImportProject initiates a GitLab project import and waits for completion.
// It respects rate limiting and polls the import status until finished or failed.
//
// Returns the final ImportStatus on success.
// Returns error if import initiation fails, import status becomes "failed", or timeout occurs.
func (s *ImportService) ImportProject(
	ctx context.Context,
	archive io.Reader,
	namespace string,
	projectPath string,
) (*gitlabapi.ImportStatus, error) {
	// Wait for rate limit
	if err := s.rateLimiterImport.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Initiate import (with context support)
	importStatus, _, err := s.importExportService.ImportFromFile(
		archive,
		&gitlabapi.ImportFileOptions{
			Namespace: &namespace,
			Path:      &projectPath,
		},
		gitlabapi.WithContext(ctx),
	)
	if err != nil {
		// Check if cancellation caused the error
		if ctx.Err() != nil {
			return nil, fmt.Errorf("import initiation cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to initiate import: %w", err)
	}

	// Wait for import to complete (10 minute default timeout)
	finalStatus, err := s.WaitForImport(ctx, importStatus.ID, importTimeoutMinutes*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("import did not complete successfully: %w", err)
	}

	return finalStatus, nil
}

// WaitForImport polls the import status until it reaches a terminal state (finished or failed).
// It respects the context deadline and rate limiting.
//
// Returns the final ImportStatus when import reaches "finished" state.
// Returns error if import fails, times out, or API errors occur.
func (s *ImportService) WaitForImport(
	ctx context.Context,
	projectID int64,
	timeout time.Duration,
) (*gitlabapi.ImportStatus, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll every 5 seconds
	ticker := time.NewTicker(importPollSeconds * time.Second)
	defer ticker.Stop()

	for {
		// Wait for rate limit
		if err := s.rateLimiterImport.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}

		// Check import status (with context support)
		status, _, err := s.importExportService.ImportStatus(projectID, gitlabapi.WithContext(ctx))
		if err != nil {
			// Check if cancellation caused the error
			if ctx.Err() != nil {
				return nil, fmt.Errorf("import status check cancelled: %w", ctx.Err())
			}
			return nil, fmt.Errorf("failed to get import status: %w", err)
		}

		// Check terminal states
		terminal, err := checkImportStatus(status)
		if err != nil {
			return nil, err
		}
		if terminal {
			return status, nil
		}

		// Wait for next poll or context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("import cancelled or timed out: %w", ctx.Err())
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}

// checkImportStatus evaluates the import status and determines if it's terminal.
// Returns true if import is finished successfully, false if still in progress.
// Returns error if import failed or reached unexpected status.
func checkImportStatus(status *gitlabapi.ImportStatus) (bool, error) {
	switch status.ImportStatus {
	case "finished":
		return true, nil
	case "failed":
		return false, fmt.Errorf("%w: %s", ErrImportFailed, status.ImportError)
	case "scheduled", "started":
		return false, nil
	default:
		return false, fmt.Errorf("%w: %s", ErrUnexpectedImportStatus, status.ImportStatus)
	}
}
