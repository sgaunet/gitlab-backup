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

package gitlabProject

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

func New(projectID int) (res GitlabProject, err error) {
	url := fmt.Sprintf("%s/api/v4/projects/%d", os.Getenv("GITLAB_URI"), projectID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return res, err
	}
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return res, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return res, err
	}

	return res, err
}

func (p *GitlabProject) askExportForProject() (int, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%d/export", os.Getenv("GITLAB_URI"), p.Id)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	// 202 means that gitlab has accepted request
	return resp.StatusCode, nil
}

func (p *GitlabProject) waitForExport() (gitlabExport respGitlabExport, err error) {
	for gitlabExport.ExportStatus != "finished" {
		// TODO : Set a timeout to avoid to wait forever
		gitlabExport, err = p.getStatusExport()
		if err != nil {
			return gitlabExport, err
		}
		//fmt.Println(gitlabExport.Name, gitlabExport.ExportStatus)
		switch gitlabExport.ExportStatus {
		case "none":
			return gitlabExport, errors.New("Project not exported")
		default:
			fmt.Printf("%s : Wait after gitlab for the export\n", gitlabExport.Name)
		}
		time.Sleep(20 * time.Second)
	}
	return gitlabExport, nil
}

func (p *GitlabProject) getStatusExport() (res respGitlabExport, err error) {
	url := fmt.Sprintf("%s/api/v4/projects/%d/export", os.Getenv("GITLAB_URI"), p.Id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return res, err
	}
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return res, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(body, &res)
	return res, nil
}

func (p *GitlabProject) downloadProject(dirToSaveFile string) error {
	tmpFile := dirToSaveFile + string(os.PathSeparator) + p.Name + ".tar.gz.tmp"
	finalFile := dirToSaveFile + string(os.PathSeparator) + p.Name + ".tar.gz"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v4/projects/%d/export/download", os.Getenv("GITLAB_URI"), p.Id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	// fmt.Println("Taille: ", resp.ContentLength)
	// fmt.Printf("Downloading %s\n", project.Name)

	if _, err = io.Copy(out, resp.Body); err != nil {
		out.Close()
		return err
	}
	out.Close()

	if err = os.Rename(tmpFile, finalFile); err != nil {
		return err
	}
	return nil
}

func (p *GitlabProject) SaveProjectOnDisk(dirpath string, wg *sync.WaitGroup) (err error) {
	defer wg.Done()
	statuscode := 0
	// fmt.Println("\tAsk export for project", project.Name)
	for statuscode != 202 {
		fmt.Printf("%s : Ask gitlab to export a backup\n", p.Name)
		statuscode, err = p.askExportForProject()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		time.Sleep(20 * time.Second)
	}
	fmt.Printf("%s : Gitlab is creating the archive\n", p.Name)
	_, err = p.waitForExport()
	if err != nil {
		fmt.Printf("%s: Export failed, reason: %s\n", p.Name, err.Error())
		return errors.New("Failed ...")
	}
	fmt.Printf("%s : Gitlab has created the archive, download is beginning\n", p.Name)
	p.downloadProject(dirpath)
	fmt.Printf("%s : Succesfully exported\n", p.Name)
	return nil
}
