package app_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/app"
	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	gitlabMocks "github.com/sgaunet/gitlab-backup/pkg/gitlab/mocks"
	"github.com/sgaunet/gitlab-backup/pkg/hooks"
	"github.com/sgaunet/gitlab-backup/pkg/storage/localstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A known-good age recipient (public key) reused from the config testdata.
const testAgeRecipient = "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

// stubStorage is a minimal storage.Storage used to exercise the StoreArchive
// error path without a real backend.
type stubStorage struct {
	err   error
	calls int
}

func (s *stubStorage) SaveFile(_ context.Context, _, _ string) error {
	s.calls++
	return s.err
}

// baseConfig returns a config with temp TmpDir/LocalPath so archives can be
// written and stored on the local filesystem.
func baseConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	storageDir := t.TempDir()
	cfg := &config.Config{
		GitlabURI: "https://gitlab.com",
		TmpDir:    t.TempDir(),
		LocalPath: storageDir,
	}
	return cfg, storageDir
}

// writeArchiveFn returns an ExportProject implementation that writes a real
// (non-empty) archive to the requested path so StoreArchive/encryption can read it.
func writeArchiveFn(t *testing.T) func(context.Context, *gitlab.Project, string) error {
	t.Helper()
	return func(_ context.Context, _ *gitlab.Project, archiveFilePath string) error {
		return os.WriteFile(archiveFilePath, []byte("archive-bytes"), 0o600)
	}
}

func TestApp_Run_Dispatch(t *testing.T) {
	t.Run("group id dispatches to ExportGroup", func(t *testing.T) {
		cfg, _ := baseConfig(t)
		cfg.GitlabGroupID = 10

		called := false
		svc := &gitlabMocks.BackupServiceMock{
			GetProjectsOfGroupFunc: func(_ context.Context, groupID int64) ([]gitlab.Project, error) {
				called = true
				assert.Equal(t, int64(10), groupID)
				return nil, nil
			},
		}
		a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)
		require.NoError(t, a.Run(context.Background()))
		assert.True(t, called, "ExportGroup should be invoked")
	})

	t.Run("project id dispatches to ExportProject", func(t *testing.T) {
		cfg, _ := baseConfig(t)
		cfg.GitlabProjectID = 55

		svc := &gitlabMocks.BackupServiceMock{
			GetProjectFunc: func(_ context.Context, projectID int64) (gitlab.Project, error) {
				assert.Equal(t, int64(55), projectID)
				return gitlab.Project{ID: 55, Name: "proj"}, nil
			},
			ExportProjectFunc: writeArchiveFn(t),
		}
		a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)
		require.NoError(t, a.Run(context.Background()))
		require.Len(t, svc.GetProjectCalls(), 1)
	})

	t.Run("neither id is a no-op", func(t *testing.T) {
		cfg, _ := baseConfig(t)
		svc := &gitlabMocks.BackupServiceMock{}
		a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)
		require.NoError(t, a.Run(context.Background()))
	})
}

func TestApp_ExportProject_HappyPath(t *testing.T) {
	cfg, storageDir := baseConfig(t)

	svc := &gitlabMocks.BackupServiceMock{
		GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
			return gitlab.Project{ID: 7, Name: "myproj"}, nil
		},
		ExportProjectFunc: writeArchiveFn(t),
	}
	a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)

	require.NoError(t, a.ExportProject(context.Background(), 7))

	// Archive should have landed in storage under its base name...
	stored := filepath.Join(storageDir, "myproj-7.tar.gz")
	_, err := os.Stat(stored)
	require.NoError(t, err, "archive should be stored")

	// ...and the temporary archive removed from TmpDir.
	tmpArchive := filepath.Join(cfg.TmpDir, "myproj-7.tar.gz")
	_, statErr := os.Stat(tmpArchive)
	assert.True(t, os.IsNotExist(statErr), "temp archive should be removed after storing")
}

func TestApp_ExportProject_GetProjectError(t *testing.T) {
	cfg, _ := baseConfig(t)
	svc := &gitlabMocks.BackupServiceMock{
		GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
			return gitlab.Project{}, errors.New("boom")
		},
	}
	a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)

	err := a.ExportProject(context.Background(), 7)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get project")
}

func TestApp_ExportProject_ExportError(t *testing.T) {
	cfg, _ := baseConfig(t)
	svc := &gitlabMocks.BackupServiceMock{
		GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
			return gitlab.Project{ID: 7, Name: "myproj"}, nil
		},
		ExportProjectFunc: func(_ context.Context, _ *gitlab.Project, _ string) error {
			return errors.New("export failed")
		},
	}
	a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)

	err := a.ExportProject(context.Background(), 7)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to export project")
}

func TestApp_StoreArchive_Error(t *testing.T) {
	cfg, _ := baseConfig(t)
	stub := &stubStorage{err: errors.New("disk full")}
	svc := &gitlabMocks.BackupServiceMock{
		GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
			return gitlab.Project{ID: 7, Name: "myproj"}, nil
		},
		ExportProjectFunc: writeArchiveFn(t),
	}
	a := app.NewAppWithService(cfg, svc, stub, nil)

	err := a.ExportProject(context.Background(), 7)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save file to storage")
	assert.Equal(t, 1, stub.calls)

	// StoreArchive still removes the temp file even when saving fails.
	tmpArchive := filepath.Join(cfg.TmpDir, "myproj-7.tar.gz")
	_, statErr := os.Stat(tmpArchive)
	assert.True(t, os.IsNotExist(statErr), "temp archive should be removed even on save error")
}

