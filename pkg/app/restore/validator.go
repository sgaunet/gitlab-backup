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
func NewValidator(
	commitsService gitlab.CommitsService,
	issuesService gitlab.IssuesService,
	labelsService gitlab.LabelsService,
) *Validator {
	return &Validator{
		commitsService: commitsService,
		issuesService:  issuesService,
		labelsService:  labelsService,
	}
}

// ValidateProjectEmpty checks if a GitLab project is empty (no commits, issues, or labels).
// It returns EmptinessChecks with detailed information about what exists in the project.
// This is used to ensure a project is empty before restore to avoid overwriting data.
// The context is propagated to GitLab API calls and allows for cancellation.
func (v *Validator) ValidateProjectEmpty(ctx context.Context, projectID int64) (*EmptinessChecks, error) {
	checks := &EmptinessChecks{}

	// Check for commits (with context support)
	commits, resp, err := v.commitsService.ListCommits(
		projectID,
		&gitlabapi.ListCommitsOptions{
			ListOptions: gitlabapi.ListOptions{
				PerPage: 1, // Only need to know if any exist
				Page:    1,
			},
		},
		gitlabapi.WithContext(ctx),
	)
	if err != nil {
		// Check if cancellation caused the error
		if ctx.Err() != nil {
			return nil, fmt.Errorf("validation cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}
	checks.HasCommits = len(commits) > 0
	checks.CommitCount = getTotalCount(resp, len(commits))

	// Early exit if context cancelled
	if ctx.Err() != nil {
		return nil, fmt.Errorf("validation cancelled: %w", ctx.Err())
	}

	// Check for issues (with context support)
	issues, resp, err := v.issuesService.ListProjectIssues(
		projectID,
		&gitlabapi.ListProjectIssuesOptions{
			ListOptions: gitlabapi.ListOptions{
				PerPage: 1, // Only need to know if any exist
				Page:    1,
			},
		},
		gitlabapi.WithContext(ctx),
	)
	if err != nil {
		// Check if cancellation caused the error
		if ctx.Err() != nil {
			return nil, fmt.Errorf("validation cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}
	checks.HasIssues = len(issues) > 0
	checks.IssueCount = getTotalCount(resp, len(issues))

	// Early exit if context cancelled
	if ctx.Err() != nil {
		return nil, fmt.Errorf("validation cancelled: %w", ctx.Err())
	}

	// Check for labels (with context support)
	labels, resp, err := v.labelsService.ListLabels(
		projectID,
		&gitlabapi.ListLabelsOptions{
			ListOptions: gitlabapi.ListOptions{
				PerPage: 1, // Only need to know if any exist
				Page:    1,
			},
		},
		gitlabapi.WithContext(ctx),
	)
	if err != nil {
		// Check if cancellation caused the error
		if ctx.Err() != nil {
			return nil, fmt.Errorf("validation cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	checks.HasLabels = len(labels) > 0
	checks.LabelCount = getTotalCount(resp, len(labels))

	return checks, nil
}

// getTotalCount extracts the total count from GitLab API response headers.
// Falls back to the length of returned items if header is not available.
func getTotalCount(resp *gitlabapi.Response, itemCount int) int {
	if resp != nil && resp.TotalItems > 0 {
		return int(resp.TotalItems)
	}
	return itemCount
}

