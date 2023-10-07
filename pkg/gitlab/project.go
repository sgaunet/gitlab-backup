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
	"sync"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/storage"
)

type GitlabProject struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Archived     bool   `json:"archived"`
	ExportStatus string `json:"export_status"`
}

// askExportForProject asks gitlab to export the project
func (s *GitlabService) askExportForProject(projectID int) (acceptedRequest bool, err error) {
	path := fmt.Sprintf("projects/%d/export", projectID)
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
func (s *GitlabService) waitForExport(projectID int) (gitlabExport GitlabProject, err error) {
	for gitlabExport.ExportStatus != "finished" {
		// !TODO : Set a timeout to avoid to wait forever
		gitlabExport, err = s.getStatusExport(projectID)
		if err != nil {
			return gitlabExport, err
		}
		//fmt.Println(gitlabExport.Name, gitlabExport.ExportStatus)
		switch gitlabExport.ExportStatus {
		case "none":
			return gitlabExport, errors.New("project not exported")
		default:
			s.log.Info("wait after gitlab to get the archive", "project name", gitlabExport.Name)
		}
		time.Sleep(5 * time.Second)
	}
	return gitlabExport, nil
}

// getStatusExport returns the status of the export
func (s *GitlabService) getStatusExport(projectID int) (res GitlabProject, err error) {
	url := fmt.Sprintf("projects/%d/export", projectID)
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
func (s *GitlabService) downloadProject(projectID int, tmpFilePath string) error {
	tmpFile := tmpFilePath + ".tmp"
	url := fmt.Sprintf("projects/%d/export/download", projectID)
	resp, err := s.Get(url)
	if err != nil {
		return err
	}
	s.log.Info("downloadProject", "url", url)
	s.log.Info("downloadProject", "tmpFile", tmpFile)
	s.log.Info("downloadProject", "tmpFilePath", tmpFilePath)
	s.log.Info("downloadProject", "ContentLength", resp.ContentLength)
	s.log.Info("downloadProject", "StatusCode", resp.StatusCode)
	s.log.Info("downloadProject", "projectID", projectID)
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
func (s *GitlabService) SaveProject(wg *sync.WaitGroup, storage storage.Storage, projectID int, tmpdir string) (err error) {
	var gitlabAcceptedRequest bool
	p, err := s.GetProject(projectID)
	if err != nil {
		return err
	}
	tmpFilePath := fmt.Sprintf("%s%s%s-%d.tar.gz", tmpdir, string(os.PathSeparator), p.Name, p.Id)
	defer wg.Done()
	s.log.Info("SaveProject (ask for export)", "project name", p.Name)
	for !gitlabAcceptedRequest {
		gitlabAcceptedRequest, err = s.askExportForProject(p.Id)
		if err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}
	s.log.Info("SaveProject (gitlab is creating the archive)", "project name", p.Name)
	_, err = s.waitForExport(p.Id)
	if err != nil {
		return fmt.Errorf("failed to export project %s (%s)", p.Name, err.Error())
	}
	s.log.Info("SaveProject (gitlab has created the archive, download is beginning)", "project name", p.Name)
	time.Sleep(5 * time.Second)
	err = s.downloadProject(p.Id, tmpFilePath)
	if err != nil {
		return err
	}
	f, err := os.Open(tmpFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// get file size
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	err = storage.SaveFile(context.TODO(), f, p.Name+".tar.gz", fi.Size())
	if err != nil {
		os.Remove(tmpdir + string(os.PathSeparator) + p.Name + ".tar.gz")
		return err
	}
	os.Remove(tmpdir + string(os.PathSeparator) + p.Name + ".tar.gz")
	s.log.Info("SaveProject (succesfully exported)", "project name", p.Name)
	return nil
}

// GetProjects returns information of the project that matches the given ID
func (s *GitlabService) GetProject(projectID int) (res GitlabProject, err error) {
	url := fmt.Sprintf("projects/%d", projectID)
	resp, err := s.Get(url)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return res, err
	}
	return res, err
}
