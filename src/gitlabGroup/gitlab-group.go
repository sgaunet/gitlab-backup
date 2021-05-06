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
	"os"

	"github.com/sgaunet/gitlab-backup/gitlabProject"
	"github.com/sgaunet/gitlab-backup/gitlabRequest"
)

func New(groupID int) (res gitlabGroup, err error) {
	url := fmt.Sprintf("groups/%d", groupID)
	_, body, err := gitlabRequest.Request(url)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return res, err
	}

	return res, err
}

func (g gitlabGroup) GetID() int {
	return g.Id
}

func (g gitlabGroup) GetSubgroupsLst() (res []gitlabGroup, err error) {
	url := fmt.Sprintf("groups/%d/subgroups", g.Id)
	_, body, err := gitlabRequest.Request(url)
	if err != nil {
		return res, err
	}
	var jsonResponse []gitlabGroup
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return res, err
	}

	// Loop for every subgroups
	for _, value := range jsonResponse {
		fmt.Printf("Subgroup %d\n", value.GetID())
		subgroup, _ := New(value.GetID())
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

func (g gitlabGroup) GetEveryProjectsOfGroup() (res []gitlabProject.GitlabProject, err error) {
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
				fmt.Println("Project :", project.GetName())
				res = append(res, project)
			}
		}
	}

	projects, err := g.GetProjectsLst()
	if err != nil {
		fmt.Printf("Got error when listing projects of %d (%s)\n", g.Id, err.Error())
	} else {
		for _, project := range projects {
			fmt.Println("Project:", project.GetName())
			res = append(res, project)
		}
	}
	return res, err
}

func (g gitlabGroup) GetProjectsLst() (res []gitlabProject.GitlabProject, err error) {
	var respGitlab []respGitlabProject
	url := fmt.Sprintf("groups/%d/projects", g.Id)
	_, body, err := gitlabRequest.Request(url)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(body, &respGitlab); err != nil {
		return res, err
	}

	for _, project := range respGitlab {
		gProject, err := gitlabProject.New(project.Id)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			res = append(res, gProject)
		}
	}

	return res, err
}
