package restore

import (
	"errors"
	"log/slog"
	"os"
	"testing"

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

	reporter := NewConsoleProgressReporter(logger)

	// Test all phase operations - verify no panics
	t.Run("StartPhase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.StartPhase(PhaseValidation)
			reporter.StartPhase(PhaseDownload)
			reporter.StartPhase(PhaseExtraction)
			reporter.StartPhase(PhaseImport)
			reporter.StartPhase(PhaseCleanup)
		})
	})

	t.Run("UpdatePhase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.UpdatePhase(PhaseImport, 5, 10)
			reporter.UpdatePhase(PhaseExtraction, 100, 200)
		})
	})

	t.Run("CompletePhase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.CompletePhase(PhaseValidation)
			reporter.CompletePhase(PhaseDownload)
			reporter.CompletePhase(PhaseExtraction)
			reporter.CompletePhase(PhaseImport)
		})
	})

	t.Run("FailPhase", func(t *testing.T) {
		testErr := errors.New("test error")
		assert.NotPanics(t, func() {
			reporter.FailPhase(PhaseValidation, testErr)
			reporter.FailPhase(PhaseImport, testErr)
		})
	})

	t.Run("SkipPhase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.SkipPhase(PhaseValidation, "overwrite flag set")
			reporter.SkipPhase(PhaseDownload, "local restore")
		})
	})

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// Simulate complete workflow
		assert.NotPanics(t, func() {
			reporter.StartPhase(PhaseValidation)
			reporter.CompletePhase(PhaseValidation)

			reporter.StartPhase(PhaseExtraction)
			reporter.UpdatePhase(PhaseExtraction, 50, 100)
			reporter.CompletePhase(PhaseExtraction)

			reporter.StartPhase(PhaseImport)
			reporter.CompletePhase(PhaseImport)
		})
	})
}

func TestNoOpProgressReporter_NoOutput(t *testing.T) {
	reporter := NewNoOpProgressReporter()

	// All operations should complete without panics or side effects
	t.Run("AllOperationsNoOp", func(t *testing.T) {
		assert.NotPanics(t, func() {
			reporter.StartPhase(PhaseValidation)
			reporter.UpdatePhase(PhaseImport, 1, 10)
			reporter.CompletePhase(PhaseValidation)
			reporter.FailPhase(PhaseImport, errors.New("test error"))
			reporter.SkipPhase(PhaseDownload, "reason")
		})
	})

	t.Run("MultipleCallsSafe", func(t *testing.T) {
		assert.NotPanics(t, func() {
			// Call each method multiple times
			for i := 0; i < 100; i++ {
				reporter.StartPhase(PhaseValidation)
				reporter.UpdatePhase(PhaseImport, i, 100)
				reporter.CompletePhase(PhaseValidation)
				reporter.FailPhase(PhaseImport, errors.New("error"))
				reporter.SkipPhase(PhaseDownload, "skip")
			}
		})
	})
}

func TestGetPhaseStartMessage_AllPhases(t *testing.T) {
	tests := []struct {
		phase           Phase
		expectedMessage string
	}{
		{PhaseValidation, "Validating project emptiness"},
		{PhaseDownload, "Downloading archive from S3"},
		{PhaseExtraction, "Extracting archive"},
		{PhaseImport, "Importing repository"},
		{PhaseCleanup, "Cleaning up temporary files"},
		{PhaseComplete, "Restore complete"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			message := getPhaseStartMessage(tt.phase)
			assert.Equal(t, tt.expectedMessage, message, "Phase message should match expected")
		})
	}

	t.Run("UnknownPhase", func(t *testing.T) {
		unknownPhase := Phase("unknown")
		message := getPhaseStartMessage(unknownPhase)
		assert.Equal(t, "unknown", message, "Unknown phase should return phase string")
	})
}

func TestConsoleProgressReporter_Creation(t *testing.T) {
	t.Run("WithLogger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		reporter := NewConsoleProgressReporter(logger)

		assert.NotNil(t, reporter, "Reporter should not be nil")
		assert.NotPanics(t, func() {
			reporter.StartPhase(PhaseValidation)
		})
	})
}

func TestNoOpProgressReporter_Creation(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		reporter := NewNoOpProgressReporter()

		assert.NotNil(t, reporter, "Reporter should not be nil")
		assert.NotPanics(t, func() {
			reporter.StartPhase(PhaseValidation)
		})
	})
}
