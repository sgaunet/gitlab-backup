package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/sgaunet/gitlab-backup/pkg/storage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/localstorage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/s3storage"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg           *config.Config
	gitlabService *gitlab.GitlabService
	storage       storage.Storage
	log           Logger
}

type Logger interface {
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
}

// NewApp returns a new App struct
func NewApp(cfg *config.Config) (*App, error) {
	var err error
	app := &App{
		cfg:           cfg,
		gitlabService: gitlab.NewGitlabService(),
		log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	gitlab.SetLogger(app.log)
	if cfg.IsS3ConfigValid() {
		app.storage, err = s3storage.NewS3Storage(cfg.S3cfg.Region, cfg.S3cfg.Endpoint, cfg.S3cfg.BucketName, cfg.S3cfg.BucketPath)
		if err != nil {
			return nil, fmt.Errorf("error occured during s3 storage creation: %s", err.Error())
		}
	} else {
		if len(cfg.LocalPath) == 0 {
			return nil, fmt.Errorf("no storage defined")
		}
		app.storage = localstorage.NewLocalStorage(cfg.LocalPath)
		if stat, err := os.Stat(cfg.LocalPath); err != nil || !stat.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", cfg.LocalPath)
		}
	}
	return app, nil
}

// SetLogger sets the logger
func (a *App) SetLogger(l Logger) {
	a.log = l
	gitlab.SetLogger(l)
}

// Run runs the app
func (a *App) Run(ctx context.Context) error {
	if a.cfg.GitlabGroupID != 0 {
		return a.ExportGroup(ctx)
	}
	if a.cfg.GitlabProjectID != 0 {
		return a.ExportProject(ctx, a.cfg.GitlabProjectID)
	}
	return nil
}

// SetGitlabEndpoint sets the gitlab endpoint
func (a *App) SetGitlabEndpoint(gitlabApiEndpoint string) {
	a.gitlabService.SetGitlabEndpoint(gitlabApiEndpoint)
}

// SetToken sets the gitlab token
func (a *App) SetToken(token string) {
	a.gitlabService.SetToken(token)
}

// SetHttpClient sets the http client
func (a *App) SetHttpClient(httpClient *http.Client) {
	a.gitlabService.SetHttpClient(httpClient)
}

// ExportGroup will export all projects of the group
func (a *App) ExportGroup(ctx context.Context) error {
	projects, err := a.gitlabService.GetEveryProjectsOfGroup(a.cfg.GitlabGroupID)
	if err != nil {
		return err
	}
	eg := errgroup.Group{}
	for project := range projects {
		if !projects[project].Archived {
			eg.Go(func() error {
				err = a.ExportProject(ctx, projects[project].Id)
				if err != nil {
					a.log.Error("error occured during backup", "project name", projects[project].Name, "error", err.Error())
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
		return fmt.Errorf("errors occured during backup")
	}
	return nil
}

// ExportProject exports the project of the given ID
func (a *App) ExportProject(ctx context.Context, projectID int) error {
	project, err := a.gitlabService.GetProject(projectID)
	if err != nil {
		return err
	}
	// call prebackup hook
	if a.cfg.Hooks.HasPreBackup() {
		a.log.Info("SaveProject (call prebackup hook)", "project name", project.Name)
		err = a.cfg.Hooks.ExecutePreBackup()
		if err != nil {
			return err
		}
	}
	archivePath := fmt.Sprintf("%s%s%s-%d.tar.gz", a.cfg.TmpDir, string(os.PathSeparator), project.Name, project.Id)
	err = a.gitlabService.ExportProject(&project, archivePath)
	if err != nil {
		return err
	}
	// call postbackup hook
	if a.cfg.Hooks.HasPostBackup() {
		a.log.Info("SaveProject (call postbackup hook)", "archivePath", archivePath)
		err = a.cfg.Hooks.ExecutePostBackup(archivePath)
		if err != nil {
			return err
		}
	}
	err = a.StoreArchive(archivePath)
	if err != nil {
		return err
	}
	a.log.Info("project succesfully exported", "project", project.Name)
	return nil
}

// StoreArchive stores the archive
func (a *App) StoreArchive(archiveFilePath string) error {
	err := a.storage.SaveFile(context.TODO(), archiveFilePath, filepath.Base(archiveFilePath))
	os.Remove(archiveFilePath)
	if err != nil {
		return err
	}
	return nil
}
