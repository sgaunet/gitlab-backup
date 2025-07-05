package gitlab

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/time/rate"
)

// Manual mocks to avoid import cycle issues with moq

// mockGitLabClient is a manual mock implementation of GitLabClient
type mockGitLabClient struct {
	groupsService            GroupsService
	projectsService          ProjectsService
	projectImportExportService ProjectImportExportService
}

func (m *mockGitLabClient) Groups() GroupsService {
	return m.groupsService
}

func (m *mockGitLabClient) Projects() ProjectsService {
	return m.projectsService
}

func (m *mockGitLabClient) ProjectImportExport() ProjectImportExportService {
	return m.projectImportExportService
}

// mockGroupsService is a manual mock implementation of GroupsService
type mockGroupsService struct {
	getGroupFunc           func(gid interface{}, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error)
	listSubGroupsFunc      func(gid interface{}, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error)
	listGroupProjectsFunc  func(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error)
}

func (m *mockGroupsService) GetGroup(gid interface{}, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
	if m.getGroupFunc != nil {
		return m.getGroupFunc(gid, opt, options...)
	}
	return nil, nil, nil
}

func (m *mockGroupsService) ListSubGroups(gid interface{}, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
	if m.listSubGroupsFunc != nil {
		return m.listSubGroupsFunc(gid, opt, options...)
	}
	return nil, nil, nil
}

func (m *mockGroupsService) ListGroupProjects(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
	if m.listGroupProjectsFunc != nil {
		return m.listGroupProjectsFunc(gid, opt, options...)
	}
	return nil, nil, nil
}

// mockProjectsService is a manual mock implementation of ProjectsService
type mockProjectsService struct {
	getProjectFunc func(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error)
}

func (m *mockProjectsService) GetProject(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	if m.getProjectFunc != nil {
		return m.getProjectFunc(pid, opt, options...)
	}
	return nil, nil, nil
}

// mockProjectImportExportService is a manual mock implementation of ProjectImportExportService
type mockProjectImportExportService struct {
	scheduleExportFunc  func(pid interface{}, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error)
	exportStatusFunc    func(pid interface{}, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error)
	exportDownloadFunc  func(pid interface{}, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error)
}

func (m *mockProjectImportExportService) ScheduleExport(pid interface{}, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
	if m.scheduleExportFunc != nil {
		return m.scheduleExportFunc(pid, opt, options...)
	}
	return nil, nil
}

func (m *mockProjectImportExportService) ExportStatus(pid interface{}, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error) {
	if m.exportStatusFunc != nil {
		return m.exportStatusFunc(pid, options...)
	}
	return nil, nil, nil
}

func (m *mockProjectImportExportService) ExportDownload(pid interface{}, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error) {
	if m.exportDownloadFunc != nil {
		return m.exportDownloadFunc(pid, options...)
	}
	return nil, nil, nil
}

// Helper function to create a test service with mock client
func createTestService(client GitLabClient) *Service {
	return &Service{
		client:               client,
		gitlabAPIEndpoint:    GitlabAPIEndpoint,
		token:                "test-token",
		rateLimitDownloadAPI: rate.NewLimiter(rate.Every(time.Second), 1),
		rateLimitExportAPI:   rate.NewLimiter(rate.Every(time.Second), 1),
	}
}

func TestService_GetGroup_Success(t *testing.T) {
	groupsService := &mockGroupsService{
		getGroupFunc: func(gid interface{}, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
			assert.Equal(t, 1, gid)
			return &gitlab.Group{
				ID:   1,
				Name: "test-group",
			}, &gitlab.Response{}, nil
		},
	}

	client := &mockGitLabClient{
		groupsService: groupsService,
	}

	service := createTestService(client)

	result, err := service.GetGroup(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, Group{ID: 1, Name: "test-group"}, result)
}

func TestService_GetGroup_Error(t *testing.T) {
	groupsService := &mockGroupsService{
		getGroupFunc: func(gid interface{}, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
			return nil, &gitlab.Response{}, errors.New("404 Group Not Found")
		},
	}

	client := &mockGitLabClient{
		groupsService: groupsService,
	}

	service := createTestService(client)

	_, err := service.GetGroup(context.Background(), 999)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error retrieving group")
}

func TestService_GetProject_Success(t *testing.T) {
	projectsService := &mockProjectsService{
		getProjectFunc: func(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
			assert.Equal(t, 1, pid)
			return &gitlab.Project{
				ID:       1,
				Name:     "test-project",
				Archived: false,
			}, &gitlab.Response{}, nil
		},
	}

	client := &mockGitLabClient{
		projectsService: projectsService,
	}

	service := createTestService(client)

	result, err := service.GetProject(context.Background(), 1)

	require.NoError(t, err)
	expected := Project{
		ID:           1,
		Name:         "test-project",
		Archived:     false,
		ExportStatus: "",
	}
	assert.Equal(t, expected, result)
}

