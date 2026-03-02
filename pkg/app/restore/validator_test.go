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

func TestValidateProjectEmpty_IssuesError(t *testing.T) {
	ctx := context.Background()

	mockCommitsService := &mocks.CommitsServiceMock{
		ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
			return []*gitlab.Commit{}, &gitlab.Response{}, nil
		},
	}
	mockIssuesService := &mocks.IssuesServiceMock{
		ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
			return nil, nil, errors.New("issues API error")
		},
	}
	mockLabelsService := &mocks.LabelsServiceMock{}

	validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)
	checks, err := validator.ValidateProjectEmpty(ctx, 123)

	require.Error(t, err)
	assert.Nil(t, checks)
	assert.Contains(t, err.Error(), "failed to list issues")
}

func TestValidateProjectEmpty_LabelsError(t *testing.T) {
	ctx := context.Background()

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
			return nil, nil, errors.New("labels API error")
		},
	}

	validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)
	checks, err := validator.ValidateProjectEmpty(ctx, 123)

	require.Error(t, err)
	assert.Nil(t, checks)
	assert.Contains(t, err.Error(), "failed to list labels")
}

func TestValidateProjectEmpty_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockCommitsService := &mocks.CommitsServiceMock{}
	mockIssuesService := &mocks.IssuesServiceMock{}
	mockLabelsService := &mocks.LabelsServiceMock{}

	validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)
	checks, err := validator.ValidateProjectEmpty(ctx, 123)

	require.Error(t, err)
	assert.Nil(t, checks)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "validation cancelled")
	assert.Empty(t, mockCommitsService.ListCommitsCalls(), "No API calls should be made with cancelled context")
	assert.Empty(t, mockIssuesService.ListProjectIssuesCalls())
	assert.Empty(t, mockLabelsService.ListLabelsCalls())
}

func TestValidateProjectEmpty_ContextCancelledDuringIssues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockCommitsService := &mocks.CommitsServiceMock{
		ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
			return []*gitlab.Commit{}, &gitlab.Response{}, nil
		},
	}
	mockIssuesService := &mocks.IssuesServiceMock{
		ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
			cancel()
			return nil, nil, context.Canceled
		},
	}
	mockLabelsService := &mocks.LabelsServiceMock{}

	validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)
	checks, err := validator.ValidateProjectEmpty(ctx, 123)

	require.Error(t, err)
	assert.Nil(t, checks)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "validation cancelled while checking issues")
}

func TestValidateProjectEmpty_PartialFailures(t *testing.T) {
	tests := []struct {
		name           string
		commitsErr     error
		issuesErr      error
		labelsErr      error
		wantErrContain string
	}{
		{
			name:           "CommitsFail",
			commitsErr:     errors.New("commits timeout"),
			wantErrContain: "commits",
		},
		{
			name:           "IssuesFail",
			issuesErr:      errors.New("issues forbidden"),
			wantErrContain: "issues",
		},
		{
			name:           "LabelsFail",
			labelsErr:      errors.New("labels not found"),
			wantErrContain: "labels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCommitsService := &mocks.CommitsServiceMock{
				ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
					if tt.commitsErr != nil {
						return nil, nil, tt.commitsErr
					}
					return []*gitlab.Commit{}, &gitlab.Response{}, nil
				},
			}
			mockIssuesService := &mocks.IssuesServiceMock{
				ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
					if tt.issuesErr != nil {
						return nil, nil, tt.issuesErr
					}
					return []*gitlab.Issue{}, &gitlab.Response{}, nil
				},
			}
			mockLabelsService := &mocks.LabelsServiceMock{
				ListLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
					if tt.labelsErr != nil {
						return nil, nil, tt.labelsErr
					}
					return []*gitlab.Label{}, &gitlab.Response{}, nil
				},
			}

			validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)
			checks, err := validator.ValidateProjectEmpty(ctx, 123)

			require.Error(t, err)
			assert.Nil(t, checks)
			assert.Contains(t, err.Error(), tt.wantErrContain)
		})
	}
}

func TestValidateProjectEmpty_TotalItemsFromResponse(t *testing.T) {
	ctx := context.Background()

	mockCommitsService := &mocks.CommitsServiceMock{
		ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
			return []*gitlab.Commit{{ID: "abc"}}, &gitlab.Response{TotalItems: 42}, nil
		},
	}
	mockIssuesService := &mocks.IssuesServiceMock{
		ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
			return []*gitlab.Issue{{ID: 1}}, &gitlab.Response{TotalItems: 15}, nil
		},
	}
	mockLabelsService := &mocks.LabelsServiceMock{
		ListLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
			return []*gitlab.Label{{ID: 1}}, &gitlab.Response{TotalItems: 7}, nil
		},
	}

	validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)
	checks, err := validator.ValidateProjectEmpty(ctx, 123)

	require.NoError(t, err)
	require.NotNil(t, checks)
	assert.True(t, checks.HasCommits)
	assert.Equal(t, 42, checks.CommitCount)
	assert.True(t, checks.HasIssues)
	assert.Equal(t, 15, checks.IssueCount)
	assert.True(t, checks.HasLabels)
	assert.Equal(t, 7, checks.LabelCount)
	assert.False(t, checks.IsEmpty())
}

func TestValidateProjectEmpty_NilResponse(t *testing.T) {
	ctx := context.Background()

	mockCommitsService := &mocks.CommitsServiceMock{
		ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
			return []*gitlab.Commit{{ID: "abc"}}, nil, nil
		},
	}
	mockIssuesService := &mocks.IssuesServiceMock{
		ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
			return []*gitlab.Issue{}, nil, nil
		},
	}
	mockLabelsService := &mocks.LabelsServiceMock{
		ListLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
			return []*gitlab.Label{}, nil, nil
		},
	}

	validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)
	checks, err := validator.ValidateProjectEmpty(ctx, 123)

	require.NoError(t, err)
	require.NotNil(t, checks)
	assert.True(t, checks.HasCommits)
	assert.Equal(t, 1, checks.CommitCount, "Should fallback to len(items) with nil response")
	assert.False(t, checks.HasIssues)
	assert.Equal(t, 0, checks.IssueCount)
	assert.False(t, checks.HasLabels)
	assert.Equal(t, 0, checks.LabelCount)
}

func TestValidateProjectEmpty_AllNonEmpty(t *testing.T) {
	ctx := context.Background()

	mockCommitsService := &mocks.CommitsServiceMock{
		ListCommitsFunc: func(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
			return []*gitlab.Commit{{ID: "abc"}}, &gitlab.Response{TotalItems: 100}, nil
		},
	}
	mockIssuesService := &mocks.IssuesServiceMock{
		ListProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
			return []*gitlab.Issue{{ID: 1}}, &gitlab.Response{TotalItems: 50}, nil
		},
	}
	mockLabelsService := &mocks.LabelsServiceMock{
		ListLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
			return []*gitlab.Label{{ID: 1}}, &gitlab.Response{TotalItems: 25}, nil
		},
	}

	validator := restore.NewValidator(mockCommitsService, mockIssuesService, mockLabelsService)
	checks, err := validator.ValidateProjectEmpty(ctx, 123)

	require.NoError(t, err)
	require.NotNil(t, checks)
	assert.True(t, checks.HasCommits)
	assert.True(t, checks.HasIssues)
	assert.True(t, checks.HasLabels)
	assert.Equal(t, 100, checks.CommitCount)
	assert.Equal(t, 50, checks.IssueCount)
	assert.Equal(t, 25, checks.LabelCount)
	assert.False(t, checks.IsEmpty())
}
