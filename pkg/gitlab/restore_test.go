package gitlab_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlabapi "gitlab.com/gitlab-org/api/client-go"
)

func TestImportProject(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessfulImport", func(t *testing.T) {
		// Setup mocks
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportFromFileFunc: func(_ context.Context, archive io.Reader, opt *gitlabapi.ImportFileOptions, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				// Return successful import initiation
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "scheduled",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
			ImportStatusFunc: func(_ context.Context, pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				// Return finished import status
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "finished",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
		}

		// Create archive reader
		archiveData := []byte("fake-archive-data")
		archiveReader := bytes.NewReader(archiveData)

		// Execute import
		service := gitlab.NewImportService(mockProjectImportExport, 10*time.Minute)
		status, err := service.ImportProject(ctx, archiveReader, "namespace", "project-path")

		// Assertions
		require.NoError(t, err, "Import should succeed")
		require.NotNil(t, status, "Status should not be nil")
		assert.Equal(t, int64(123), status.ID, "Project ID should match")
		assert.Equal(t, "finished", status.ImportStatus, "Import should be finished")
	})

	t.Run("ImportInitiationFails", func(t *testing.T) {
		// Setup mocks
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportFromFileFunc: func(_ context.Context, archive io.Reader, opt *gitlabapi.ImportFileOptions, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				return nil, nil, errors.New("import initiation failed")
			},
		}

		archiveReader := bytes.NewReader([]byte("fake-data"))

		// Execute import
		service := gitlab.NewImportService(mockProjectImportExport, 10*time.Minute)
		status, err := service.ImportProject(ctx, archiveReader, "namespace", "project-path")

		// Assertions
		require.Error(t, err, "Import should fail")
		assert.Nil(t, status, "Status should be nil on error")
		assert.Contains(t, err.Error(), "import initiation failed", "Error should indicate initiation failure")
	})

	t.Run("ImportStatusFailed", func(t *testing.T) {
		// Setup mocks
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportFromFileFunc: func(_ context.Context, archive io.Reader, opt *gitlabapi.ImportFileOptions, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "scheduled",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
			ImportStatusFunc: func(_ context.Context, pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				// Return failed import status
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "failed",
					ImportError: "disk full",
				}, &gitlabapi.Response{}, nil
			},
		}

		archiveReader := bytes.NewReader([]byte("fake-data"))

		// Execute import
		service := gitlab.NewImportService(mockProjectImportExport, 10*time.Minute)
		_, err := service.ImportProject(ctx, archiveReader, "namespace", "project-path")

		// Assertions
		require.Error(t, err, "Import should fail when status is failed")
		assert.Contains(t, err.Error(), "disk full", "Error should contain failure reason")
	})

	t.Run("ImportTimeout", func(t *testing.T) {
		// Setup mocks with delayed status
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportFromFileFunc: func(_ context.Context, archive io.Reader, opt *gitlabapi.ImportFileOptions, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "scheduled",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
			ImportStatusFunc: func(_ context.Context, pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				// Always return "started" to simulate timeout
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "started",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
		}

		archiveReader := bytes.NewReader([]byte("fake-data"))

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		// Execute import — service-level timeout is generous; the parent ctx timeout
		// is what actually fires here. WaitForImport's grace-period check kicks in.
		service := gitlab.NewImportService(mockProjectImportExport, 10*time.Minute)
		_, err := service.ImportProject(ctx, archiveReader, "namespace", "project-path")

		// Assertions
		require.Error(t, err, "Import should timeout")
		require.ErrorIs(t, err, gitlab.ErrImportTimeout, "Error should be ErrImportTimeout")
		assert.Contains(t, err.Error(), "may still be in progress",
			"Error should hint at checking the GitLab web UI")
	})
}

func TestWaitForImport(t *testing.T) {
	ctx := context.Background()

	t.Run("QuickFinish", func(t *testing.T) {
		// Setup mock that finishes immediately
		callCount := 0
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportStatusFunc: func(_ context.Context, pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				callCount++
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "finished",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
		}

		service := gitlab.NewImportService(mockProjectImportExport, 10*time.Minute)
		status, err := service.WaitForImport(ctx, 123, 10*time.Minute)

		// Assertions
		require.NoError(t, err, "Wait should succeed")
		assert.Equal(t, "finished", status.ImportStatus, "Status should be finished")
		assert.Equal(t, 1, callCount, "Should only call status once")
	})

	t.Run("ProgressThenFinish", func(t *testing.T) {
		// Setup mock that progresses through states
		callCount := 0
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportStatusFunc: func(_ context.Context, pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				callCount++
				if callCount == 1 {
					return &gitlabapi.ImportStatus{ID: 123, ImportStatus: "scheduled"}, &gitlabapi.Response{}, nil
				} else if callCount == 2 {
					return &gitlabapi.ImportStatus{ID: 123, ImportStatus: "started"}, &gitlabapi.Response{}, nil
				}
				return &gitlabapi.ImportStatus{ID: 123, ImportStatus: "finished"}, &gitlabapi.Response{}, nil
			},
		}

		service := gitlab.NewImportService(mockProjectImportExport, 10*time.Minute)
		status, err := service.WaitForImport(ctx, 123, 10*time.Minute)

		// Assertions
		require.NoError(t, err, "Wait should succeed")
		assert.Equal(t, "finished", status.ImportStatus, "Status should be finished")
		assert.GreaterOrEqual(t, callCount, 3, "Should poll multiple times")
	})

	t.Run("TimeoutReturnsImportTimeoutError", func(t *testing.T) {
		// Mock returns "started" forever — forces WaitForImport to hit its deadline.
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportStatusFunc: func(_ context.Context, _ any, _ ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				return &gitlabapi.ImportStatus{ID: 123, ImportStatus: "started"}, &gitlabapi.Response{}, nil
			},
		}

		service := gitlab.NewImportService(mockProjectImportExport, 10*time.Minute)
		// Tight per-call timeout — well under the grace period — to force the timeout path.
		_, err := service.WaitForImport(ctx, 123, 100*time.Millisecond)

		require.Error(t, err, "WaitForImport should fail on timeout")
		require.ErrorIs(t, err, gitlab.ErrImportTimeout,
			"timeout failures must map to ErrImportTimeout, not a rate-limit error")
		assert.NotContains(t, err.Error(), "rate limit wait failed",
			"Error must not mislead about rate limiting")
		assert.Contains(t, err.Error(), "project ID 123",
			"Error should include the project ID for cross-checking on web UI")
	})
}

func TestNewImportService_ZeroTimeoutFallsBackToDefault(t *testing.T) {
	// Mock that always finishes — only serves to confirm the constructor works.
	mock := &mocks.ProjectImportExportServiceMock{
		ImportStatusFunc: func(_ context.Context, _ any, _ ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
			return &gitlabapi.ImportStatus{ID: 1, ImportStatus: "finished"}, &gitlabapi.Response{}, nil
		},
	}

	// Zero timeout should trigger the DefaultImportTimeoutMins fallback.
	// We can't directly inspect the unexported timeout field from a black-box test;
	// instead, we verify the constructor doesn't panic and the service behaves correctly.
	service := gitlab.NewImportService(mock, 0)
	require.NotNil(t, service, "Service should be created with zero timeout")

	// Negative also falls back.
	service = gitlab.NewImportService(mock, -1*time.Second)
	require.NotNil(t, service, "Service should be created with negative timeout")
}
