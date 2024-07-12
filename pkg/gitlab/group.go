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
func (s *GitlabService) GetSubgroupsLst(groupID int) (res []GitlabGroup, err error) {
	url := fmt.Sprintf("%s/groups/%d/subgroups?per_page=20&order_by=id&sort=asc&pagination=keyset", s.gitlabApiEndpoint, groupID)
	return s.retrieveSubgroups(url)
}

// GetEveryProjectsOfGroup returns the list of every projects of the group and subgroups
func (s *GitlabService) GetEveryProjectsOfGroup(groupID int) (res []GitlabProject, err error) {
	subgroups, err := s.GetSubgroupsLst(groupID)
	if err != nil {
		return res, fmt.Errorf("got error when listing subgroups of %d (%s)", groupID, err.Error())
	}
	for _, group := range subgroups {
		projects, err := s.GetProjectsLst(group.Id)
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
	projects, err := s.GetProjectsLst(groupID)
	if err != nil {
		return res, err
	}
	res = append(res, projects...)
	return res, nil
}

// GetProjectsLst returns the list of projects of the group
func (s *GitlabService) GetProjectsLst(groupID int) (res []GitlabProject, err error) {
	// add pagination
	// there is a pagination parameter to set to "keyset" value
	// (https://docs.gitlab.com/ee/api/rest/index.html#pagination)
	// per_page can be set between 20 and 100
	// order_by and sort must be set also
	// order_by=id&sort=asc
	url := fmt.Sprintf("%s/groups/%d/projects?per_page=20&order_by=id&sort=asc&pagination=keyset", s.gitlabApiEndpoint, groupID)
	res, err = s.retrieveProjects(url)
	return res, err
}

// retrieveProjects returns the list of projects
func (s *GitlabService) retrieveProjects(url string) (res []GitlabProject, err error) {
	resp, err := s.get(url)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	var jsonResponse []GitlabProject
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		var errMsg ErrorMessage
		if err := json.Unmarshal(body, &errMsg); err != nil {
			return res, fmt.Errorf("error unmarshalling json: %s", err.Error())
		}
		return res, fmt.Errorf("error retrieving projects: %s", errMsg.Message)
	}

	// check if response header contains a link to the next page
	// if yes, recursively call retrieveProjects with the next page url
	// and append the result to the current response
	if link := resp.Header.Get("Link"); link != "" {
		// link is formatted like this:
		// <https://gitlab.com/api/v4/groups/1234/projects?page=2&per_page=100>; rel="next"
		// we only need the next page url
		// so we split the string with the ; separator
		// and take the first element
		nextPageUrl := getNextLink(link)
		if nextPageUrl != "" {
			nextPageProjects, err := s.retrieveProjects(nextPageUrl)
			if err != nil {
				return res, err
			}
			jsonResponse = append(jsonResponse, nextPageProjects...)
		}
	}
	return jsonResponse, err
}

// retrieveSubgroups returns the list of subgroups
func (s *GitlabService) retrieveSubgroups(url string) (res []GitlabGroup, err error) {
	resp, err := s.get(url)
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
		var errMsg ErrorMessage
		if err := json.Unmarshal(body, &errMsg); err != nil {
			return res, fmt.Errorf("error unmarshalling json: %s", err.Error())
		}
		return res, fmt.Errorf("error retrieving subgroups: %s", errMsg.Message)
	}

	// check if response header contains a link to the next page
	// if yes, recursively call retrieveProjects with the next page url
	// and append the result to the current response
	if link := resp.Header.Get("Link"); link != "" {
		// link is formatted like this:
		// "<https://gitlab.com/api/v4/groups/6939159/subgroups?id=6939159&order_by=id&owned=false&page=1&pagination=keyset&per_page=20&sort=asc&statistics=false&with_custom_attributes=false>; rel=\"first\", <https://gitlab.com/api/v4/groups/6939159/subgroups?id=6939159&order_by=id&owned=false&page=1&pagination=keyset&per_page=20&sort=asc&statistics=false&with_custom_attributes=false>; rel=\"last\""
		// we only need the next page url
		// so we split the string with the ; separator
		// and take the first element
		// nextPageUrl := strings.Split(link, ";")[0]
		nextPageUrl := getNextLink(link)
		if nextPageUrl != "" {
			nextPageGroups, err := s.retrieveSubgroups(nextPageUrl)
			if err != nil {
				return res, err
			}
			jsonResponse = append(jsonResponse, nextPageGroups...)
		}
	}
	return jsonResponse, err
}