func TestService_askExport_Success(t *testing.T) {
	exportService := &mockProjectImportExportService{
		scheduleExportFunc: func(pid interface{}, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
			assert.Equal(t, 1, pid)
			return &gitlab.Response{
				Response: &http.Response{
					StatusCode: http.StatusAccepted,
				},
			}, nil
		},
	}

	client := &mockGitLabClient{
		projectImportExportService: exportService,
	}

	service := createTestService(client)

	result, err := service.askExport(context.Background(), 1)

	require.NoError(t, err)
	assert.True(t, result)
}

func TestService_askExport_Error(t *testing.T) {
	exportService := &mockProjectImportExportService{
		scheduleExportFunc: func(pid interface{}, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
			return nil, errors.New("API error")
		},
	}

	client := &mockGitLabClient{
		projectImportExportService: exportService,
	}

	service := createTestService(client)

	result, err := service.askExport(context.Background(), 1)

	require.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to make export request")
}

func TestService_getStatusExport_Success(t *testing.T) {
	exportService := &mockProjectImportExportService{
		exportStatusFunc: func(pid interface{}, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error) {
			assert.Equal(t, 1, pid)
			return &gitlab.ExportStatus{
				ExportStatus: "finished",
			}, &gitlab.Response{}, nil
		},
	}

	client := &mockGitLabClient{
		projectImportExportService: exportService,
	}

	service := createTestService(client)

	result, err := service.getStatusExport(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, "finished", result)
}

func TestService_GetSubgroups_Success(t *testing.T) {
	callCount := 0
	groupsService := &mockGroupsService{
		listSubGroupsFunc: func(gid interface{}, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
			callCount++
			if callCount == 1 {
				// First call returns subgroups
				return []*gitlab.Group{
					{ID: 2, Name: "subgroup1"},
					{ID: 3, Name: "subgroup2"},
				}, &gitlab.Response{NextPage: 0}, nil
			}
			// Subsequent calls for recursive subgroups return empty
			return []*gitlab.Group{}, &gitlab.Response{NextPage: 0}, nil
		},
	}

	client := &mockGitLabClient{
		groupsService: groupsService,
	}

	service := createTestService(client)

	result, err := service.GetSubgroups(context.Background(), 1)

	require.NoError(t, err)
	expected := []Group{
		{ID: 2, Name: "subgroup1"},
		{ID: 3, Name: "subgroup2"},
	}
	assert.Equal(t, expected, result)
}

func TestService_GetProjectsLst_Success(t *testing.T) {
	groupsService := &mockGroupsService{
		listGroupProjectsFunc: func(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
			assert.Equal(t, 1, gid)
			return []*gitlab.Project{
				{
					ID:       1,
					Name:     "project1",
					Archived: false,
				},
				{
					ID:       2,
					Name:     "project2",
					Archived: true,
				},
			}, &gitlab.Response{NextPage: 0}, nil
		},
	}

	client := &mockGitLabClient{
		groupsService: groupsService,
	}

	service := createTestService(client)

	result, err := service.GetProjectsLst(context.Background(), 1)

	require.NoError(t, err)
	expected := []Project{
		{ID: 1, Name: "project1", Archived: false, ExportStatus: ""},
		{ID: 2, Name: "project2", Archived: true, ExportStatus: ""},
	}
	assert.Equal(t, expected, result)
}

func TestNewGitlabService_CreatesCorrectDefaults(t *testing.T) {
	service := NewGitlabService()
	
	// Service might be nil if no GITLAB_TOKEN is set, which is expected behavior
	if service != nil {
		assert.NotNil(t, service.client)
		assert.Equal(t, GitlabAPIEndpoint, service.gitlabAPIEndpoint)
		assert.NotNil(t, service.rateLimitDownloadAPI)
		assert.NotNil(t, service.rateLimitExportAPI)
		
		// Test rate limiter configuration
		downloadLimit := service.rateLimitDownloadAPI.Limit()
		exportLimit := service.rateLimitExportAPI.Limit()
		
		assert.Equal(t, rate.Every(DownloadRateLimitIntervalSeconds*time.Second), downloadLimit)
		assert.Equal(t, rate.Every(ExportRateLimitIntervalSeconds*time.Second), exportLimit)
	}
}

func TestService_SetToken(t *testing.T) {
	service := NewGitlabService()
	if service == nil {
		t.Skip("Cannot create service without GITLAB_TOKEN")
	}

	originalToken := service.token
	newToken := "new-test-token"
	
	service.SetToken(newToken)
	
	assert.Equal(t, newToken, service.token)
	assert.NotEqual(t, originalToken, service.token)
	assert.NotNil(t, service.client)
}

func TestService_SetGitlabEndpoint(t *testing.T) {
	service := NewGitlabService()
	if service == nil {
		t.Skip("Cannot create service without GITLAB_TOKEN")
	}

	originalEndpoint := service.gitlabAPIEndpoint
	newEndpoint := "https://custom-gitlab.example.com/api/v4"
	
	service.SetGitlabEndpoint(newEndpoint)
	
	assert.Equal(t, newEndpoint, service.gitlabAPIEndpoint)
	assert.NotEqual(t, originalEndpoint, service.gitlabAPIEndpoint)
	assert.NotNil(t, service.client)
}