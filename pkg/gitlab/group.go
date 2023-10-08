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
	"fmt"
	"io"
)

type GitlabGroup struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// GetSubgroupsLst returns the list of subgroups of the group
func (g *GitlabGroup) GetSubgroupsLst() (res []GitlabGroup, err error) {
	s := NewGitlabService()
	url := fmt.Sprintf("groups/%d/subgroups", g.Id)
	resp, err := s.Get(url)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	var jsonResponse []GitlabGroup
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return res, err
	}
	// Loop for every subgroups
	for _, value := range jsonResponse {
		log.Info("Get subgroup list", "subgroup", value.Id)
		subgroup, err := s.GetGroup(value.Id)
		if err != nil {
			return res, err
		}
		res = append(res, value)
		recursiveGroups, err := subgroup.GetSubgroupsLst()
		if err != nil {
			return res, err
		}
		res = append(res, recursiveGroups...)
	}
	return res, err
}

// GetEveryProjectsOfGroup returns the list of every projects of the group and subgroups
func (g *GitlabGroup) GetEveryProjectsOfGroup() (res []GitlabProject, err error) {
	subgroups, err := g.GetSubgroupsLst()
	if err != nil {
		return res, fmt.Errorf("got error when listing subgroups of %d (%s)", g.Id, err.Error())
	}
	for _, group := range subgroups {
		projects, err := group.GetProjectsLst()
		if err != nil {
			return res, fmt.Errorf("got error when listing projects of %d (%s)", group.Id, err.Error())
		}
		for _, project := range projects {
			if !project.Archived {
				log.Info("GetEveryProjectsOfGroup", "projectName", project.Name)
				res = append(res, project)
			}
		}
	}
	projects, err := g.GetProjectsLst()
	if err != nil {
		return res, err
	}
	res = append(res, projects...)
	return res, nil
}

// GetProjectsLst returns the list of projects of the group
func (g *GitlabGroup) GetProjectsLst() (res []GitlabProject, err error) {
	var respGitlab []GitlabProject
	s := NewGitlabService()
	url := fmt.Sprintf("groups/%d/projects", g.Id)
	resp, err := s.Get(url)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &respGitlab); err != nil {
		return res, err
	}
	res = append(res, respGitlab...)
	return res, err
}