func TestApp_ExportProject_Hooks(t *testing.T) {
	t.Run("pre and post hooks succeed", func(t *testing.T) {
		cfg, _ := baseConfig(t)
		cfg.Hooks = hooks.Hooks{PreBackup: "true", PostBackup: "true"}
		svc := &gitlabMocks.BackupServiceMock{
			GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
				return gitlab.Project{ID: 7, Name: "myproj"}, nil
			},
			ExportProjectFunc: writeArchiveFn(t),
		}
		a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)
		require.NoError(t, a.ExportProject(context.Background(), 7))
	})

	t.Run("failing pre-backup hook aborts", func(t *testing.T) {
		cfg, _ := baseConfig(t)
		cfg.Hooks = hooks.Hooks{PreBackup: "false"}
		svc := &gitlabMocks.BackupServiceMock{
			GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
				return gitlab.Project{ID: 7, Name: "myproj"}, nil
			},
			ExportProjectFunc: writeArchiveFn(t),
		}
		a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)
		err := a.ExportProject(context.Background(), 7)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pre-backup hook failed")
	})
}

func TestApp_ExportProject_Encryption(t *testing.T) {
	t.Run("inline recipient encrypts before storing", func(t *testing.T) {
		cfg, storageDir := baseConfig(t)
		cfg.Age = config.AgeConfig{Recipients: []string{testAgeRecipient}}
		svc := &gitlabMocks.BackupServiceMock{
			GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
				return gitlab.Project{ID: 7, Name: "myproj"}, nil
			},
			ExportProjectFunc: writeArchiveFn(t),
		}
		a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)
		require.NoError(t, a.ExportProject(context.Background(), 7))

		stored := filepath.Join(storageDir, "myproj-7.tar.gz")
		data, err := os.ReadFile(stored)
		require.NoError(t, err)
		assert.NotEqual(t, []byte("archive-bytes"), data, "stored archive should be encrypted, not plaintext")
	})

	t.Run("recipients file encrypts", func(t *testing.T) {
		cfg, _ := baseConfig(t)
		recipientsFile := filepath.Join(t.TempDir(), "recipients.txt")
		require.NoError(t, os.WriteFile(recipientsFile, []byte(testAgeRecipient+"\n"), 0o600))
		cfg.Age = config.AgeConfig{RecipientsFile: recipientsFile}
		svc := &gitlabMocks.BackupServiceMock{
			GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
				return gitlab.Project{ID: 7, Name: "myproj"}, nil
			},
			ExportProjectFunc: writeArchiveFn(t),
		}
		a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)
		require.NoError(t, a.ExportProject(context.Background(), 7))
	})

	t.Run("invalid recipient errors", func(t *testing.T) {
		cfg, _ := baseConfig(t)
		cfg.Age = config.AgeConfig{Recipients: []string{"not-a-valid-age-key"}}
		svc := &gitlabMocks.BackupServiceMock{
			GetProjectFunc: func(_ context.Context, _ int64) (gitlab.Project, error) {
				return gitlab.Project{ID: 7, Name: "myproj"}, nil
			},
			ExportProjectFunc: writeArchiveFn(t),
		}
		a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)
		err := a.ExportProject(context.Background(), 7)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "age encryption")
	})
}

func TestApp_ExportGroup_ArchivedSkipAndFailureSummary(t *testing.T) {
	cfg, _ := baseConfig(t)
	cfg.GitlabGroupID = 100

	svc := &gitlabMocks.BackupServiceMock{
		GetProjectsOfGroupFunc: func(_ context.Context, _ int64) ([]gitlab.Project, error) {
			return []gitlab.Project{
				{ID: 1, Name: "ok"},
				{ID: 2, Name: "arch", Archived: true},
				{ID: 3, Name: "boom"},
			}, nil
		},
		GetProjectFunc: func(_ context.Context, projectID int64) (gitlab.Project, error) {
			switch projectID {
			case 1:
				return gitlab.Project{ID: 1, Name: "ok"}, nil
			case 3:
				return gitlab.Project{ID: 3, Name: "boom"}, nil
			default:
				return gitlab.Project{}, errors.New("unexpected project id")
			}
		},
		ExportProjectFunc: func(_ context.Context, project *gitlab.Project, archiveFilePath string) error {
			if project.ID == 3 {
				return errors.New("export exploded")
			}
			return os.WriteFile(archiveFilePath, []byte("archive-bytes"), 0o600)
		},
	}
	a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)

	err := a.ExportGroup(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, app.ErrBackupErrors)

	// Archived project 2 must be skipped (never fetched/exported).
	for _, c := range svc.GetProjectCalls() {
		assert.NotEqual(t, int64(2), c.ProjectID, "archived project should never be fetched")
	}
	// Only projects 1 and 3 are processed.
	assert.Len(t, svc.GetProjectCalls(), 2)
}

func TestApp_ExportGroup_AllSuccess(t *testing.T) {
	cfg, _ := baseConfig(t)
	cfg.GitlabGroupID = 100

	svc := &gitlabMocks.BackupServiceMock{
		GetProjectsOfGroupFunc: func(_ context.Context, _ int64) ([]gitlab.Project, error) {
			return []gitlab.Project{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, nil
		},
		GetProjectFunc: func(_ context.Context, projectID int64) (gitlab.Project, error) {
			return gitlab.Project{ID: projectID, Name: "p"}, nil
		},
		ExportProjectFunc: writeArchiveFn(t),
	}
	a := app.NewAppWithService(cfg, svc, localstorage.NewLocalStorage(cfg.LocalPath), nil)

	require.NoError(t, a.ExportGroup(context.Background()))
}
