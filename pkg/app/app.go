// Package app provides the main application logic for GitLab backup.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/sgaunet/gitlab-backup/pkg/storage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/localstorage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/s3storage"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrNoStorageDefined is returned when no storage configuration is provided.
	ErrNoStorageDefined = errors.New("no storage defined")
	// ErrBackupErrors is returned when errors occur during backup process.
	ErrBackupErrors = errors.New("errors occurred during backup")
	// ErrNotDirectory is returned when a path is not a directory.
	ErrNotDirectory = errors.New("path is not a directory")
)

// App represents the main application structure.
type App struct {
	cfg           *config.Config
	gitlabService *gitlab.Service
	storage       storage.Storage
	log           Logger
}

// Logger interface defines the logging methods used by the application.
type Logger interface {
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
}

// NewApp returns a new App struct.
func NewApp(cfg *config.Config) (*App, error) {
	var err error
	app := &App{
		cfg:           cfg,
		gitlabService: gitlab.NewGitlabServiceWithTimeout(cfg.ExportTimeoutMins),
		log:           slog.New(slog.DiscardHandler),
	}
	gitlab.SetLogger(app.log)
	if cfg.IsS3ConfigValid() {
		app.storage, err = s3storage.NewS3Storage(
			cfg.S3cfg.Region,
			cfg.S3cfg.Endpoint,
			cfg.S3cfg.BucketName,
			cfg.S3cfg.BucketPath,
		)
		if err != nil {
			return nil, fmt.Errorf("error occurred during s3 storage creation: %w", err)
		}
	} else {
		if len(cfg.LocalPath) == 0 {
			return nil, ErrNoStorageDefined
		}
		app.storage = localstorage.NewLocalStorage(cfg.LocalPath)
		if stat, err := os.Stat(cfg.LocalPath); err != nil || !stat.IsDir() {
			return nil, fmt.Errorf("%s: %w", cfg.LocalPath, ErrNotDirectory)
		}
	}
	return app, nil
}

// SetLogger sets the logger.
func (a *App) SetLogger(l Logger) {
	a.log = l
	gitlab.SetLogger(l)
}

// Run runs the app.
func (a *App) Run(ctx context.Context) error {
	if a.cfg.GitlabGroupID != 0 {
		return a.ExportGroup(ctx)
	}
	if a.cfg.GitlabProjectID != 0 {
		return a.ExportProject(ctx, a.cfg.GitlabProjectID)
	}
	return nil
}

// SetGitlabEndpoint sets the gitlab endpoint.
func (a *App) SetGitlabEndpoint(gitlabAPIEndpoint string) {
	a.gitlabService.SetGitlabEndpoint(gitlabAPIEndpoint)
}

// SetToken sets the gitlab token.
func (a *App) SetToken(token string) {
	a.gitlabService.SetToken(token)
}

// ExportGroup will export all projects of the group.
func (a *App) ExportGroup(ctx context.Context) error {
	projects, err := a.gitlabService.GetProjectsOfGroup(ctx, a.cfg.GitlabGroupID)
	if err != nil {
		return fmt.Errorf("failed to get projects of group %d: %w", a.cfg.GitlabGroupID, err)
	}
	eg := errgroup.Group{}
	for project := range projects {
		if !projects[project].Archived {
			eg.Go(func() error {
				err = a.ExportProject(ctx, projects[project].ID)
				if err != nil {
					a.log.Error("error occurred during backup", "project name", projects[project].Name, "error", err.Error())
					return err
				}
				return nil
			})
		} else {
			a.log.Info("project is archived, skip", "project name", projects[project].Name)
		}
	}
	err = eg.Wait()
	if err != nil {
		return ErrBackupErrors
	}
	return nil
}

// ExportProject exports the project of the given ID.
func (a *App) ExportProject(ctx context.Context, projectID int64) error {
	project, err := a.gitlabService.GetProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project %d: %w", projectID, err)
	}

	// call prebackup hook
	if err := a.executePreBackupHook(project.Name); err != nil {
		return err
	}

	// Export GitLab archive directly as final archive
	archivePath := fmt.Sprintf("%s%s%s-%d.tar.gz", a.cfg.TmpDir, string(os.PathSeparator), project.Name, project.ID)
	err = a.gitlabService.ExportProject(ctx, &project, archivePath)
	if err != nil {
		return fmt.Errorf("failed to export project %s: %w", project.Name, err)
	}

	// call postbackup hook with archive path
	if err := a.executePostBackupHook(archivePath); err != nil {
		return err
	}

	err = a.StoreArchive(ctx, archivePath)
	if err != nil {
		return fmt.Errorf("failed to store archive %s: %w", archivePath, err)
	}

	a.log.Info("project successfully exported", "project", project.Name)
	return nil
}

// StoreArchive stores the archive.
func (a *App) StoreArchive(ctx context.Context, archiveFilePath string) error {
	err := a.storage.SaveFile(ctx, archiveFilePath, filepath.Base(archiveFilePath))
	if removeErr := os.Remove(archiveFilePath); removeErr != nil {
		a.log.Warn("failed to remove temporary file", "file", archiveFilePath, "error", removeErr)
	}
	if err != nil {
		return fmt.Errorf("failed to save file to storage: %w", err)
	}
	return nil
}

// executePreBackupHook executes the pre-backup hook if configured.
func (a *App) executePreBackupHook(projectName string) error {
	if a.cfg.Hooks.HasPreBackup() {
		a.log.Info("SaveProject (call prebackup hook)", "project name", projectName)
		err := a.cfg.Hooks.ExecutePreBackup()
		if err != nil {
			return fmt.Errorf("pre-backup hook failed: %w", err)
		}
	}
	return nil
}

// executePostBackupHook executes the post-backup hook if configured.
func (a *App) executePostBackupHook(archivePath string) error {
	if a.cfg.Hooks.HasPostBackup() {
		a.log.Info("SaveProject (call postbackup hook)", "archivePath", archivePath)
		err := a.cfg.Hooks.ExecutePostBackup(archivePath)
		if err != nil {
			return fmt.Errorf("post-backup hook failed: %w", err)
		}
	}
	return nil
}
