package restore_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/app/restore"
	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	restoreMocks "github.com/sgaunet/gitlab-backup/pkg/app/restore/mocks"
	gitlabMocks "github.com/sgaunet/gitlab-backup/pkg/gitlab/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	gitlabAPI "gitlab.com/gitlab-org/api/client-go"
)

// setupMockGitLabService creates a standard mock with common happy-path responses.
func setupMockGitLabService(t *testing.T, customizations ...func(*gitlabMocks.GitLabClientMock)) gitlab.GitLabService {
	t.Helper()

	mockClient := &gitlabMocks.GitLabClientMock{
		ProjectsFunc: func() gitlab.ProjectsService {
			return &gitlabMocks.ProjectsServiceMock{
				GetProjectFunc: func(pid any, opt *gitlabAPI.GetProjectOptions, options ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Project, *gitlabAPI.Response, error) {
					return nil, nil, errors.New("404 Project Not Found")
				},
			}
		},
		CommitsFunc: func() gitlab.CommitsService {
			return &gitlabMocks.CommitsServiceMock{
				ListCommitsFunc: func(pid any, opt *gitlabAPI.ListCommitsOptions, options ...gitlabAPI.RequestOptionFunc) ([]*gitlabAPI.Commit, *gitlabAPI.Response, error) {
					return []*gitlabAPI.Commit{}, &gitlabAPI.Response{}, nil
				},
			}
		},
		IssuesFunc: func() gitlab.IssuesService {
			return &gitlabMocks.IssuesServiceMock{
				ListProjectIssuesFunc: func(pid any, opt *gitlabAPI.ListProjectIssuesOptions, options ...gitlabAPI.RequestOptionFunc) ([]*gitlabAPI.Issue, *gitlabAPI.Response, error) {
					return []*gitlabAPI.Issue{}, &gitlabAPI.Response{}, nil
				},
			}
		},
		LabelsFunc: func() gitlab.LabelsService {
			return &gitlabMocks.LabelsServiceMock{
				ListLabelsFunc: func(pid any, opt *gitlabAPI.ListLabelsOptions, options ...gitlabAPI.RequestOptionFunc) ([]*gitlabAPI.Label, *gitlabAPI.Response, error) {
					return []*gitlabAPI.Label{}, &gitlabAPI.Response{}, nil
				},
			}
		},
		ProjectImportExportFunc: func() gitlab.ProjectImportExportService {
			return &gitlabMocks.ProjectImportExportServiceMock{}
		},
	}

	for _, customize := range customizations {
		customize(mockClient)
	}

	return &gitlabMocks.GitLabServiceMock{
		ClientFunc: func() gitlab.GitLabClient {
			return mockClient
		},
		RateLimitImportAPIFunc: func() *rate.Limiter {
			return rate.NewLimiter(rate.Every(60*time.Second), 6)
		},
	}
}

// setupMockStorage creates a mock storage with happy-path responses.
func setupMockStorage(t *testing.T) *restoreMocks.StorageMock {
	t.Helper()

	return &restoreMocks.StorageMock{
		GetFunc: func(_ context.Context, _ string) (string, error) {
			return "/tmp/fake-archive.tar.gz", nil
		},
	}
}

// createTestArchive creates a minimal placeholder archive file.
func createTestArchive(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test-archive.tar.gz")

	err := os.WriteFile(archivePath, []byte("fake-archive-data"), 0o600)
	require.NoError(t, err)

	return archivePath
}

func TestNewOrchestrator(t *testing.T) {
	t.Run("SuccessfulCreation", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t)
		mockStorage := setupMockStorage(t)
		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)

		assert.NotNil(t, orchestrator)
	})

	t.Run("WithNoLogTime", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t)
		mockStorage := setupMockStorage(t)
		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			NoLogTime:         true,
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)

		assert.NotNil(t, orchestrator)
	})
}

