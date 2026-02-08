package restore

import (
	"fmt"
	"log/slog"
)

// ProgressReporter provides user-visible progress reporting for restore operations.
type ProgressReporter interface {
	// StartPhase signals the beginning of a phase.
	StartPhase(phase Phase)

	// UpdatePhase provides mid-phase progress (e.g., "5/10 issues restored").
	UpdatePhase(phase Phase, current, total int)

	// CompletePhase signals successful phase completion.
	CompletePhase(phase Phase)

	// FailPhase signals phase failure.
	FailPhase(phase Phase, err error)

	// SkipPhase signals that a phase was skipped.
	SkipPhase(phase Phase, reason string)
}

// ConsoleProgressReporter implements ProgressReporter with console output.
type ConsoleProgressReporter struct {
	logger *slog.Logger
}

// NewConsoleProgressReporter creates a new console progress reporter.
func NewConsoleProgressReporter(logger *slog.Logger) *ConsoleProgressReporter {
	return &ConsoleProgressReporter{
		logger: logger,
	}
}

// StartPhase logs the start of a restore phase.
func (r *ConsoleProgressReporter) StartPhase(phase Phase) {
	message := getPhaseStartMessage(phase)
	r.logger.Info(fmt.Sprintf("[RESTORE] %s...", message))
}

// UpdatePhase logs mid-phase progress.
func (r *ConsoleProgressReporter) UpdatePhase(phase Phase, current, total int) {
	message := getPhaseStartMessage(phase)
	r.logger.Info(fmt.Sprintf("[RESTORE] %s... (%d/%d)", message, current, total))
}

// CompletePhase logs successful phase completion.
func (r *ConsoleProgressReporter) CompletePhase(phase Phase) {
	message := getPhaseStartMessage(phase)
	r.logger.Info(fmt.Sprintf("[RESTORE] %s ✓", message))
}

// FailPhase logs phase failure.
func (r *ConsoleProgressReporter) FailPhase(phase Phase, err error) {
	message := getPhaseStartMessage(phase)
	r.logger.Error(fmt.Sprintf("[RESTORE] %s ✗ %v", message, err))
}

// SkipPhase logs that a phase was skipped.
func (r *ConsoleProgressReporter) SkipPhase(phase Phase, reason string) {
	message := getPhaseStartMessage(phase)
	r.logger.Info(fmt.Sprintf("[RESTORE] %s (skipped: %s)", message, reason))
}

// getPhaseStartMessage returns a human-readable message for each phase.
func getPhaseStartMessage(phase Phase) string {
	switch phase {
	case PhaseValidation:
		return "Validating project emptiness"
	case PhaseDownload:
		return "Downloading archive from S3"
	case PhaseExtraction:
		return "Extracting archive"
	case PhaseImport:
		return "Importing repository"
	case PhaseCleanup:
		return "Cleaning up temporary files"
	case PhaseComplete:
		return "Restore complete"
	default:
		return string(phase)
	}
}

// NoOpProgressReporter is a progress reporter that does nothing.
// Useful for testing or when progress reporting is not desired.
type NoOpProgressReporter struct{}

// NewNoOpProgressReporter creates a new no-op progress reporter.
func NewNoOpProgressReporter() *NoOpProgressReporter {
	return &NoOpProgressReporter{}
}

// StartPhase does nothing.
func (r *NoOpProgressReporter) StartPhase(_ Phase) {}

// UpdatePhase does nothing.
func (r *NoOpProgressReporter) UpdatePhase(_ Phase, _, _ int) {}

// CompletePhase does nothing.
func (r *NoOpProgressReporter) CompletePhase(_ Phase) {}

// FailPhase does nothing.
func (r *NoOpProgressReporter) FailPhase(_ Phase, _ error) {}

// SkipPhase does nothing.
func (r *NoOpProgressReporter) SkipPhase(_ Phase, _ string) {}
