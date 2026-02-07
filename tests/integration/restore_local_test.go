package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/app/restore"
	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRestoreFromLocalArchive tests the complete restore workflow from a local archive.
// This integration test validates:
// - Archive extraction
// - Project validation
// - Repository import (GitLab handles labels and issues internally)
//
// Prerequisites:
// - GitLab instance (testcontainers or manual)
// - Empty target project
// - Valid backup archive with project.tar.gz
func TestRestoreFromLocalArchive(t *testing.T) {
	// Skip in short mode as this requires GitLab instance
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test will be implemented after the core restore functionality
	// It requires testcontainers with GitLab CE or a test GitLab instance

	t.Run("RestoreCompleteArchive", func(t *testing.T) {
		// Setup
		ctx := context.Background()

		// Create test configuration
		cfg := &config.Config{
			GitlabURI:   os.Getenv("TEST_GITLAB_URI"),
			GitlabToken: os.Getenv("TEST_GITLAB_TOKEN"),
			TmpDir:      t.TempDir(),
		}

		if cfg.GitlabURI == "" || cfg.GitlabToken == "" {
			t.Skip("TEST_GITLAB_URI and TEST_GITLAB_TOKEN must be set for integration tests")
		}

		// Create test archive path
		testArchivePath := filepath.Join("testdata", "test-project-backup.tar.gz")
		if _, err := os.Stat(testArchivePath); os.IsNotExist(err) {
			t.Skip("Test archive not found: " + testArchivePath)
		}

		// Initialize GitLab service
		svc := gitlab.NewGitlabServiceWithTimeout(10)
		require.NotNil(t, svc, "GitLab service should be initialized")

		svc.SetToken(cfg.GitlabToken)
		svc.SetGitlabEndpoint(cfg.GitlabURI)

		// Create mock storage
		mockStorage := &mockStorage{archivePath: testArchivePath}

		// Create orchestrator
		orchestrator := restore.NewOrchestrator(svc, mockStorage, cfg)
		require.NotNil(t, orchestrator, "Orchestrator should be created")

		// Execute restore
		result, err := orchestrator.Restore(ctx, cfg)

		// Assertions
		require.NoError(t, err, "Restore should succeed")
		require.NotNil(t, result, "Result should not be nil")
		assert.True(t, result.Success, "Restore should be successful")
		assert.Greater(t, result.ProjectID, int64(0), "Project ID should be set")
		assert.NotEmpty(t, result.ProjectURL, "Project URL should be set")

		// Verify metrics
		assert.Greater(t, result.Metrics.DurationSeconds, int64(0), "Duration should be positive")
	})

	t.Run("RestoreToNonEmptyProjectFails", func(t *testing.T) {
		// This test validates that restore fails when target project is not empty
		t.Skip("TODO: Implement non-empty project validation test")
	})

	t.Run("RestoreWithInvalidArchiveFails", func(t *testing.T) {
		// This test validates that restore fails with invalid archive
		t.Skip("TODO: Implement invalid archive test")
	})
}

// mockStorage implements restore.Storage for testing.
type mockStorage struct {
	archivePath string
}

// Get returns the local archive path (no download needed for local testing).
func (m *mockStorage) Get(ctx context.Context, key string) (string, error) {
	return m.archivePath, nil
}
