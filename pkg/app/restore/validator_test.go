package restore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/app/restore"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestValidateProjectEmpty(t *testing.T) {
	ctx := context.Background()

	t.Run("EmptyProject", func(t *testing.T) {
		// Setup mocks
		mockCommitsService := &mocks.CommitsServiceMock{
			ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
				// Return empty commits list
				return []*gitlab.Commit{}, &gitlab.Response{}, nil
			},
		}

		mockIssuesService := &mocks.IssuesServiceMock{
			ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
				// Return empty issues list
				return []*gitlab.Issue{}, &gitlab.Response{}, nil
			},
		}

		mockLabelsService := &mocks.LabelsServiceMock{
			ListLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
				// Return empty labels list
				return []*gitlab.Label{}, &gitlab.Response{}, nil
			},
		}

		validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)

		// Execute validation
		checks, err := validator.ValidateProjectEmpty(ctx, 123)

		// Assertions
		require.NoError(t, err, "Validation should succeed for empty project")
		require.NotNil(t, checks, "EmptinessChecks should not be nil")
		assert.True(t, checks.IsEmpty(), "Project should be empty")
		assert.False(t, checks.HasCommits, "Should have no commits")
		assert.False(t, checks.HasIssues, "Should have no issues")
		assert.False(t, checks.HasLabels, "Should have no labels")
		assert.Equal(t, 0, checks.CommitCount, "Commit count should be zero")
		assert.Equal(t, 0, checks.IssueCount, "Issue count should be zero")
		assert.Equal(t, 0, checks.LabelCount, "Label count should be zero")
	})

	t.Run("ProjectWithCommits", func(t *testing.T) {
		// Setup mocks
		mockCommitsService := &mocks.CommitsServiceMock{
			ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
				// Return commits
				return []*gitlab.Commit{
					{ID: "abc123", Message: "Initial commit"},
				}, &gitlab.Response{}, nil
			},
		}

		mockIssuesService := &mocks.IssuesServiceMock{
			ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
				return []*gitlab.Issue{}, &gitlab.Response{}, nil
			},
		}

		mockLabelsService := &mocks.LabelsServiceMock{
			ListLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
				return []*gitlab.Label{}, &gitlab.Response{}, nil
			},
		}

		validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)

		// Execute validation
		checks, err := validator.ValidateProjectEmpty(ctx, 123)

		// Assertions
		require.NoError(t, err, "Validation should succeed")
		require.NotNil(t, checks, "EmptinessChecks should not be nil")
		assert.False(t, checks.IsEmpty(), "Project should not be empty")
		assert.True(t, checks.HasCommits, "Should have commits")
		assert.Equal(t, 1, checks.CommitCount, "Should have one commit")
	})

	t.Run("ProjectWithIssues", func(t *testing.T) {
		// Setup mocks
		mockCommitsService := &mocks.CommitsServiceMock{
			ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
				return []*gitlab.Commit{}, &gitlab.Response{}, nil
			},
		}

		mockIssuesService := &mocks.IssuesServiceMock{
			ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
				// Return issues
				return []*gitlab.Issue{
					{ID: 1, Title: "Test issue"},
				}, &gitlab.Response{}, nil
			},
		}

		mockLabelsService := &mocks.LabelsServiceMock{
			ListLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
				return []*gitlab.Label{}, &gitlab.Response{}, nil
			},
		}

		validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)

		// Execute validation
		checks, err := validator.ValidateProjectEmpty(ctx, 123)

		// Assertions
		require.NoError(t, err, "Validation should succeed")
		require.NotNil(t, checks, "EmptinessChecks should not be nil")
		assert.False(t, checks.IsEmpty(), "Project should not be empty")
		assert.True(t, checks.HasIssues, "Should have issues")
		assert.Equal(t, 1, checks.IssueCount, "Should have one issue")
	})

	t.Run("ProjectWithLabels", func(t *testing.T) {
		// Setup mocks
		mockCommitsService := &mocks.CommitsServiceMock{
			ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
				return []*gitlab.Commit{}, &gitlab.Response{}, nil
			},
		}

		mockIssuesService := &mocks.IssuesServiceMock{
			ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
				return []*gitlab.Issue{}, &gitlab.Response{}, nil
			},
		}

		mockLabelsService := &mocks.LabelsServiceMock{
			ListLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
				// Return labels
				return []*gitlab.Label{
					{ID: 1, Name: "bug", Color: "#FF0000"},
				}, &gitlab.Response{}, nil
			},
		}

		validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)

		// Execute validation
		checks, err := validator.ValidateProjectEmpty(ctx, 123)

		// Assertions
		require.NoError(t, err, "Validation should succeed")
		require.NotNil(t, checks, "EmptinessChecks should not be nil")
		assert.False(t, checks.IsEmpty(), "Project should not be empty")
		assert.True(t, checks.HasLabels, "Should have labels")
		assert.Equal(t, 1, checks.LabelCount, "Should have one label")
	})

	t.Run("APIError", func(t *testing.T) {
		// Setup mocks with error
		mockCommitsService := &mocks.CommitsServiceMock{
			ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
				return nil, nil, errors.New("API error")
			},
		}

		mockIssuesService := &mocks.IssuesServiceMock{}
		mockLabelsService := &mocks.LabelsServiceMock{}

		validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)

		// Execute validation
		checks, err := validator.ValidateProjectEmpty(ctx, 123)

		// Assertions
		require.Error(t, err, "Validation should fail on API error")
		assert.Nil(t, checks, "EmptinessChecks should be nil on error")
	})
}
