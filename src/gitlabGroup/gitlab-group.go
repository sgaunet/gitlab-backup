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

package gitlabGroup

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/sgaunet/gitlab-backup/gitlabProject"
)

func New(groupID int) (res GitlabGroup, err error) {
	url := fmt.Sprintf("%s/api/v4/groups/%d", os.Getenv("GITLAB_URI"), groupID)
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

func (g *GitlabGroup) GetSubgroupsLst() (res []GitlabGroup, err error) {
	url := fmt.Sprintf("%s/api/v4/groups/%d/subgroups", os.Getenv("GITLAB_URI"), g.Id)
	//fmt.Println(url)
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

	var jsonResponse []GitlabGroup
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return res, err
	}

	// Loop for every subgroups
	for _, value := range jsonResponse {
		fmt.Printf("Subgroup %d\n", value.Id)
		subgroup, _ := New(value.Id)
		//getProjectsLst(value.Id)
		res = append(res, value)
		recursiveGroups, err := subgroup.GetSubgroupsLst()
		if err != nil {
			fmt.Printf("Got an error when trying to get the subgroups of %d (%s)\n", value.Id, err.Error())
		} else {
			for _, newGroup := range recursiveGroups {
				res = append(res, newGroup)
			}
		}
	}

	return res, err
}

func (g *GitlabGroup) GetEveryProjectsOfGroup() (res []gitlabProject.GitlabProject, err error) {
	subgroups, err := g.GetSubgroupsLst()
	if err != nil {
		fmt.Printf("Got error when listing subgroups of %d (%s)\n", g.Id, err.Error())
		os.Exit(1)
	}
	for _, group := range subgroups {
		//fmt.Println("ID =>", v.Id)
		projects, err := group.GetProjectsLst()
		if err != nil {
			fmt.Printf("Got error when listing projects of %d (%s)\n", g.Id, err.Error())
		} else {
			for _, project := range projects {
				fmt.Println("project ID:", project.Name)
				res = append(res, project)
			}
		}
	}

	projects, err := g.GetProjectsLst()
	if err != nil {
		fmt.Printf("Got error when listing projects of %d (%s)\n", g.Id, err.Error())
	} else {
		for _, project := range projects {
			fmt.Println("project:", project.Name)
			res = append(res, project)
		}
	}
	return res, err
}

func (g *GitlabGroup) GetProjectsLst() (res []gitlabProject.GitlabProject, err error) {
	url := fmt.Sprintf("%s/api/v4/groups/%d/projects", os.Getenv("GITLAB_URI"), g.Id)
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
