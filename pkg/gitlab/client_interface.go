package gitlab

import (
	"io"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

//go:generate go tool github.com/matryer/moq -out mocks/client.go -pkg mocks . GitLabClient
//go:generate go tool github.com/matryer/moq -out mocks/groups.go -pkg mocks . GroupsService
//go:generate go tool github.com/matryer/moq -out mocks/projects.go -pkg mocks . ProjectsService
//go:generate go tool github.com/matryer/moq -out mocks/project_import_export.go -pkg mocks . ProjectImportExportService
//go:generate go tool github.com/matryer/moq -out mocks/labels.go -pkg mocks . LabelsService
//go:generate go tool github.com/matryer/moq -out mocks/issues.go -pkg mocks . IssuesService
//go:generate go tool github.com/matryer/moq -out mocks/notes.go -pkg mocks . NotesService
//go:generate go tool github.com/matryer/moq -out mocks/commits.go -pkg mocks . CommitsService

// GitLabClient defines the interface for GitLab client operations.
//
//nolint:revive // Client interface naming is intentionally explicit
type GitLabClient interface {
	Groups() GroupsService
	Projects() ProjectsService
	ProjectImportExport() ProjectImportExportService
	Labels() LabelsService
	Issues() IssuesService
	Notes() NotesService
	Commits() CommitsService
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
	//nolint:lll // GitLab API method signatures are inherently long
	ImportFromFile(archive io.Reader, opt *gitlab.ImportFileOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error)
	ImportStatus(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error)
}

// LabelsService defines the interface for GitLab Labels API operations.
type LabelsService interface {
	//nolint:lll // GitLab API method signatures are inherently long
	ListLabels(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error)
	//nolint:lll // GitLab API method signatures are inherently long
	CreateLabel(pid any, opt *gitlab.CreateLabelOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Label, *gitlab.Response, error)
}

// IssuesService defines the interface for GitLab Issues API operations.
type IssuesService interface {
	//nolint:lll // GitLab API method signatures are inherently long
	ListProjectIssues(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error)
	//nolint:lll // GitLab API method signatures are inherently long
	CreateIssue(pid any, opt *gitlab.CreateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error)
	//nolint:lll // GitLab API method signatures are inherently long
	UpdateIssue(pid any, issue int64, opt *gitlab.UpdateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error)
}

// NotesService defines the interface for GitLab Notes API operations.
type NotesService interface {
	//nolint:lll // GitLab API method signatures are inherently long
	CreateIssueNote(pid any, issue int64, opt *gitlab.CreateIssueNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error)
}

// CommitsService defines the interface for GitLab Commits API operations.
type CommitsService interface {
	//nolint:lll // GitLab API method signatures are inherently long
	ListCommits(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error)
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

// Labels returns the labels service.
//
//nolint:ireturn // Interface return is intentional for dependency injection
func (w *gitlabClientWrapper) Labels() LabelsService {
	return &labelsServiceWrapper{service: w.client.Labels}
}

// Issues returns the issues service.
//
//nolint:ireturn // Interface return is intentional for dependency injection
func (w *gitlabClientWrapper) Issues() IssuesService {
	return &issuesServiceWrapper{service: w.client.Issues}
}

// Notes returns the notes service.
//
//nolint:ireturn // Interface return is intentional for dependency injection
func (w *gitlabClientWrapper) Notes() NotesService {
	return &notesServiceWrapper{service: w.client.Notes}
}

// Commits returns the commits service.
//
//nolint:ireturn // Interface return is intentional for dependency injection
func (w *gitlabClientWrapper) Commits() CommitsService {
	return &commitsServiceWrapper{service: w.client.Commits}
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

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *projectImportExportServiceWrapper) ImportFromFile(archive io.Reader, opt *gitlab.ImportFileOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error) {
	return w.service.ImportFromFile(archive, opt, options...)
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *projectImportExportServiceWrapper) ImportStatus(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error) {
	return w.service.ImportStatus(pid, options...)
}

// labelsServiceWrapper wraps the official GitLab labels service.
type labelsServiceWrapper struct {
	service gitlab.LabelsServiceInterface
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *labelsServiceWrapper) ListLabels(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
	return w.service.ListLabels(pid, opt, options...)
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *labelsServiceWrapper) CreateLabel(pid any, opt *gitlab.CreateLabelOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Label, *gitlab.Response, error) {
	return w.service.CreateLabel(pid, opt, options...)
}

// issuesServiceWrapper wraps the official GitLab issues service.
type issuesServiceWrapper struct {
	service gitlab.IssuesServiceInterface
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *issuesServiceWrapper) ListProjectIssues(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
	return w.service.ListProjectIssues(pid, opt, options...)
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *issuesServiceWrapper) CreateIssue(pid any, opt *gitlab.CreateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
	return w.service.CreateIssue(pid, opt, options...)
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *issuesServiceWrapper) UpdateIssue(pid any, issue int64, opt *gitlab.UpdateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
	return w.service.UpdateIssue(pid, issue, opt, options...)
}

// notesServiceWrapper wraps the official GitLab notes service.
type notesServiceWrapper struct {
	service gitlab.NotesServiceInterface
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *notesServiceWrapper) CreateIssueNote(pid any, issue int64, opt *gitlab.CreateIssueNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error) {
	return w.service.CreateIssueNote(pid, issue, opt, options...)
}

// commitsServiceWrapper wraps the official GitLab commits service.
type commitsServiceWrapper struct {
	service gitlab.CommitsServiceInterface
}

//nolint:lll,wrapcheck // Wrapper method with long signature, error passthrough intentional
func (w *commitsServiceWrapper) ListCommits(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
	return w.service.ListCommits(pid, opt, options...)
}
