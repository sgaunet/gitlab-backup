package gitlab_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	gitlabAPI "gitlab.com/gitlab-org/api/client-go"
)

// unlimited builds a service whose rate limiters never block, for fast tests.
func unlimited() gitlab.ServiceOption {
	return gitlab.WithRateLimiters(
		rate.NewLimiter(rate.Inf, 1),
		rate.NewLimiter(rate.Inf, 1),
		rate.NewLimiter(rate.Inf, 1),
	)
}

// importExportClient wires a GitLabClientMock that returns the given import/export mock.
func importExportClient(ie *mocks.ProjectImportExportServiceMock) *mocks.GitLabClientMock {
	return &mocks.GitLabClientMock{
		ProjectImportExportFunc: func() gitlab.ProjectImportExportService { return ie },
	}
}

func TestService_ExportProject_HappyPath(t *testing.T) {
	var downloaded []byte
	ie := &mocks.ProjectImportExportServiceMock{
		ScheduleExportFunc: func(_ context.Context, _ any, _ *gitlabAPI.ScheduleExportOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Response, error) {
			return &gitlabAPI.Response{Response: &http.Response{StatusCode: http.StatusAccepted}}, nil
		},
		ExportStatusFunc: func(_ context.Context, _ any, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.ExportStatus, *gitlabAPI.Response, error) {
			return &gitlabAPI.ExportStatus{ExportStatus: "finished"}, &gitlabAPI.Response{}, nil
		},
		ExportDownloadStreamFunc: func(_ context.Context, _ any, w io.Writer, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Response, error) {
			_, _ = w.Write([]byte("archive-content"))
			return &gitlabAPI.Response{}, nil
		},
	}

	svc := gitlab.NewServiceWithClient(importExportClient(ie), unlimited())
	archivePath := filepath.Join(t.TempDir(), "proj.tar.gz")

	err := svc.ExportProject(context.Background(), &gitlab.Project{ID: 1, Name: "proj"}, archivePath)
	require.NoError(t, err)

	downloaded, err = os.ReadFile(archivePath)
	require.NoError(t, err)
	assert.Equal(t, "archive-content", string(downloaded))
}

func TestService_ExportProject_ArchivedShortCircuit(t *testing.T) {
	ie := &mocks.ProjectImportExportServiceMock{} // no funcs -> would panic if called
	svc := gitlab.NewServiceWithClient(importExportClient(ie), unlimited())

	err := svc.ExportProject(context.Background(), &gitlab.Project{ID: 1, Name: "p", Archived: true}, "/does/not/matter")
	require.NoError(t, err)
	assert.Empty(t, ie.ScheduleExportCalls(), "archived project must not schedule an export")
}

