package gitlab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/constants"
	"golang.org/x/time/rate"
	gitlabapi "gitlab.com/gitlab-org/api/client-go"
)

var (
	// ErrImportFailed is returned when GitLab import fails.
	ErrImportFailed = errors.New("import failed")
	// ErrUnexpectedImportStatus is returned when import reaches an unexpected status.
	ErrUnexpectedImportStatus = errors.New("unexpected import status")
	// ErrImportTimeout is returned when the local import polling deadline is reached
	// before GitLab reports the import as finished. The import may still be running
	// server-side; users should check the GitLab web UI before retrying.
	ErrImportTimeout = errors.New(
		"import timeout — operation may still be in progress on GitLab, check the web UI",
	)
)

// ImportService provides GitLab project import functionality.
type ImportService struct {
	importExportService ProjectImportExportService
	rateLimiterImport   *rate.Limiter
	timeout             time.Duration
}

// resolveImportTimeout returns the configured timeout, or the default if zero/negative.
func resolveImportTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return constants.DefaultImportTimeoutMins * time.Minute
	}
	return timeout
}

// NewImportService creates a new import service instance.
// A non-positive timeout falls back to constants.DefaultImportTimeoutMins.
func NewImportService(
	importExportService ProjectImportExportService,
	timeout time.Duration,
) *ImportService {
	return &ImportService{
		importExportService: importExportService,
		rateLimiterImport: rate.NewLimiter(
			rate.Every(constants.ImportRateLimitIntervalSeconds*time.Second),
			constants.ImportRateLimitBurst,
		),
		timeout: resolveImportTimeout(timeout),
	}
}

// NewImportServiceWithRateLimiters creates an import service with custom rate limiters.
// A non-positive timeout falls back to constants.DefaultImportTimeoutMins.
func NewImportServiceWithRateLimiters(
	importExportService ProjectImportExportService,
	rateLimiterImport *rate.Limiter,
	timeout time.Duration,
) *ImportService {
	return &ImportService{
		importExportService: importExportService,
		rateLimiterImport:   rateLimiterImport,
		timeout:             resolveImportTimeout(timeout),
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
		ctx,
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

	// Wait for import to complete (timeout configurable via Config.ImportTimeoutMins).
	finalStatus, err := s.WaitForImport(ctx, importStatus.ID, s.timeout)
	if err != nil {
		// Pass timeout errors through unwrapped to preserve the sentinel-based
		// detection in callers and avoid double-prefixing the message.
		if errors.Is(err, ErrImportTimeout) {
			return nil, err
		}
		return nil, fmt.Errorf("import did not complete successfully: %w", err)
	}

	return finalStatus, nil
}

// importTimeoutClassifier classifies poll-loop failures as timeouts or genuine errors.
// We need it because golang.org/x/time/rate.Wait returns "would exceed context deadline"
// proactively — before ctx.Err() is set — when remaining ctx time is less than the
// required wait. The grace-period check catches that case.
type importTimeoutClassifier struct {
	deadline    time.Time
	gracePeriod time.Duration
	timeout     time.Duration
	projectID   int64
}

func (c importTimeoutClassifier) isTimeout(ctx context.Context, err error) bool {
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return true
	}
	return time.Until(c.deadline) <= c.gracePeriod
}

func (c importTimeoutClassifier) timeoutErr() error {
	return fmt.Errorf("%w after %s (project ID %d)", ErrImportTimeout, c.timeout, c.projectID)
}

// WaitForImport polls the import status until it reaches a terminal state (finished or failed).
// It respects the context deadline and rate limiting.
//
// Returns the final ImportStatus when import reaches "finished" state.
// Returns error if import fails, times out, or API errors occur.
// Timeout-flavored failures are mapped to ErrImportTimeout so callers can
// distinguish them from genuine rate-limit / status-check errors.
func (s *ImportService) WaitForImport(
	ctx context.Context,
	projectID int64,
	timeout time.Duration,
) (*gitlabapi.ImportStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	deadline, _ := ctx.Deadline()
	tc := importTimeoutClassifier{
		deadline:    deadline,
		gracePeriod: constants.ImportRateLimitGracePeriodSeconds * time.Second,
		timeout:     timeout,
		projectID:   projectID,
	}

	ticker := time.NewTicker(constants.ImportPollSeconds * time.Second)
	defer ticker.Stop()

	for {
		status, err := s.pollImportOnce(ctx, projectID, tc)
		if err != nil {
			return nil, err
		}
		if status != nil {
			return status, nil
		}

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, tc.timeoutErr()
			}
			return nil, fmt.Errorf("import cancelled: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// pollImportOnce performs a single poll iteration: rate-limit wait, status fetch,
// terminal-state check. Returns (status, nil) when the import has finished,
// (nil, err) when it has failed or the loop should abort, or (nil, nil) when
// polling should continue.
func (s *ImportService) pollImportOnce(
	ctx context.Context,
	projectID int64,
	tc importTimeoutClassifier,
) (*gitlabapi.ImportStatus, error) {
	if err := s.rateLimiterImport.Wait(ctx); err != nil {
		if tc.isTimeout(ctx, err) {
			return nil, tc.timeoutErr()
		}
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	status, _, err := s.importExportService.ImportStatus(ctx, projectID, gitlabapi.WithContext(ctx))
	if err != nil {
		if tc.isTimeout(ctx, err) {
			return nil, tc.timeoutErr()
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("import status check cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to get import status: %w", err)
	}

	terminal, err := checkImportStatus(status)
	if err != nil {
		return nil, err
	}
	if terminal {
		return status, nil
	}
	return nil, nil //nolint:nilnil // (nil, nil) signals "continue polling" to the caller
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
