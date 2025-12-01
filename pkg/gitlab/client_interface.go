package gitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

//go:generate go tool github.com/matryer/moq -out mocks/client.go -pkg mocks . GitLabClient
//go:generate go tool github.com/matryer/moq -out mocks/groups.go -pkg mocks . GroupsService
//go:generate go tool github.com/matryer/moq -out mocks/projects.go -pkg mocks . ProjectsService
//go:generate go tool github.com/matryer/moq -out mocks/project_import_export.go -pkg mocks . ProjectImportExportService

// GitLabClient defines the interface for GitLab client operations.
//
//nolint:revive // Client interface naming is intentionally explicit
type GitLabClient interface {
	Groups() GroupsService
	Projects() ProjectsService
	ProjectImportExport() ProjectImportExportService
}

// GroupsService defines the interface for GitLab Groups API operations.
type GroupsService interface {
	//nolint:lll // GitLab API method signatures are inherently long
	GetGroup(gid any, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error)
	//nolint:lll // GitLab API method signatures are inherently long
	ListSubGroups(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error)
	//nolint:lll // GitLab API method signatures are inherently long
	ListGroupProjects(gid any, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error)
}

// ProjectsService defines the interface for GitLab Projects API operations.
type ProjectsService interface {
	//nolint:lll // GitLab API method signatures are inherently long
	GetProject(pid any, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error)
}

// ProjectImportExportService defines the interface for GitLab Project Import/Export API operations.
type ProjectImportExportService interface {
	//nolint:lll // GitLab API method signatures are inherently long
	ScheduleExport(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error)
	ExportStatus(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error)
	ExportDownload(pid any, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error)
}

// gitlabClientWrapper wraps the official GitLab client to implement our interface.
type gitlabClientWrapper struct {
	client *gitlab.Client
}

// NewGitLabClientWrapper creates a new wrapper around the official GitLab client.
//
//nolint:ireturn // Interface return is intentional for dependency injection
func NewGitLabClientWrapper(client *gitlab.Client) GitLabClient {
	return &gitlabClientWrapper{client: client}
}

// Groups returns the groups service.
//
//nolint:ireturn // Interface return is intentional for dependency injection
func (w *gitlabClientWrapper) Groups() GroupsService {
	return &groupsServiceWrapper{service: w.client.Groups}
}

// Projects returns the projects service.
//
//nolint:ireturn // Interface return is intentional for dependency injection
func (w *gitlabClientWrapper) Projects() ProjectsService {
	return &projectsServiceWrapper{service: w.client.Projects}
}

// ProjectImportExport returns the project import/export service.
//
//nolint:ireturn // Interface return is intentional for dependency injection
func (w *gitlabClientWrapper) ProjectImportExport() ProjectImportExportService {
	return &projectImportExportServiceWrapper{service: w.client.ProjectImportExport}
}

// groupsServiceWrapper wraps the official GitLab groups service.
type groupsServiceWrapper struct {
	service gitlab.GroupsServiceInterface
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *groupsServiceWrapper) GetGroup(gid any, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
	return w.service.GetGroup(gid, opt, options...)
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *groupsServiceWrapper) ListSubGroups(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
	return w.service.ListSubGroups(gid, opt, options...)
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *groupsServiceWrapper) ListGroupProjects(gid any, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
	return w.service.ListGroupProjects(gid, opt, options...)
}

// projectsServiceWrapper wraps the official GitLab projects service.
type projectsServiceWrapper struct {
	service gitlab.ProjectsServiceInterface
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *projectsServiceWrapper) GetProject(pid any, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	return w.service.GetProject(pid, opt, options...)
}

// projectImportExportServiceWrapper wraps the official GitLab project import/export service.
type projectImportExportServiceWrapper struct {
	service gitlab.ProjectImportExportServiceInterface
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *projectImportExportServiceWrapper) ScheduleExport(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
	return w.service.ScheduleExport(pid, opt, options...)
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *projectImportExportServiceWrapper) ExportStatus(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error) {
	return w.service.ExportStatus(pid, options...)
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *projectImportExportServiceWrapper) ExportDownload(pid any, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error) {
	return w.service.ExportDownload(pid, options...)
}
