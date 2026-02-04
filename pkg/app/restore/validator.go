package restore

import (
	"context"
	"fmt"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	gitlabapi "gitlab.com/gitlab-org/api/client-go"
)

// Validator provides project emptiness validation functionality.
type Validator struct {
	commitsService gitlab.CommitsService
	issuesService  gitlab.IssuesService
	labelsService  gitlab.LabelsService
}

// NewValidator creates a new Validator instance.
func NewValidator(commitsService gitlab.CommitsService, issuesService gitlab.IssuesService, labelsService gitlab.LabelsService) *Validator {
	return &Validator{
		commitsService: commitsService,
		issuesService:  issuesService,
		labelsService:  labelsService,
	}
}

// ValidateProjectEmpty checks if a GitLab project is empty (no commits, issues, or labels).
// It returns EmptinessChecks with detailed information about what exists in the project.
// This is used to ensure a project is empty before restore to avoid overwriting data.
func (v *Validator) ValidateProjectEmpty(ctx context.Context, projectID int64) (*EmptinessChecks, error) {
	checks := &EmptinessChecks{}

	// Check for commits
	commits, _, err := v.commitsService.ListCommits(
		projectID,
		&gitlabapi.ListCommitsOptions{
			ListOptions: gitlabapi.ListOptions{
				PerPage: 1, // Only need to know if any exist
				Page:    1,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}
	checks.HasCommits = len(commits) > 0
	checks.CommitCount = len(commits)

	// Check for issues
	issues, _, err := v.issuesService.ListProjectIssues(
		projectID,
		&gitlabapi.ListProjectIssuesOptions{
			ListOptions: gitlabapi.ListOptions{
				PerPage: 1, // Only need to know if any exist
				Page:    1,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}
	checks.HasIssues = len(issues) > 0
	checks.IssueCount = len(issues)

	// Check for labels
	labels, _, err := v.labelsService.ListLabels(
		projectID,
		&gitlabapi.ListLabelsOptions{
			ListOptions: gitlabapi.ListOptions{
				PerPage: 1, // Only need to know if any exist
				Page:    1,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	checks.HasLabels = len(labels) > 0
	checks.LabelCount = len(labels)

	return checks, nil
}

// buildEmptinessChecks constructs an EmptinessChecks struct from API responses.
// This helper function is used internally to build the checks result.
func buildEmptinessChecks(commits []*gitlabapi.Commit, issues []*gitlabapi.Issue, labels []*gitlabapi.Label) *EmptinessChecks {
	return &EmptinessChecks{
		HasCommits:  len(commits) > 0,
		HasIssues:   len(issues) > 0,
		HasLabels:   len(labels) > 0,
		CommitCount: len(commits),
		IssueCount:  len(issues),
		LabelCount:  len(labels),
	}
}
