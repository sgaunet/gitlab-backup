package gitlab

import (
	"context"
	"errors"
	"io"
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
	groupsService              GroupsService
	projectsService            ProjectsService
	projectImportExportService ProjectImportExportService
	labelsService              LabelsService
	issuesService              IssuesService
	notesService               NotesService
	commitsService             CommitsService
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

func (m *mockGitLabClient) Labels() LabelsService {
	return m.labelsService
}

func (m *mockGitLabClient) Issues() IssuesService {
	return m.issuesService
}

func (m *mockGitLabClient) Notes() NotesService {
	return m.notesService
}

func (m *mockGitLabClient) Commits() CommitsService {
	return m.commitsService
}

// mockGroupsService is a manual mock implementation of GroupsService
type mockGroupsService struct {
	getGroupFunc           func(gid any, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error)
	listSubGroupsFunc      func(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error)
	listGroupProjectsFunc  func(gid any, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error)
}

func (m *mockGroupsService) GetGroup(gid any, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
	if m.getGroupFunc != nil {
		return m.getGroupFunc(gid, opt, options...)
	}
	return nil, nil, nil
}

func (m *mockGroupsService) ListSubGroups(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
	if m.listSubGroupsFunc != nil {
		return m.listSubGroupsFunc(gid, opt, options...)
	}
	return nil, nil, nil
}

func (m *mockGroupsService) ListGroupProjects(gid any, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
	if m.listGroupProjectsFunc != nil {
		return m.listGroupProjectsFunc(gid, opt, options...)
	}
	return nil, nil, nil
}

// mockProjectsService is a manual mock implementation of ProjectsService
type mockProjectsService struct {
	getProjectFunc func(pid any, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error)
}

func (m *mockProjectsService) GetProject(pid any, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	if m.getProjectFunc != nil {
		return m.getProjectFunc(pid, opt, options...)
	}
	return nil, nil, nil
}

// mockProjectImportExportService is a manual mock implementation of ProjectImportExportService
type mockProjectImportExportService struct {
	scheduleExportFunc  func(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error)
	exportStatusFunc    func(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error)
	exportDownloadFunc  func(pid any, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error)
	importFromFileFunc  func(archive io.Reader, opt *gitlab.ImportFileOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error)
	importStatusFunc    func(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error)
}

func (m *mockProjectImportExportService) ScheduleExport(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
	if m.scheduleExportFunc != nil {
		return m.scheduleExportFunc(pid, opt, options...)
	}
	return nil, nil
}

func (m *mockProjectImportExportService) ExportStatus(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error) {
	if m.exportStatusFunc != nil {
		return m.exportStatusFunc(pid, options...)
	}
	return nil, nil, nil
}

func (m *mockProjectImportExportService) ExportDownload(pid any, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error) {
	if m.exportDownloadFunc != nil {
		return m.exportDownloadFunc(pid, options...)
	}
	return nil, nil, nil
}

//nolint:ireturn // Mock method signature must match interface
func (m *mockProjectImportExportService) ImportFromFile(archive io.Reader, opt *gitlab.ImportFileOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error) {
	if m.importFromFileFunc != nil {
		return m.importFromFileFunc(archive, opt, options...)
	}
	return nil, nil, nil
}

func (m *mockProjectImportExportService) ImportStatus(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error) {
	if m.importStatusFunc != nil {
		return m.importStatusFunc(pid, options...)
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
			assert.Equal(t, int64(1), gid)
			return &gitlab.Group{
				ID:   int64(1),
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
	assert.Equal(t, Group{ID: int64(1), Name: "test-group"}, result)
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
			assert.Equal(t, int64(1), pid)
			return &gitlab.Project{
				ID:       int64(1),
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
		ID:           int64(1),
		Name:         "test-project",
		Archived:     false,
		ExportStatus: "",
	}
	assert.Equal(t, expected, result)
}

func TestService_askExport_Success(t *testing.T) {
	exportService := &mockProjectImportExportService{
		scheduleExportFunc: func(pid interface{}, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
			assert.Equal(t, int64(1), pid)
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
			assert.Equal(t, int64(1), pid)
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
					{ID: int64(2), Name: "subgroup1"},
					{ID: int64(3), Name: "subgroup2"},
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
		{ID: int64(2), Name: "subgroup1"},
		{ID: int64(3), Name: "subgroup2"},
	}
	assert.Equal(t, expected, result)
}

func TestService_GetProjectsLst_Success(t *testing.T) {
	groupsService := &mockGroupsService{
		listGroupProjectsFunc: func(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
			assert.Equal(t, int64(1), gid)
			return []*gitlab.Project{
				{
					ID:       int64(1),
					Name:     "project1",
					Archived: false,
				},
				{
					ID:       int64(2),
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
		{ID: int64(1), Name: "project1", Archived: false, ExportStatus: ""},
		{ID: int64(2), Name: "project2", Archived: true, ExportStatus: ""},
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