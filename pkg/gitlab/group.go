package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
)

// GitlabGroup represents a Gitlab group
// https://docs.gitlab.com/ee/api/groups.html
// struct fields are not exhaustive - most of them won't be used
type GitlabGroup struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// GetSubgroups returns the list of subgroups of the group
// It's a recursive function that will return all subgroups of the group
func (s *GitlabService) GetSubgroups(groupID int) (res []GitlabGroup, err error) {
	url := fmt.Sprintf("%s/groups/%d/subgroups?per_page=20&order_by=id&sort=asc&pagination=keyset", s.gitlabApiEndpoint, groupID)
	subgroups, err := s.retrieveSubgroups(url)
	if err != nil {
		return res, err
	}
	res = append(res, subgroups...)
	for _, group := range subgroups {
		sub, err := s.GetSubgroups(group.Id)
		if err != nil {
			return res, fmt.Errorf("got error when listing subgroups of %d (%s)", group.Id, err.Error())
		}
		res = append(res, sub...)
	}
	return res, nil
}

// GetProjectsOfGroup returns the list of every projects of the group and subgroups
func (s *GitlabService) GetProjectsOfGroup(groupID int) (res []GitlabProject, err error) {
	// First get all subgroups recursively
	subgroups, err := s.GetSubgroups(groupID)
	if err != nil {
		return res, fmt.Errorf("got error when listing subgroups of %d (%s)", groupID, err.Error())
	}

	// Get projects for each subgroup
	for _, group := range subgroups {
		projects, err := s.GetProjectsLst(group.Id)
		if err != nil {
			return res, fmt.Errorf("got error when listing projects of %d (%s)", group.Id, err.Error())
		}
		// Filter out archived projects
		for _, project := range projects {
			if !project.Archived {
				res = append(res, project)
			}
		}
	}

	// Get projects for the main group
	projects, err := s.GetProjectsLst(groupID)
	if err != nil {
		return res, err
	}
	// Filter out archived projects from the main group as well
	for _, project := range projects {
		if !project.Archived {
			res = append(res, project)
		}
	}

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
	if err = json.Unmarshal(body, &jsonResponse); err != nil {
		// If the response is an error message, unmarshal it
		return res, UnmarshalErrorMessage(body)
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
	// Unmarshal the response
	if err = json.Unmarshal(body, &jsonResponse); err != nil {
		// If the response is an error message, unmarshal it
		return res, UnmarshalErrorMessage(body)
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
