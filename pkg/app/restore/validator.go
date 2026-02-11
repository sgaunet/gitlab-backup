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

	// Check for commits
	hasCommits, commitCount, err := v.checkCommits(ctx, projectID)
	if err != nil {
		return nil, err
	}
	checks.HasCommits = hasCommits
	checks.CommitCount = commitCount

	// Check for issues
	hasIssues, issueCount, err := v.checkIssues(ctx, projectID)
	if err != nil {
		return nil, err
	}
	checks.HasIssues = hasIssues
	checks.IssueCount = issueCount

	// Check for labels
	hasLabels, labelCount, err := v.checkLabels(ctx, projectID)
	if err != nil {
		return nil, err
	}
	checks.HasLabels = hasLabels
	checks.LabelCount = labelCount

	return checks, nil
}

// checkCommits checks if a project has any commits.
// Returns whether commits exist, the total count, and any error encountered.
func (v *Validator) checkCommits(ctx context.Context, projectID int64) (bool, int, error) {
	// Early exit if context cancelled
	if ctx.Err() != nil {
		return false, 0, fmt.Errorf("validation cancelled: %w", ctx.Err())
	}

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
		if ctx.Err() != nil {
			return false, 0, fmt.Errorf("validation cancelled: %w", ctx.Err())
		}
		return false, 0, fmt.Errorf("failed to list commits: %w", err)
	}

	return len(commits) > 0, getTotalCount(resp, len(commits)), nil
}

// checkIssues checks if a project has any issues.
// Returns whether issues exist, the total count, and any error encountered.
func (v *Validator) checkIssues(ctx context.Context, projectID int64) (bool, int, error) {
	// Early exit if context cancelled
	if ctx.Err() != nil {
		return false, 0, fmt.Errorf("validation cancelled: %w", ctx.Err())
	}

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
		if ctx.Err() != nil {
			return false, 0, fmt.Errorf("validation cancelled: %w", ctx.Err())
		}
		return false, 0, fmt.Errorf("failed to list issues: %w", err)
	}

	return len(issues) > 0, getTotalCount(resp, len(issues)), nil
}

// checkLabels checks if a project has any labels.
// Returns whether labels exist, the total count, and any error encountered.
func (v *Validator) checkLabels(ctx context.Context, projectID int64) (bool, int, error) {
	// Early exit if context cancelled
	if ctx.Err() != nil {
		return false, 0, fmt.Errorf("validation cancelled: %w", ctx.Err())
	}

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
		if ctx.Err() != nil {
			return false, 0, fmt.Errorf("validation cancelled: %w", ctx.Err())
		}
		return false, 0, fmt.Errorf("failed to list labels: %w", err)
	}

	return len(labels) > 0, getTotalCount(resp, len(labels)), nil
}

// getTotalCount extracts the total count from GitLab API response headers.
// Falls back to the length of returned items if header is not available.
func getTotalCount(resp *gitlabapi.Response, itemCount int) int {
	if resp != nil && resp.TotalItems > 0 {
		return int(resp.TotalItems)
	}
	return itemCount
}

