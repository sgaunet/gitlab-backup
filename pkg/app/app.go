package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/sgaunet/gitlab-backup/pkg/storage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/localstorage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/s3storage"
	"github.com/sgaunet/ratelimit"
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

func NewApp(configFile string) (*App, error) {
	var cfg *config.Config
	var err error
	if len(configFile) > 0 {
		cfg, err = config.NewConfigFromFile(configFile)
		if err != nil {
			return nil, err
		}
	} else {
		cfg, err = config.NewConfigFromEnv()
		if err != nil {
			return nil, err
		}
	}
	app := &App{
		cfg:           cfg,
		gitlabService: gitlab.NewGitlabService(),
		log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	gitlab.SetLogger(app.log)
	if cfg.IsS3ConfigValid() {
		app.storage, err = s3storage.NewS3Storage(cfg.S3cfg.Region, cfg.S3cfg.Endpoint, cfg.S3cfg.BucketName, cfg.S3cfg.BucketPath)
		if err != nil {
			return nil, err
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

func (a *App) SetLogger(l Logger) {
	a.log = l
	gitlab.SetLogger(l)

}

func (a *App) Run() error {
	if a.cfg.GitlabGroupID != 0 {
		return a.ExportGroup()
	}
	if a.cfg.GitlabProjectID != 0 {
		return a.ExportProject(a.cfg.GitlabProjectID)
	}
	return nil
}

func (a *App) SetGitlabEndpoint(gitlabApiEndpoint string) {
	a.gitlabService.SetGitlabEndpoint(gitlabApiEndpoint)
}

func (a *App) SetToken(token string) {
	a.gitlabService.SetToken(token)
}

func (a *App) SetHttpClient(httpClient *http.Client) {
	a.gitlabService.SetHttpClient(httpClient)
}

func (a *App) ExportGroup() error {
	var returnErr int
	ctx := context.Background()
	r, _ := ratelimit.New(ctx, 60*time.Second, 1)
	group, err := a.gitlabService.GetGroup(a.cfg.GitlabGroupID)
	if err != nil {
		return err
	}
	projects, err := group.GetEveryProjectsOfGroup()
	if err != nil {
		return err
	}
	for project := range projects {
		r.WaitIfLimitReached()
		if !projects[project].Archived {
			err = a.ExportProject(projects[project].Id)
			if err != nil {
				returnErr = 1
				continue
			}
		} else {
			a.log.Info("project is archived, skip", "project name", projects[project].Name)
		}
	}
	if returnErr != 0 {
		return fmt.Errorf("errors occured during backup")
	}
	return nil
}

func (a *App) ExportProject(projectID int) error {
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
	err = project.Export(a.cfg.TmpDir)
	if err != nil {
		return err
	}
	// call postbackup hook
	if a.cfg.Hooks.HasPostBackup() {
		a.log.Info("SaveProject (call postbackup hook)", "projectExportedArchive", project.ExportedArchivePath(a.cfg.TmpDir))
		err = a.cfg.Hooks.ExecutePostBackup(project.ExportedArchivePath(a.cfg.TmpDir))
		if err != nil {
			return err
		}
	}
	err = a.StoreArchive(project.ExportedArchivePath(a.cfg.TmpDir))
	if err != nil {
		return err
	}
	a.log.Info("project succesfully exported", "project", project.Name)
	return nil
}

func (a *App) StoreArchive(archiveFilePath string) error {
	// f, err := os.Open(archiveFilePath)
	// if err != nil {
	// 	return err
	// }
	// defer f.Close()
	// // get file size
	// fi, err := f.Stat()
	// if err != nil {
	// 	return err
	// }
	err := a.storage.SaveFile(context.TODO(), archiveFilePath, filepath.Base(archiveFilePath))
	os.Remove(archiveFilePath)
	if err != nil {
		return err
	}
	return nil
}
