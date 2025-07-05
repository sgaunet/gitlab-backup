package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	// ErrExportTimeout is returned when timeout occurs waiting for GitLab to start project export.
	ErrExportTimeout = errors.New("timeout waiting for gitlab to start the export project")
	// ErrRateLimit is returned when rate limit is exceeded.
	ErrRateLimit = errors.New("rate limit error")
)

const (
	// ExportCheckIntervalSeconds defines the interval between export status checks.
	ExportCheckIntervalSeconds = 5
	// MaxExportRetries defines the maximum number of export retries.
	MaxExportRetries = 5
)

// Project represents a Gitlab project
// https://docs.gitlab.com/ee/api/projects.html
// struct fields are not exhaustive - most of them won't be used.
type Project struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Archived     bool   `json:"archived"`
	ExportStatus string `json:"export_status"`
}

// askExport asks gitlab to export the project.
func (s *Service) askExport(ctx context.Context, projectID int) (bool, error) {
	resp, err := s.client.ProjectImportExport().ScheduleExport(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("failed to make export request: %w", err)
	}

	// 202 means that gitlab has accepted request
	return resp.StatusCode == http.StatusAccepted, nil
}

// waitForExport waits for gitlab to finish the export.
func (s *Service) waitForExport(ctx context.Context, projectID int) error {
	nbTries := 0
loop:
	for nbTries < MaxExportRetries {
		// !TODO : Set a timeout to avoid to wait forever
		exportStatus, err := s.getStatusExport(ctx, projectID)
		if err != nil {
			return err
		}
		switch exportStatus {
		case "none":
			nbTries++
			log.Warn("no export in progress", "projectID", projectID)
		case "finished":
			break loop
		default:
			log.Info("wait after gitlab to get the archive", "projectID", projectID)
		}
		time.Sleep(ExportCheckIntervalSeconds * time.Second)
	}
	if nbTries == MaxExportRetries {
		return fmt.Errorf("%w %d", ErrExportTimeout, projectID)
	}
	return nil
}

// getStatusExport returns the status of the export.
func (s *Service) getStatusExport(ctx context.Context, projectID int) (string, error) {
	exportStatus, _, err := s.client.ProjectImportExport().ExportStatus(projectID, gitlab.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to get export status: %w", err)
	}
	return exportStatus.ExportStatus, nil
}

// ExportProject exports the project to the given archive file path.
func (s *Service) ExportProject(ctx context.Context, project *Project, archiveFilePath string) error {
	var gitlabAcceptedRequest bool
	if project.Archived {
		log.Warn("SaveProject", "project name", project.Name, "is archived, skip it")
		return nil
	}
	err := s.rateLimitExportAPI.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRateLimit, err)
	}
	for !gitlabAcceptedRequest {
		gitlabAcceptedRequest, err = s.askExport(ctx, project.ID)
		if err != nil {
			return err
		}
	}
	log.Info("SaveProject (gitlab is creating the archive)", "project name", project.Name)
	err = s.waitForExport(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("failed to export project %s: %w", project.Name, err)
	}
	log.Info("SaveProject (gitlab has created the archive, download is beginning)", "project name", project.Name)
	err = s.downloadProject(ctx, project.ID, archiveFilePath)
	if err != nil {
		return err
	}
	return nil
}

// downloadProject downloads the project and save the archive to the given path.
func (s *Service) downloadProject(ctx context.Context, projectID int, tmpFilePath string) error {
	err := s.rateLimitDownloadAPI.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRateLimit, err)
	}

	tmpFile := tmpFilePath + ".tmp"

	// Download the export using the official client
	data, _, err := s.client.ProjectImportExport().ExportDownload(projectID, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to download export: %w", err)
	}

	log.Debug("downloadProject", "tmpFile", tmpFile)
	log.Debug("downloadProject", "tmpFilePath", tmpFilePath)
	log.Debug("downloadProject", "projectID", projectID)
	log.Debug("downloadProject", "dataSize", len(data))

	//nolint:gosec,mnd // G304: File creation is intentional for download functionality
	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write temporary file %s: %w", tmpFile, err)
	}

	if err = os.Rename(tmpFile, tmpFilePath); err != nil {
		return fmt.Errorf("failed to rename temporary file %s to %s: %w", tmpFile, tmpFilePath, err)
	}
	return nil
}
