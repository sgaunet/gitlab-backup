package restore_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/app/restore"
	restoreMocks "github.com/sgaunet/gitlab-backup/pkg/app/restore/mocks"
	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	gitlabMocks "github.com/sgaunet/gitlab-backup/pkg/gitlab/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlabAPI "gitlab.com/gitlab-org/api/client-go"
)

// createValidArchive writes a genuinely valid .tar.gz so ExtractArchive's
// gzip/tar validation passes and the workflow reaches the import phase.
func createValidArchive(t *testing.T) string {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	body := []byte("project export placeholder")
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "project.bundle",
		Mode: 0o600,
		Size: int64(len(body)),
	}))
	_, err := tw.Write(body)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	archivePath := filepath.Join(t.TempDir(), "valid-archive.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, buf.Bytes(), 0o600))
	return archivePath
}

// withImportSuccess customizes the mock GitLab client so ImportFromFile is
// accepted and the first ImportStatus poll reports "finished".
func withImportSuccess(client *gitlabMocks.GitLabClientMock) {
	client.ProjectImportExportFunc = func() gitlab.ProjectImportExportService {
		return &gitlabMocks.ProjectImportExportServiceMock{
			ImportFromFileFunc: func(_ context.Context, _ io.Reader, _ *gitlabAPI.ImportFileOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.ImportStatus, *gitlabAPI.Response, error) {
				return &gitlabAPI.ImportStatus{ID: 42, ImportStatus: "scheduled"}, &gitlabAPI.Response{}, nil
			},
			ImportStatusFunc: func(_ context.Context, _ any, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.ImportStatus, *gitlabAPI.Response, error) {
				return &gitlabAPI.ImportStatus{ID: 42, ImportStatus: "finished"}, &gitlabAPI.Response{}, nil
			},
		}
	}
}

func successRestoreConfig(t *testing.T, archivePath string) *config.Config {
	t.Helper()
	return &config.Config{
		GitlabURI:         "https://gitlab.com",
		RestoreSource:     archivePath,
		RestoreTargetNS:   "test-ns",
		RestoreTargetPath: "test-project",
		RestoreOverwrite:  true, // skip validation phase
		StorageType:       "local",
		TmpDir:            t.TempDir(),
		ImportTimeoutMins: 60,
	}
}

// TestRestore_FullSuccess_NoOpProgress drives the whole workflow to completion
// with the NoOp reporter, covering the success paths of Restore (extraction →
// import → complete) and the NoOpProgressReporter methods.
func TestRestore_FullSuccess_NoOpProgress(t *testing.T) {
	mockGitLab := setupMockGitLabService(t, withImportSuccess)
	mockStorage := setupMockStorage(t)
	cfg := successRestoreConfig(t, createValidArchive(t))

	orchestrator := restore.NewOrchestratorWithProgress(mockGitLab, mockStorage, restore.NewNoOpProgressReporter())
	result, err := orchestrator.Restore(context.Background(), cfg)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, int64(42), result.ProjectID)
	assert.Empty(t, result.Errors)
}

// TestRestore_FullSuccess_MockProgress asserts the phase transitions reported
// during a successful restore, using the injectable ProgressReporter seam.
func TestRestore_FullSuccess_MockProgress(t *testing.T) {
	mockGitLab := setupMockGitLabService(t, withImportSuccess)
	mockStorage := setupMockStorage(t)
	cfg := successRestoreConfig(t, createValidArchive(t))

	progress := &restoreMocks.ProgressReporterMock{
		StartPhaseFunc:    func(_ restore.Phase) {},
		UpdatePhaseFunc:   func(_ restore.Phase, _, _ int) {},
		CompletePhaseFunc: func(_ restore.Phase) {},
		FailPhaseFunc:     func(_ restore.Phase, _ error) {},
		SkipPhaseFunc:     func(_ restore.Phase, _ string) {},
	}

	orchestrator := restore.NewOrchestratorWithProgress(mockGitLab, mockStorage, progress)
	result, err := orchestrator.Restore(context.Background(), cfg)

	require.NoError(t, err)
	assert.True(t, result.Success)

	// Validation is skipped (overwrite), extraction + import must complete.
	started := make(map[restore.Phase]bool)
	for _, c := range progress.StartPhaseCalls() {
		started[c.Phase] = true
	}
	assert.True(t, started[restore.PhaseExtraction], "extraction phase should start")
	assert.True(t, started[restore.PhaseImport], "import phase should start")

	completed := make(map[restore.Phase]bool)
	for _, c := range progress.CompletePhaseCalls() {
		completed[c.Phase] = true
	}
	assert.True(t, completed[restore.PhaseImport], "import phase should complete")

	// Overwrite means validation is skipped, never failed.
	for _, c := range progress.FailPhaseCalls() {
		assert.NotEqual(t, restore.PhaseValidation, c.Phase)
	}
	assert.NotEmpty(t, progress.SkipPhaseCalls(), "validation should be skipped with overwrite")
}
