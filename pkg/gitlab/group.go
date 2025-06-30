package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Group represents a Gitlab group
// https://docs.gitlab.com/ee/api/groups.html
// struct fields are not exhaustive - most of them won't be used.
type Group struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GetSubgroups returns the list of subgroups of the group.
// It's a recursive function that will return all subgroups of the group.
func (s *Service) GetSubgroups(ctx context.Context, groupID int) ([]Group, error) {
	url := fmt.Sprintf("%s/groups/%d/subgroups?per_page=20&order_by=id&sort=asc&pagination=keyset",
		s.gitlabAPIEndpoint, groupID)
	subgroups, err := s.retrieveSubgroups(ctx, url)
	if err != nil {
		return nil, err
	}
	res := append([]Group{}, subgroups...)
	for _, group := range subgroups {
		sub, err := s.GetSubgroups(ctx, group.ID)
		if err != nil {
			return nil, fmt.Errorf("got error when listing subgroups of %d: %w", group.ID, err)
		}
		res = append(res, sub...)
	}
	return res, nil
}

// GetProjectsOfGroup returns the list of every projects of the group and subgroups.
func (s *Service) GetProjectsOfGroup(ctx context.Context, groupID int) ([]Project, error) {
	// First get all subgroups recursively
	subgroups, err := s.GetSubgroups(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("got error when listing subgroups of %d: %w", groupID, err)
	}

	var res []Project
	// Get projects for each subgroup
	for _, group := range subgroups {
		projects, err := s.GetProjectsLst(ctx, group.ID)
		if err != nil {
			return nil, fmt.Errorf("got error when listing projects of %d: %w", group.ID, err)
		}
		// Filter out archived projects
		for _, project := range projects {
			if !project.Archived {
				res = append(res, project)
			}
		}
	}

	// Get projects for the main group
	projects, err := s.GetProjectsLst(ctx, groupID)
	if err != nil {
		return nil, err
	}
	// Filter out archived projects from the main group as well
	for _, project := range projects {
		if !project.Archived {
			res = append(res, project)
		}
	}

	return res, nil
}

// GetProjectsLst returns the list of projects of the group.
func (s *Service) GetProjectsLst(ctx context.Context, groupID int) ([]Project, error) {
	// add pagination
	// there is a pagination parameter to set to "keyset" value
	// (https://docs.gitlab.com/ee/api/rest/index.html#pagination)
	// per_page can be set between 20 and 100
	// order_by and sort must be set also
	// order_by=id&sort=asc
	url := fmt.Sprintf("%s/groups/%d/projects?per_page=20&order_by=id&sort=asc&pagination=keyset",
		s.gitlabAPIEndpoint, groupID)
	return s.retrieveProjects(ctx, url)
}

// retrievePaginatedData handles paginated API responses and returns all items.
func (s *Service) retrievePaginatedData(ctx context.Context, url string) ([]byte, error) {
	resp, err := s.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// check if response header contains a link to the next page
	// if yes, recursively call retrievePaginatedData with the next page url
	// and append the result to the current response
	link := resp.Header.Get("Link")
	if link == "" {
		return body, nil
	}
	
	nextPageURL := getNextLink(link)
	if nextPageURL == "" {
		return body, nil
	}
	
	nextPageData, err := s.retrievePaginatedData(ctx, nextPageURL)
	if err != nil {
		return nil, err
	}
	
	// Merge JSON arrays
	var currentPage, nextPage []json.RawMessage
	if err := json.Unmarshal(body, &currentPage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal current page: %w", err)
	}
	if err := json.Unmarshal(nextPageData, &nextPage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal next page: %w", err)
	}
	currentPage = append(currentPage, nextPage...)
	body, err = json.Marshal(currentPage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged pages: %w", err)
	}
	return body, nil
}

// retrieveProjects returns the list of projects.
func (s *Service) retrieveProjects(ctx context.Context, url string) ([]Project, error) {
	body, err := s.retrievePaginatedData(ctx, url)
	if err != nil {
		return nil, err
	}
	var jsonResponse []Project
	if err = json.Unmarshal(body, &jsonResponse); err != nil {
		// If the response is an error message, unmarshal it
		return nil, UnmarshalErrorMessage(body)
	}
	return jsonResponse, nil
}

// retrieveSubgroups returns the list of subgroups.
func (s *Service) retrieveSubgroups(ctx context.Context, url string) ([]Group, error) {
	body, err := s.retrievePaginatedData(ctx, url)
	if err != nil {
		return nil, err
	}
	var jsonResponse []Group
	if err = json.Unmarshal(body, &jsonResponse); err != nil {
		// If the response is an error message, unmarshal it
		return nil, UnmarshalErrorMessage(body)
	}
	return jsonResponse, nil
}