func TestService_ExportProject_ScheduleError(t *testing.T) {
	ie := &mocks.ProjectImportExportServiceMock{
		ScheduleExportFunc: func(_ context.Context, _ any, _ *gitlabAPI.ScheduleExportOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Response, error) {
			return nil, errors.New("schedule boom")
		},
	}
	svc := gitlab.NewServiceWithClient(importExportClient(ie), unlimited())

	err := svc.ExportProject(context.Background(), &gitlab.Project{ID: 1, Name: "p"}, filepath.Join(t.TempDir(), "a.tar.gz"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "export request")
}

func TestService_ExportProject_WaitTimeout(t *testing.T) {
	ie := &mocks.ProjectImportExportServiceMock{
		ScheduleExportFunc: func(_ context.Context, _ any, _ *gitlabAPI.ScheduleExportOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Response, error) {
			return &gitlabAPI.Response{Response: &http.Response{StatusCode: http.StatusAccepted}}, nil
		},
		ExportStatusFunc: func(_ context.Context, _ any, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.ExportStatus, *gitlabAPI.Response, error) {
			return &gitlabAPI.ExportStatus{ExportStatus: "started"}, &gitlabAPI.Response{}, nil
		},
	}
	svc := gitlab.NewServiceWithClient(importExportClient(ie), unlimited(), gitlab.WithExportTimeout(time.Millisecond))

	err := svc.ExportProject(context.Background(), &gitlab.Project{ID: 1, Name: "p"}, filepath.Join(t.TempDir(), "a.tar.gz"))
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestService_ExportProject_RetryExhaustion(t *testing.T) {
	ie := &mocks.ProjectImportExportServiceMock{
		ScheduleExportFunc: func(_ context.Context, _ any, _ *gitlabAPI.ScheduleExportOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Response, error) {
			return &gitlabAPI.Response{Response: &http.Response{StatusCode: http.StatusAccepted}}, nil
		},
		ExportStatusFunc: func(_ context.Context, _ any, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.ExportStatus, *gitlabAPI.Response, error) {
			// "none" is counted against MaxExportRetries.
			return &gitlabAPI.ExportStatus{ExportStatus: "none"}, &gitlabAPI.Response{}, nil
		},
	}
	svc := gitlab.NewServiceWithClient(
		importExportClient(ie),
		unlimited(),
		gitlab.WithExportTimeout(time.Minute),
		gitlab.WithExportCheckInterval(time.Millisecond),
	)

	err := svc.ExportProject(context.Background(), &gitlab.Project{ID: 1, Name: "p"}, filepath.Join(t.TempDir(), "a.tar.gz"))
	require.Error(t, err)
	assert.ErrorIs(t, err, gitlab.ErrExportTimeout)
}

func TestService_ExportProject_DownloadError(t *testing.T) {
	ie := &mocks.ProjectImportExportServiceMock{
		ScheduleExportFunc: func(_ context.Context, _ any, _ *gitlabAPI.ScheduleExportOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Response, error) {
			return &gitlabAPI.Response{Response: &http.Response{StatusCode: http.StatusAccepted}}, nil
		},
		ExportStatusFunc: func(_ context.Context, _ any, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.ExportStatus, *gitlabAPI.Response, error) {
			return &gitlabAPI.ExportStatus{ExportStatus: "finished"}, &gitlabAPI.Response{}, nil
		},
		ExportDownloadStreamFunc: func(_ context.Context, _ any, _ io.Writer, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Response, error) {
			return &gitlabAPI.Response{}, errors.New("download boom")
		},
	}
	svc := gitlab.NewServiceWithClient(importExportClient(ie), unlimited())
	archivePath := filepath.Join(t.TempDir(), "a.tar.gz")

	err := svc.ExportProject(context.Background(), &gitlab.Project{ID: 1, Name: "p"}, archivePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "download export")

	// The final archive must not exist (only a .tmp was created, never renamed).
	_, statErr := os.Stat(archivePath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestService_GetProjectsOfGroup(t *testing.T) {
	groups := &mocks.GroupsServiceMock{
		ListSubGroupsFunc: func(_ context.Context, gid any, _ *gitlabAPI.ListSubGroupsOptions, _ ...gitlabAPI.RequestOptionFunc) ([]*gitlabAPI.Group, *gitlabAPI.Response, error) {
			if gid == int64(100) {
				return []*gitlabAPI.Group{{ID: 200, Name: "sub"}}, &gitlabAPI.Response{NextPage: 0}, nil
			}
			return nil, &gitlabAPI.Response{NextPage: 0}, nil
		},
		ListGroupProjectsFunc: func(_ context.Context, gid any, _ *gitlabAPI.ListGroupProjectsOptions, _ ...gitlabAPI.RequestOptionFunc) ([]*gitlabAPI.Project, *gitlabAPI.Response, error) {
			switch gid {
			case int64(200):
				return []*gitlabAPI.Project{{ID: 2, Name: "sub-proj"}}, &gitlabAPI.Response{NextPage: 0}, nil
			case int64(100):
				return []*gitlabAPI.Project{
					{ID: 1, Name: "main"},
					{ID: 3, Name: "arch", Archived: true},
				}, &gitlabAPI.Response{NextPage: 0}, nil
			default:
				return nil, &gitlabAPI.Response{NextPage: 0}, nil
			}
		},
	}
	client := &mocks.GitLabClientMock{
		GroupsFunc: func() gitlab.GroupsService { return groups },
	}
	svc := gitlab.NewServiceWithClient(client, unlimited())

	projects, err := svc.GetProjectsOfGroup(context.Background(), 100)
	require.NoError(t, err)

	// Archived project 3 filtered out; projects from subgroup and main group aggregated.
	require.Len(t, projects, 2)
	ids := map[int64]bool{}
	for _, p := range projects {
		ids[p.ID] = true
	}
	assert.True(t, ids[1], "main project should be present")
	assert.True(t, ids[2], "subgroup project should be present")
	assert.False(t, ids[3], "archived project must be filtered out")
}

func TestService_Getters(t *testing.T) {
	svc := gitlab.NewServiceWithClient(&mocks.GitLabClientMock{}, unlimited())

	svc.SetGitlabEndpoint("https://gitlab.example.com/api/v4")
	assert.Equal(t, "https://gitlab.example.com/api/v4", svc.GitlabEndpoint())

	assert.NotNil(t, svc.Client(), "Client should return the injected client")
	assert.NotNil(t, svc.RateLimitImportAPI(), "import rate limiter should be set")
}

func TestService_GetProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		projects := &mocks.ProjectsServiceMock{
			GetProjectFunc: func(_ context.Context, _ any, _ *gitlabAPI.GetProjectOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Project, *gitlabAPI.Response, error) {
				return &gitlabAPI.Project{ID: 42, Name: "answer"}, &gitlabAPI.Response{}, nil
			},
		}
		client := &mocks.GitLabClientMock{ProjectsFunc: func() gitlab.ProjectsService { return projects }}
		svc := gitlab.NewServiceWithClient(client, unlimited())

		p, err := svc.GetProject(context.Background(), 42)
		require.NoError(t, err)
		assert.Equal(t, int64(42), p.ID)
		assert.Equal(t, "answer", p.Name)
	})

	t.Run("error", func(t *testing.T) {
		projects := &mocks.ProjectsServiceMock{
			GetProjectFunc: func(_ context.Context, _ any, _ *gitlabAPI.GetProjectOptions, _ ...gitlabAPI.RequestOptionFunc) (*gitlabAPI.Project, *gitlabAPI.Response, error) {
				return nil, &gitlabAPI.Response{}, errors.New("not found")
			},
		}
		client := &mocks.GitLabClientMock{ProjectsFunc: func() gitlab.ProjectsService { return projects }}
		svc := gitlab.NewServiceWithClient(client, unlimited())

		_, err := svc.GetProject(context.Background(), 42)
		require.Error(t, err)
	})
}
