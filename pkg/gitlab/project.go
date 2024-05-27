// gitlab-backup
// Copyright (C) 2021  Sylvain Gaunet

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

type GitlabProject struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Archived     bool   `json:"archived"`
	ExportStatus string `json:"export_status"`
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
loop:
	for {
		// !TODO : Set a timeout to avoid to wait forever
		exportStatus, err := s.getStatusExport(projectID)
		if err != nil {
			return err
		}
		switch exportStatus {
		case "none":
			return errors.New("project not exported")
		case "finished":
			break loop
		default:
			log.Info("wait after gitlab to get the archive", "projectID", projectID)
		}
		time.Sleep(5 * time.Second)
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
	err = json.Unmarshal(body, &res)
	return res.ExportStatus, err
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
