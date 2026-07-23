package restore_test

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/app/restore"
	"github.com/stretchr/testify/assert"
)

func TestConsoleProgressReporter_AllPhases(t *testing.T) {
	// Create logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	reporter := restore.NewConsoleProgressReporter(logger)

	// Test all phase operations - verify no panics
	t.Run("StartPhase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.StartPhase(restore.PhaseValidation)
			reporter.StartPhase(restore.PhaseDownload)
			reporter.StartPhase(restore.PhaseExtraction)
			reporter.StartPhase(restore.PhaseImport)
			reporter.StartPhase(restore.PhaseCleanup)
		})
	})

	t.Run("UpdatePhase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.UpdatePhase(restore.PhaseImport, 5, 10)
			reporter.UpdatePhase(restore.PhaseExtraction, 100, 200)
		})
	})

	t.Run("CompletePhase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.CompletePhase(restore.PhaseValidation)
			reporter.CompletePhase(restore.PhaseDownload)
			reporter.CompletePhase(restore.PhaseExtraction)
			reporter.CompletePhase(restore.PhaseImport)
		})
	})

	t.Run("FailPhase", func(t *testing.T) {
		testErr := errors.New("test error")
		assert.NotPanics(t, func() {
			reporter.FailPhase(restore.PhaseValidation, testErr)
			reporter.FailPhase(restore.PhaseImport, testErr)
		})
	})

	t.Run("SkipPhase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.SkipPhase(restore.PhaseValidation, "overwrite flag set")
			reporter.SkipPhase(restore.PhaseDownload, "local restore")
		})
	})

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// Simulate complete workflow
		assert.NotPanics(t, func() {
			reporter.StartPhase(restore.PhaseValidation)
			reporter.CompletePhase(restore.PhaseValidation)

			reporter.StartPhase(restore.PhaseExtraction)
			reporter.UpdatePhase(restore.PhaseExtraction, 50, 100)
			reporter.CompletePhase(restore.PhaseExtraction)

			reporter.StartPhase(restore.PhaseImport)
			reporter.CompletePhase(restore.PhaseImport)
		})
	})
}

func TestNoOpProgressReporter_NoOutput(t *testing.T) {
	reporter := restore.NewNoOpProgressReporter()

	// All operations should complete without panics or side effects
	t.Run("AllOperationsNoOp", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.StartPhase(restore.PhaseValidation)
			reporter.UpdatePhase(restore.PhaseImport, 1, 10)
			reporter.CompletePhase(restore.PhaseValidation)
			reporter.FailPhase(restore.PhaseImport, errors.New("test error"))
			reporter.SkipPhase(restore.PhaseDownload, "reason")
		})
	})

	t.Run("MultipleCallsSafe", func(t *testing.T) {
		assert.NotPanics(t, func() {
			// Call each method multiple times
			for i := 0; i < 100; i++ {
				reporter.StartPhase(restore.PhaseValidation)
				reporter.UpdatePhase(restore.PhaseImport, i, 100)
				reporter.CompletePhase(restore.PhaseValidation)
				reporter.FailPhase(restore.PhaseImport, errors.New("error"))
				reporter.SkipPhase(restore.PhaseDownload, "skip")
			}
		})
	})
}

// TestConsoleProgressReporter_PhaseMessages verifies the human-readable message
// emitted for each phase. It is the black-box replacement for the previous
// white-box test of the unexported getPhaseStartMessage: the message is asserted
// through StartPhase's logged output captured from a buffer.
func TestConsoleProgressReporter_PhaseMessages(t *testing.T) {
	tests := []struct {
		phase           restore.Phase
		expectedMessage string
	}{
		{restore.PhaseValidation, "Validating project emptiness"},
		{restore.PhaseDownload, "Downloading archive from S3"},
		{restore.PhaseExtraction, "Extracting archive"},
		{restore.PhaseImport, "Importing repository"},
		{restore.PhaseCleanup, "Cleaning up temporary files"},
		{restore.PhaseComplete, "Restore complete"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			reporter := restore.NewConsoleProgressReporter(logger)

			reporter.StartPhase(tt.phase)

			assert.Contains(t, buf.String(), tt.expectedMessage, "Phase message should match expected")
		})
	}

	t.Run("UnknownPhase", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		reporter := restore.NewConsoleProgressReporter(logger)

		reporter.StartPhase(restore.Phase("unknown"))

		assert.Contains(t, buf.String(), "unknown", "Unknown phase should log the phase string")
	})
}

func TestConsoleProgressReporter_Creation(t *testing.T) {
	t.Run("WithLogger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		reporter := restore.NewConsoleProgressReporter(logger)

		assert.NotNil(t, reporter, "Reporter should not be nil")
		assert.NotPanics(t, func() {
			reporter.StartPhase(restore.PhaseValidation)
		})
	})
}

func TestNoOpProgressReporter_Creation(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		reporter := restore.NewNoOpProgressReporter()

		assert.NotNil(t, reporter, "Reporter should not be nil")
		assert.NotPanics(t, func() {
			reporter.StartPhase(restore.PhaseValidation)
		})
	})
}
