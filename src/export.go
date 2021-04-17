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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/sgaunet/gitlab-backup/gitlabProject"
)

// Part of the response of GITLAB API to export a project
type respGitlabExport struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	ExportStatus string `json:"export_status"`
}

func askExportForProject(projectID int) (int, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%d/export", os.Getenv("GITLAB_URI"), projectID)
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

func waitForExport(projectID int) (gitlabExport respGitlabExport, err error) {
	for gitlabExport.ExportStatus != "finished" {
		// TODO : Set a timeout to avoid to wait forever
		gitlabExport, err = getStatusExport(projectID)
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

func getStatusExport(projectID int) (res respGitlabExport, err error) {
	url := fmt.Sprintf("%s/api/v4/projects/%d/export", os.Getenv("GITLAB_URI"), projectID)
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

func getProjectsLst(groupID int) (res []gitlabProject.GitlabProject, err error) {
	url := fmt.Sprintf("%s/api/v4/groups/%d/projects", os.Getenv("GITLAB_URI"), groupID)
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
