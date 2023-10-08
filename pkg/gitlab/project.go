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
func (p *GitlabProject) askExport() (acceptedRequest bool, err error) {
	s := NewGitlabService()
	path := fmt.Sprintf("projects/%d/export", p.Id)
	resp, err := s.Post(path)
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
func (p *GitlabProject) waitForExport() (gitlabExport GitlabProject, err error) {
	for gitlabExport.ExportStatus != "finished" {
		// !TODO : Set a timeout to avoid to wait forever
		gitlabExport, err = p.getStatusExport()
		if err != nil {
			return gitlabExport, err
		}
		switch gitlabExport.ExportStatus {
		case "none":
			return gitlabExport, errors.New("project not exported")
		default:
			log.Info("wait after gitlab to get the archive", "project name", gitlabExport.Name)
		}
		time.Sleep(5 * time.Second)
	}
	return gitlabExport, nil
}

// getStatusExport returns the status of the export
func (p *GitlabProject) getStatusExport() (res GitlabProject, err error) {
	s := NewGitlabService()
	url := fmt.Sprintf("projects/%d/export", p.Id)
	resp, err := s.Get(url)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(body, &res)
	return res, err
}

// downloadProject downloads the project and save the archive to the given path
func (p *GitlabProject) downloadProject(tmpFilePath string) error {
	s := NewGitlabService()
	tmpFile := tmpFilePath + ".tmp"
	url := fmt.Sprintf("projects/%d/export/download", p.Id)
	resp, err := s.Get(url)
	if err != nil {
		return err
	}
	log.Debug("downloadProject", "url", url)
	log.Debug("downloadProject", "tmpFile", tmpFile)
	log.Debug("downloadProject", "tmpFilePath", tmpFilePath)
	log.Debug("downloadProject", "ContentLength", resp.ContentLength)
	log.Debug("downloadProject", "StatusCode", resp.StatusCode)
	log.Debug("downloadProject", "projectID", p.Id)
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
func (p *GitlabProject) Export(tmpdir string) (err error) {
	var gitlabAcceptedRequest bool
	if p.Archived {
		log.Warn("SaveProject", "project name", p.Name, "is archived, skip it")
		return nil
	}
	for !gitlabAcceptedRequest {
		gitlabAcceptedRequest, err = p.askExport()
		if err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}
	log.Info("SaveProject (gitlab is creating the archive)", "project name", p.Name)
	_, err = p.waitForExport()
	if err != nil {
		return fmt.Errorf("failed to export project %s (%s)", p.Name, err.Error())
	}
	log.Info("SaveProject (gitlab has created the archive, download is beginning)", "project name", p.Name)
	time.Sleep(5 * time.Second)
	err = p.downloadProject(p.ExportedArchivePath(tmpdir))
	if err != nil {
		return err
	}
	return nil
}

func (p *GitlabProject) ExportedArchivePath(tmpdir string) string {
	return fmt.Sprintf("%s%s%s-%d.tar.gz", tmpdir, string(os.PathSeparator), p.Name, p.Id)
}
