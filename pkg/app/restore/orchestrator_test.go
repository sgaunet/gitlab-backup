package restore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPhaseConstants tests that phase constants are correctly defined.
func TestPhaseConstants(t *testing.T) {
	phases := []Phase{
		PhaseValidation,
		PhaseDownload,
		PhaseExtraction,
		PhaseImport,
		PhaseCleanup,
		PhaseComplete,
	}

	// Verify all phases are unique
	phaseMap := make(map[Phase]bool)
	for _, phase := range phases {
		assert.False(t, phaseMap[phase], "Phase %s should be unique", phase)
		phaseMap[phase] = true
		assert.NotEmpty(t, string(phase), "Phase should have a string value")
	}

	assert.Len(t, phaseMap, 6, "Should have 6 unique phases")
}

// TestErrorStructure tests the Error type.
func TestErrorStructure(t *testing.T) {
	now := time.Now()
	err := Error{
		Phase:     PhaseImport,
		Component: "GitLabImport",
		Message:   "import failed",
		Fatal:     true,
		Timestamp: now,
	}

	assert.Equal(t, PhaseImport, err.Phase)
	assert.Equal(t, "GitLabImport", err.Component)
	assert.Equal(t, "import failed", err.Message)
	assert.True(t, err.Fatal)
	assert.Equal(t, now, err.Timestamp)
}

// TestMetricsStructure tests the Metrics type.
func TestMetricsStructure(t *testing.T) {
	metrics := Metrics{
		BytesDownloaded: 1024,
		BytesExtracted:  2048,
		DurationSeconds: 60,
	}

	assert.Equal(t, int64(1024), metrics.BytesDownloaded)
	assert.Equal(t, int64(2048), metrics.BytesExtracted)
	assert.Equal(t, int64(60), metrics.DurationSeconds)
}

// TestEmptinessChecks_IsEmpty tests the IsEmpty method.
func TestEmptinessChecks_IsEmpty(t *testing.T) {
	t.Run("EmptyProject", func(t *testing.T) {
		checks := &EmptinessChecks{
			HasCommits: false,
			HasIssues:  false,
			HasLabels:  false,
		}
		assert.True(t, checks.IsEmpty())
	})

	t.Run("ProjectWithCommits", func(t *testing.T) {
		checks := &EmptinessChecks{
			HasCommits:  true,
			HasIssues:   false,
			HasLabels:   false,
			CommitCount: 5,
		}
		assert.False(t, checks.IsEmpty())
	})

	t.Run("ProjectWithMultipleItems", func(t *testing.T) {
		checks := &EmptinessChecks{
			HasCommits:  true,
			HasIssues:   true,
			HasLabels:   true,
			CommitCount: 10,
			IssueCount:  3,
			LabelCount:  2,
		}
		assert.False(t, checks.IsEmpty())
	})
}