func TestRestore_ValidationPhase(t *testing.T) {
	ctx := context.Background()

	t.Run("SkippedWithOverwrite", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t)
		mockStorage := setupMockStorage(t)
		archivePath := createTestArchive(t)

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     archivePath,
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  true,
			StorageType:       "local",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		result, err := orchestrator.Restore(ctx, cfg)

		// Fails at extraction (fake archive), but validation should be skipped
		assert.Error(t, err)
		for _, e := range result.Errors {
			assert.NotEqual(t, restore.PhaseValidation, e.Phase, "Should not have validation error when overwrite is enabled")
		}
	})

	t.Run("FailsNonEmptyProject", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t, func(client *gitlabMocks.GitLabClientMock) {
			client.ProjectsFunc = func() gitlab.ProjectsService {
				return &gitlabMocks.ProjectsServiceMock{
					GetProjectFunc: func(pid any, _ *gitlabAPI.GetProjectOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Project, *gitlabAPI.Response, error) {
						return &gitlabAPI.Project{ID: 123, Name: "test-project"}, &gitlabAPI.Response{}, nil
					},
				}
			}
			client.CommitsFunc = func() gitlab.CommitsService {
				return &gitlabMocks.CommitsServiceMock{
					ListCommitsFunc: func(pid any, _ *gitlabAPI.ListCommitsOptions, _ ...gitlabAPI.RequestOptionFunc) ([]*gitlabAPI.Commit, *gitlabAPI.Response, error) {
						return []*gitlabAPI.Commit{{ID: "abc123"}}, &gitlabAPI.Response{}, nil
					},
				}
			}
		})
		mockStorage := setupMockStorage(t)
		archivePath := createTestArchive(t)

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     archivePath,
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  false,
			StorageType:       "local",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		result, err := orchestrator.Restore(ctx, cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is not empty")
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("PassesProjectDoesNotExist", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t) // Default: 404
		mockStorage := setupMockStorage(t)
		archivePath := createTestArchive(t)

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     archivePath,
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  false,
			StorageType:       "local",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		_, err := orchestrator.Restore(ctx, cfg)

		// Validation passes (404), extraction fails (fake archive)
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "project is not empty")
	})

	t.Run("PassesEmptyProject", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t, func(client *gitlabMocks.GitLabClientMock) {
			client.ProjectsFunc = func() gitlab.ProjectsService {
				return &gitlabMocks.ProjectsServiceMock{
					GetProjectFunc: func(pid any, _ *gitlabAPI.GetProjectOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Project, *gitlabAPI.Response, error) {
						return &gitlabAPI.Project{ID: 123, Name: "test-project"}, &gitlabAPI.Response{}, nil
					},
				}
			}
		})
		mockStorage := setupMockStorage(t)
		archivePath := createTestArchive(t)

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     archivePath,
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  false,
			StorageType:       "local",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		_, err := orchestrator.Restore(ctx, cfg)

		// Validation passes (empty project), extraction fails (fake archive)
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "project is not empty")
	})
}

func TestRestore_DownloadPhase(t *testing.T) {
	ctx := context.Background()

	t.Run("SkippedForLocalStorage", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t)
		mockStorage := setupMockStorage(t)
		archivePath := createTestArchive(t)

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     archivePath,
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  true,
			StorageType:       "local",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		_, _ = orchestrator.Restore(ctx, cfg)

		assert.Empty(t, mockStorage.GetCalls(), "Storage.Get should not be called for local storage")
	})

	t.Run("CalledForS3Storage", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t)

		tempFile := filepath.Join(t.TempDir(), "downloaded-archive.tar.gz")
		err := os.WriteFile(tempFile, []byte("fake-downloaded-data"), 0o600)
		require.NoError(t, err)

		mockStorage := &restoreMocks.StorageMock{
			GetFunc: func(_ context.Context, _ string) (string, error) {
				return tempFile, nil
			},
		}

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     "s3://bucket/archive.tar.gz",
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  true,
			StorageType:       "s3",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		_, _ = orchestrator.Restore(ctx, cfg)

		assert.NotEmpty(t, mockStorage.GetCalls(), "Storage.Get should be called for S3 storage")
		assert.Equal(t, "s3://bucket/archive.tar.gz", mockStorage.GetCalls()[0].Key)
	})

	t.Run("FailsFromS3", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t)
		mockStorage := &restoreMocks.StorageMock{
			GetFunc: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("S3 connection failed")
			},
		}

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     "s3://bucket/archive.tar.gz",
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  true,
			StorageType:       "s3",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		result, err := orchestrator.Restore(ctx, cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "download failed")
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Errors)
		assert.Equal(t, restore.PhaseDownload, result.Errors[0].Phase)
	})
}

