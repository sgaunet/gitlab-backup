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
			ImportFromFileFunc: func(archive io.Reader, opt *gitlabapi.ImportFileOptions, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				// Return successful import initiation
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "scheduled",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
			ImportStatusFunc: func(pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
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
		service := gitlab.NewImportService(mockProjectImportExport)
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
			ImportFromFileFunc: func(archive io.Reader, opt *gitlabapi.ImportFileOptions, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				return nil, nil, errors.New("import initiation failed")
			},
		}

		archiveReader := bytes.NewReader([]byte("fake-data"))

		// Execute import
		service := gitlab.NewImportService(mockProjectImportExport)
		status, err := service.ImportProject(ctx, archiveReader, "namespace", "project-path")

		// Assertions
		require.Error(t, err, "Import should fail")
		assert.Nil(t, status, "Status should be nil on error")
		assert.Contains(t, err.Error(), "import initiation failed", "Error should indicate initiation failure")
	})

	t.Run("ImportStatusFailed", func(t *testing.T) {
		// Setup mocks
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportFromFileFunc: func(archive io.Reader, opt *gitlabapi.ImportFileOptions, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "scheduled",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
			ImportStatusFunc: func(pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
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
		service := gitlab.NewImportService(mockProjectImportExport)
		_, err := service.ImportProject(ctx, archiveReader, "namespace", "project-path")

		// Assertions
		require.Error(t, err, "Import should fail when status is failed")
		assert.Contains(t, err.Error(), "disk full", "Error should contain failure reason")
	})

	t.Run("ImportTimeout", func(t *testing.T) {
		// Setup mocks with delayed status
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportFromFileFunc: func(archive io.Reader, opt *gitlabapi.ImportFileOptions, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "scheduled",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
			ImportStatusFunc: func(pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
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

		// Execute import
		service := gitlab.NewImportService(mockProjectImportExport)
		_, err := service.ImportProject(ctx, archiveReader, "namespace", "project-path")

		// Assertions
		require.Error(t, err, "Import should timeout")
		assert.Contains(t, err.Error(), "context deadline exceeded", "Error should indicate timeout")
	})
}

func TestWaitForImport(t *testing.T) {
	ctx := context.Background()

	t.Run("QuickFinish", func(t *testing.T) {
		// Setup mock that finishes immediately
		callCount := 0
		mockProjectImportExport := &mocks.ProjectImportExportServiceMock{
			ImportStatusFunc: func(pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				callCount++
				return &gitlabapi.ImportStatus{
					ID:            123,
					ImportStatus:  "finished",
					ImportError: "",
				}, &gitlabapi.Response{}, nil
			},
		}

		service := gitlab.NewImportService(mockProjectImportExport)
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
			ImportStatusFunc: func(pid any, options ...gitlabapi.RequestOptionFunc) (*gitlabapi.ImportStatus, *gitlabapi.Response, error) {
				callCount++
				if callCount == 1 {
					return &gitlabapi.ImportStatus{ID: 123, ImportStatus: "scheduled"}, &gitlabapi.Response{}, nil
				} else if callCount == 2 {
					return &gitlabapi.ImportStatus{ID: 123, ImportStatus: "started"}, &gitlabapi.Response{}, nil
				}
				return &gitlabapi.ImportStatus{ID: 123, ImportStatus: "finished"}, &gitlabapi.Response{}, nil
			},
		}

		service := gitlab.NewImportService(mockProjectImportExport)
		status, err := service.WaitForImport(ctx, 123, 10*time.Minute)

		// Assertions
		require.NoError(t, err, "Wait should succeed")
		assert.Equal(t, "finished", status.ImportStatus, "Status should be finished")
		assert.GreaterOrEqual(t, callCount, 3, "Should poll multiple times")
	})
}
