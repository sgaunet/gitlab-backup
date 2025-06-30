package gitlab

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
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
	opt := &gitlab.ListSubGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20, //nolint:mnd // GitLab API pagination default
		},
		OrderBy: gitlab.Ptr("id"),
		Sort:    gitlab.Ptr("asc"),
	}
	
	var allSubgroups []Group
	for {
		subgroups, resp, err := s.client.Groups.ListSubGroups(groupID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("error listing subgroups: %w", err)
		}
		
		// Convert to our Group type
		for _, sg := range subgroups {
			allSubgroups = append(allSubgroups, Group{
				ID:   sg.ID,
				Name: sg.Name,
			})
		}
		
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	
	// Recursively get subgroups
	res := append([]Group{}, allSubgroups...)
	for _, group := range allSubgroups {
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
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20, //nolint:mnd // GitLab API pagination default
		},
		OrderBy: gitlab.Ptr("id"),
		Sort:    gitlab.Ptr("asc"),
	}
	
	var allProjects []Project
	for {
		projects, resp, err := s.client.Groups.ListGroupProjects(groupID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("error listing group projects: %w", err)
		}
		
		// Convert to our Project type
		for _, p := range projects {
			allProjects = append(allProjects, Project{
				ID:           p.ID,
				Name:         p.Name,
				Archived:     p.Archived,
				ExportStatus: "", // ExportStatus not available in project struct
			})
		}
		
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	
	return allProjects, nil
}

