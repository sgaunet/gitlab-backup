package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	// ErrExportTimeout is returned when timeout occurs waiting for GitLab to start project export.
	ErrExportTimeout = errors.New("timeout waiting for gitlab to start the export project")
	// ErrRateLimit is returned when rate limit is exceeded.
	ErrRateLimit     = errors.New("rate limit error")
)

const (
	// ExportCheckIntervalSeconds defines the interval between export status checks.
	ExportCheckIntervalSeconds = 5
	// MaxExportRetries defines the maximum number of export retries.
	MaxExportRetries          = 5
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

// ProjectAccessToken represents a GitLab project access token.
type ProjectAccessToken struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Revoked     bool     `json:"revoked"`
	CreatedAt   string   `json:"created_at"`
	Scopes      []string `json:"scopes"`
	UserID      int      `json:"user_id"`
	LastUsedAt  string   `json:"last_used_at"`
	Active      bool     `json:"active"`
	ExpiresAt   string   `json:"expires_at"`
	AccessLevel int      `json:"access_level"`
}

// askExport asks gitlab to export the project.
func (s *Service) askExport(ctx context.Context, projectID int) (bool, error) {
	url := fmt.Sprintf("%s/projects/%d/export", s.gitlabAPIEndpoint, projectID)
	resp, err := s.post(ctx, url)
	if err != nil {
		return false, fmt.Errorf("failed to make export request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	
	// Read and discard response body to allow connection reuse
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
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
	var res Project
	url := fmt.Sprintf("%s/projects/%d/export", s.gitlabAPIEndpoint, projectID)
	resp, err := s.get(ctx, url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if err = json.Unmarshal(body, &res); err != nil {
		// If the response is an error message, unmarshal it
		return "", UnmarshalErrorMessage(body)
	}
	return res.ExportStatus, nil
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
	url := fmt.Sprintf("%s/projects/%d/export/download", s.gitlabAPIEndpoint, projectID)
	resp, err := s.get(ctx, url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	log.Debug("downloadProject", "url", url)
	log.Debug("downloadProject", "tmpFile", tmpFile)
	log.Debug("downloadProject", "tmpFilePath", tmpFilePath)
	log.Debug("downloadProject", "ContentLength", resp.ContentLength)
	log.Debug("downloadProject", "StatusCode", resp.StatusCode)
	log.Debug("downloadProject", "projectID", projectID)
	out, err := os.Create(tmpFile) //nolint:gosec // G304: File creation is intentional for download functionality
	if err != nil {
		return fmt.Errorf("failed to create temporary file %s: %w", tmpFile, err)
	}
	defer func() { _ = out.Close() }()
	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to copy response body to file: %w", err)
	}
	if err = os.Rename(tmpFile, tmpFilePath); err != nil {
		return fmt.Errorf("failed to rename temporary file %s to %s: %w", tmpFile, tmpFilePath, err)
	}
	return nil
}
