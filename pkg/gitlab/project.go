package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// GitlabProject represents a Gitlab project
// https://docs.gitlab.com/ee/api/projects.html
// struct fields are not exhaustive - most of them won't be used
type GitlabProject struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Archived     bool   `json:"archived"`
	ExportStatus string `json:"export_status"`
}

// GetProject returns the gitlab project from the given ID
type ProjectAccessToken struct {
	Id          int      `json:"id"`
	Name        string   `json:"name"`
	Revoked     bool     `json:"revoked"`
	CreatedAt   string   `json:"created_at"`
	Scopes      []string `json:"scopes"`
	UserId      int      `json:"user_id"`
	LastUsedAt  string   `json:"last_used_at"`
	Active      bool     `json:"active"`
	ExpiresAt   string   `json:"expires_at"`
	AccessLevel int      `json:"access_level"`
}

// askExport asks gitlab to export the project
func (s *GitlabService) askExport(projectID int) (acceptedRequest bool, err error) {
	url := fmt.Sprintf("%s/projects/%d/export", s.gitlabApiEndpoint, projectID)
	resp, err := s.post(url)
	if err != nil {
		return
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	// 202 means that gitlab has accepted request
	return statusCode == http.StatusAccepted, nil
}

// waitForExport waits for gitlab to finish the export
func (s *GitlabService) waitForExport(projectID int) (err error) {
	nbTries := 0
loop:
	for nbTries < 5 {
		// !TODO : Set a timeout to avoid to wait forever
		exportStatus, err := s.getStatusExport(projectID)
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
		time.Sleep(5 * time.Second)
	}
	if nbTries == 5 {
		return fmt.Errorf("timeout waiting for gitlab to start the export project %d", projectID)
	}
	return nil
}

// getStatusExport returns the status of the export
func (s *GitlabService) getStatusExport(projectID int) (exportStatus string, err error) {
	var res GitlabProject
	url := fmt.Sprintf("%s/projects/%d/export", s.gitlabApiEndpoint, projectID)
	resp, err := s.get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err = json.Unmarshal(body, &res); err != nil {
		// If the response is an error message, unmarshal it
		return "", UnmarshalErrorMessage(body)
	}
	return res.ExportStatus, err
}

// SaveProject saves the project in the given storage
func (s *GitlabService) ExportProject(project *GitlabProject, archiveFilePath string) (err error) {
	var gitlabAcceptedRequest bool
	if project.Archived {
		log.Warn("SaveProject", "project name", project.Name, "is archived, skip it")
		return nil
	}
	err = s.rateLimitExportAPI.Wait(context.Background()) // This is a blocking call. Honors the rate limit
	if err != nil {
		return fmt.Errorf("rate limit error: %v", err)
	}
	for !gitlabAcceptedRequest {
		gitlabAcceptedRequest, err = s.askExport(project.Id)
		if err != nil {
			return err
		}
	}
	log.Info("SaveProject (gitlab is creating the archive)", "project name", project.Name)
	err = s.waitForExport(project.Id)
	if err != nil {
		return fmt.Errorf("failed to export project %s (%s)", project.Name, err.Error())
	}
	log.Info("SaveProject (gitlab has created the archive, download is beginning)", "project name", project.Name)
	err = s.downloadProject(project.Id, archiveFilePath)
	if err != nil {
		return err
	}
	return nil
}

// downloadProject downloads the project and save the archive to the given path
func (s *GitlabService) downloadProject(projectID int, tmpFilePath string) error {
	err := s.rateLimitDownloadAPI.Wait(context.Background()) // This is a blocking call. Honors the rate limit
	if err != nil {
		return fmt.Errorf("rate limit error: %v", err)
	}

	tmpFile := tmpFilePath + ".tmp"
	url := fmt.Sprintf("%s/projects/%d/export/download", s.gitlabApiEndpoint, projectID)
	resp, err := s.get(url)
	if err != nil {
		return err
	}
	log.Debug("downloadProject", "url", url)
	log.Debug("downloadProject", "tmpFile", tmpFile)
	log.Debug("downloadProject", "tmpFilePath", tmpFilePath)
	log.Debug("downloadProject", "ContentLength", resp.ContentLength)
	log.Debug("downloadProject", "StatusCode", resp.StatusCode)
	log.Debug("downloadProject", "projectID", projectID)
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}
	if err = os.Rename(tmpFile, tmpFilePath); err != nil {
		return err
	}
	return nil
}
