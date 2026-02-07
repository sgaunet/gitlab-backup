package gitlab

import (
	"context"
	"fmt"
	"io"
	"time"

	"golang.org/x/time/rate"
	gitlabapi "gitlab.com/gitlab-org/api/client-go"
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
func (s *ImportService) ImportProject(ctx context.Context, archive io.Reader, namespace string, projectPath string) (*gitlabapi.ImportStatus, error) {
	// Wait for rate limit
	if err := s.rateLimiterImport.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Initiate import
	importStatus, _, err := s.importExportService.ImportFromFile(
		archive,
		&gitlabapi.ImportFileOptions{
			Namespace: &namespace,
			Path:      &projectPath,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate import: %w", err)
	}

	// Wait for import to complete (10 minute default timeout)
	finalStatus, err := s.WaitForImport(ctx, importStatus.ID, 10*time.Minute)
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
func (s *ImportService) WaitForImport(ctx context.Context, projectID int64, timeout time.Duration) (*gitlabapi.ImportStatus, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		// Wait for rate limit
		if err := s.rateLimiterImport.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}

		// Check import status
		status, _, err := s.importExportService.ImportStatus(projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get import status: %w", err)
		}

		// Check terminal states
		switch status.ImportStatus {
		case "finished":
			return status, nil
		case "failed":
			return nil, fmt.Errorf("import failed: %s", status.ImportError)
		case "scheduled", "started":
			// Continue polling
		default:
			return nil, fmt.Errorf("unexpected import status: %s", status.ImportStatus)
		}

		// Wait for next poll or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}