func TestRestore_ExtractionPhase(t *testing.T) {
	ctx := context.Background()

	t.Run("FailsInvalidArchive", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t)
		mockStorage := setupMockStorage(t)

		invalidArchive := filepath.Join(t.TempDir(), "invalid.tar.gz")
		err := os.WriteFile(invalidArchive, []byte("not-a-valid-archive"), 0o600)
		require.NoError(t, err)

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     invalidArchive,
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  true,
			StorageType:       "local",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		result, err := orchestrator.Restore(ctx, cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "archive extraction failed")
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Errors)
		assert.Equal(t, restore.PhaseExtraction, result.Errors[0].Phase)
	})

	t.Run("FailsTempDirCreation", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t)
		mockStorage := setupMockStorage(t)
		archivePath := createTestArchive(t)

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     archivePath,
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  true,
			StorageType:       "local",
			TmpDir:            "/invalid/nonexistent/path",
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		result, err := orchestrator.Restore(ctx, cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create temp directory")
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Errors)
		assert.Equal(t, restore.PhaseExtraction, result.Errors[0].Phase)
	})
}

func TestRestore_CleanupRunsOnError(t *testing.T) {
	ctx := context.Background()

	mockGitLab := setupMockGitLabService(t)
	mockStorage := setupMockStorage(t)

	invalidArchive := filepath.Join(t.TempDir(), "invalid.tar.gz")
	err := os.WriteFile(invalidArchive, []byte("invalid"), 0o600)
	require.NoError(t, err)

	cfg := &config.Config{
		GitlabURI:         "https://gitlab.com",
		RestoreSource:     invalidArchive,
		RestoreTargetNS:   "test-ns",
		RestoreTargetPath: "test-project",
		RestoreOverwrite:  true,
		StorageType:       "local",
		TmpDir:            t.TempDir(),
	}

	orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
	result, restoreErr := orchestrator.Restore(ctx, cfg)

	// Cleanup runs via defer even on extraction failure
	require.Error(t, restoreErr)
	assert.False(t, result.Success)
}

func TestRestore_ErrorCollection(t *testing.T) {
	ctx := context.Background()

	t.Run("ErrorsCollectedWithPhaseInfo", func(t *testing.T) {
		mockGitLab := setupMockGitLabService(t, func(client *gitlabMocks.GitLabClientMock) {
			client.ProjectsFunc = func() gitlab.ProjectsService {
				return &gitlabMocks.ProjectsServiceMock{
					GetProjectFunc: func(pid any, _ *gitlabAPI.GetProjectOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Project, *gitlabAPI.Response, error) {
						return &gitlabAPI.Project{ID: 123}, &gitlabAPI.Response{}, nil
					},
				}
			}
			client.IssuesFunc = func() gitlab.IssuesService {
				return &gitlabMocks.IssuesServiceMock{
					ListProjectIssuesFunc: func(pid any, _ *gitlabAPI.ListProjectIssuesOptions, _ ...gitlabAPI.RequestOptionFunc) ([]*gitlabAPI.Issue, *gitlabAPI.Response, error) {
						return []*gitlabAPI.Issue{{ID: 1}}, &gitlabAPI.Response{}, nil
					},
				}
			}
		})
		mockStorage := setupMockStorage(t)
		archivePath := createTestArchive(t)

		cfg := &config.Config{
			GitlabURI:         "https://gitlab.com",
			RestoreSource:     archivePath,
			RestoreTargetNS:   "test-ns",
			RestoreTargetPath: "test-project",
			RestoreOverwrite:  false,
			StorageType:       "local",
			TmpDir:            t.TempDir(),
		}

		orchestrator := restore.NewOrchestrator(mockGitLab, mockStorage, cfg)
		result, err := orchestrator.Restore(ctx, cfg)

		require.Error(t, err)
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Errors)
		assert.Equal(t, restore.PhaseValidation, result.Errors[0].Phase)
		assert.True(t, result.Errors[0].Fatal)
	})
}
