package restore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResult_AddError(t *testing.T) {
	t.Run("SingleError", func(t *testing.T) {
		result := &Result{
			Success: true,
			Errors:  []Error{},
		}

		// Add an error
		result.addError(PhaseValidation, "Validator", "validation failed")

		// Verify error was added
		assert.Len(t, result.Errors, 1, "Should have one error")
		assert.Equal(t, PhaseValidation, result.Errors[0].Phase)
		assert.Equal(t, "Validator", result.Errors[0].Component)
		assert.Equal(t, "validation failed", result.Errors[0].Message)
		assert.True(t, result.Errors[0].Fatal, "Error should be fatal")
		assert.False(t, result.Errors[0].Timestamp.IsZero(), "Timestamp should be set")
	})

	t.Run("MultipleErrors", func(t *testing.T) {
		result := &Result{
			Success: true,
			Errors:  []Error{},
		}

		// Add multiple errors
		result.addError(PhaseValidation, "Validator", "validation failed")
		result.addError(PhaseImport, "GitLabImport", "import failed")

		// Verify both errors were added
		assert.Len(t, result.Errors, 2, "Should have two errors")
		assert.Equal(t, PhaseValidation, result.Errors[0].Phase)
		assert.Equal(t, PhaseImport, result.Errors[1].Phase)
	})

	t.Run("TimestampSet", func(t *testing.T) {
		result := &Result{
			Errors: []Error{},
		}

		beforeAdd := time.Now()
		time.Sleep(10 * time.Millisecond)

		result.addError(PhaseExtraction, "ArchiveExtractor", "extraction failed")

		time.Sleep(10 * time.Millisecond)
		afterAdd := time.Now()

		// Verify timestamp is within expected range
		assert.True(t, result.Errors[0].Timestamp.After(beforeAdd), "Timestamp should be after start time")
		assert.True(t, result.Errors[0].Timestamp.Before(afterAdd), "Timestamp should be before end time")
	})
}

func TestResult_AddWarning(t *testing.T) {
	t.Run("SingleWarning", func(t *testing.T) {
		result := &Result{
			Warnings: []string{},
		}

		// Add a warning
		result.addWarning("Failed to cleanup temp dir")

		// Verify warning was added
		assert.Len(t, result.Warnings, 1, "Should have one warning")
		assert.Equal(t, "Failed to cleanup temp dir", result.Warnings[0])
	})

	t.Run("MultipleWarnings", func(t *testing.T) {
		result := &Result{
			Warnings: []string{},
		}

		// Add multiple warnings
		result.addWarning("Warning 1")
		result.addWarning("Warning 2")
		result.addWarning("Warning 3")

		// Verify all warnings were added
		assert.Len(t, result.Warnings, 3, "Should have three warnings")
		assert.Equal(t, "Warning 1", result.Warnings[0])
		assert.Equal(t, "Warning 2", result.Warnings[1])
		assert.Equal(t, "Warning 3", result.Warnings[2])
	})
}

func TestResult_HasFatalErrors(t *testing.T) {
	t.Run("NoErrors", func(t *testing.T) {
		result := &Result{
			Errors: []Error{},
		}

		// Verify no fatal errors
		assert.False(t, result.hasFatalErrors(), "Should have no fatal errors")
	})

	t.Run("WithFatalErrors", func(t *testing.T) {
		result := &Result{
			Errors: []Error{
				{
					Phase:     PhaseValidation,
					Component: "Validator",
					Message:   "validation failed",
					Fatal:     true,
					Timestamp: time.Now(),
				},
			},
		}

		// Verify has fatal errors
		assert.True(t, result.hasFatalErrors(), "Should have fatal errors")
	})

	t.Run("OnlyWarnings", func(t *testing.T) {
		result := &Result{
			Errors:   []Error{},
			Warnings: []string{"Warning 1", "Warning 2"},
		}

		// Verify no fatal errors (warnings are not errors)
		assert.False(t, result.hasFatalErrors(), "Should have no fatal errors with only warnings")
	})

	t.Run("MultipleFatalErrors", func(t *testing.T) {
		result := &Result{
			Errors: []Error{
				{
					Phase:     PhaseValidation,
					Component: "Validator",
					Message:   "validation failed",
					Fatal:     true,
					Timestamp: time.Now(),
				},
				{
					Phase:     PhaseImport,
					Component: "GitLabImport",
					Message:   "import failed",
					Fatal:     true,
					Timestamp: time.Now(),
				},
			},
		}

		// Verify has fatal errors
		assert.True(t, result.hasFatalErrors(), "Should have fatal errors")
	})
}

func TestResult_ErrorsAndWarnings(t *testing.T) {
	t.Run("CombinedScenario", func(t *testing.T) {
		result := &Result{
			Success:  true,
			Errors:   []Error{},
			Warnings: []string{},
		}

		// Add errors and warnings
		result.addError(PhaseValidation, "Validator", "validation failed")
		result.addWarning("Cleanup warning 1")
		result.addWarning("Cleanup warning 2")

		// Verify both were added
		assert.Len(t, result.Errors, 1, "Should have one error")
		assert.Len(t, result.Warnings, 2, "Should have two warnings")
		assert.True(t, result.hasFatalErrors(), "Should have fatal errors")
	})

	t.Run("SuccessFalseWithFatalErrors", func(t *testing.T) {
		result := &Result{
			Success: true,
			Errors:  []Error{},
		}

		// Add fatal error
		result.addError(PhaseImport, "GitLabImport", "import failed")

		// If we were to set Success based on hasFatalErrors (as done in Restore method)
		result.Success = !result.hasFatalErrors()

		// Verify Success is false when fatal errors exist
		assert.False(t, result.Success, "Success should be false when fatal errors exist")
	})
}
