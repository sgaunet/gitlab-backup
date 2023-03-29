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

	"github.com/sgaunet/gitlab-backup/gitlabRequest"
	log "github.com/sirupsen/logrus"
)

func New(projectID int) (res gitlabProject, err error) {
	var project respGitlabExport
	url := fmt.Sprintf("projects/%d", projectID)
	_, body, err := gitlabRequest.Request(url)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(body, &project); err != nil {
		return res, err
	}
	return gitlabProject{Id: project.Id, Name: project.Name}, err
}

func (p gitlabProject) askExportForProject() (int, error) {
	url := fmt.Sprintf("%s/api/%s/projects/%d/export", os.Getenv("GITLAB_URI"), gitlabRequest.GitlabApiVersion, p.Id)
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	log.Debugf(string(body))
	// 202 means that gitlab has accepted request
	return resp.StatusCode, nil
}

func (p gitlabProject) waitForExport() (gitlabExport respGitlabExport, err error) {
	for gitlabExport.ExportStatus != "finished" {
		// TODO : Set a timeout to avoid to wait forever
		gitlabExport, err = p.getStatusExport()
		if err != nil {
			return gitlabExport, err
		}
		//fmt.Println(gitlabExport.Name, gitlabExport.ExportStatus)
		switch gitlabExport.ExportStatus {
		case "none":
			return gitlabExport, errors.New("project not exported")
		default:
			log.Infof("%s : Wait after gitlab for the export\n", gitlabExport.Name)
		}
		time.Sleep(5 * time.Second)
	}
	return gitlabExport, nil
}

func (p gitlabProject) getStatusExport() (res respGitlabExport, err error) {
	url := fmt.Sprintf("projects/%d/export", p.Id)
	_, body, err := gitlabRequest.Request(url)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(body, &res)
	return res, err
}

func (p gitlabProject) downloadProject(dirToSaveFile string) error {
	tmpFile := dirToSaveFile + string(os.PathSeparator) + p.Name + ".tar.gz.tmp"
	finalFile := fmt.Sprintf("%s%s%s-%d.tar.gz", dirToSaveFile, string(os.PathSeparator), p.Name, p.Id)
	url := fmt.Sprintf("%s/api/%s/projects/%d/export/download", os.Getenv("GITLAB_URI"), gitlabRequest.GitlabApiVersion, p.Id)
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
	// fmt.Println(resp.StatusCode)
	if resp.StatusCode != 200 {
		errMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		return fmt.Errorf("error while downloading the project (%s)", string(errMsg))
	}

	// fmt.Println("Taille: ", resp.ContentLength)
	// fmt.Printf("Downloading %s\n", project.Name)
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}
	if err = os.Rename(tmpFile, finalFile); err != nil {
		return err
	}
	return nil
}

func (p gitlabProject) SaveProjectOnDisk(dirpath string, wg *sync.WaitGroup) (err error) {
	defer wg.Done()
	statuscode := 0
	// fmt.Println("\tAsk export for project", project.Name)
	for statuscode != 202 {
		log.Infof("%s : Ask gitlab to export a backup\n", p.Name)
		statuscode, err = p.askExportForProject()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
	}
	log.Infof("%s : Gitlab is creating the archive\n", p.Name)
	_, err = p.waitForExport()
	if err != nil {
		log.Errorf("%s: Export failed, reason: %s\n", p.Name, err.Error())
		return fmt.Errorf("failed to export project %s (%s)", p.Name, err.Error())
	}
	log.Infof("%s : Gitlab has created the archive, download is beginning\n", p.Name)
	err = p.downloadProject(dirpath)
	if err != nil {
		log.Errorln(err.Error())
		return err
	}
	log.Infof("%s : Succesfully exported\n", p.Name)

	return nil
}

func (p gitlabProject) GetID() int {
	return p.Id
}

func (p gitlabProject) GetName() string {
	return p.Name
}
