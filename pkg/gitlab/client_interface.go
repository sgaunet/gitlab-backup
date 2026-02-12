package gitlab

import (
	"fmt"
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

//nolint:lll // Wrapper method with long signature
func (w *groupsServiceWrapper) GetGroup(gid any, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
	group, resp, err := w.service.GetGroup(gid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to get group %v: %w", gid, err)
	}
	return group, resp, nil
}

//nolint:lll // Wrapper method with long signature
func (w *groupsServiceWrapper) ListSubGroups(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
	groups, resp, err := w.service.ListSubGroups(gid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list subgroups for group %v: %w", gid, err)
	}
	return groups, resp, nil
}

//nolint:lll // Wrapper method with long signature
func (w *groupsServiceWrapper) ListGroupProjects(gid any, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
	projects, resp, err := w.service.ListGroupProjects(gid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list projects for group %v: %w", gid, err)
	}
	return projects, resp, nil
}

// projectsServiceWrapper wraps the official GitLab projects service.
type projectsServiceWrapper struct {
	service gitlab.ProjectsServiceInterface
}

//nolint:lll // Wrapper method with long signature
func (w *projectsServiceWrapper) GetProject(pid any, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	project, resp, err := w.service.GetProject(pid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to get project %v: %w", pid, err)
	}
	return project, resp, nil
}

// projectImportExportServiceWrapper wraps the official GitLab project import/export service.
type projectImportExportServiceWrapper struct {
	service gitlab.ProjectImportExportServiceInterface
}

//nolint:lll // Wrapper method with long signature
func (w *projectImportExportServiceWrapper) ScheduleExport(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
	resp, err := w.service.ScheduleExport(pid, opt, options...)
	if err != nil {
		return resp, fmt.Errorf("failed to schedule export for project %v: %w", pid, err)
	}
	return resp, nil
}

//nolint:lll // Wrapper method with long signature
func (w *projectImportExportServiceWrapper) ExportStatus(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error) {
	status, resp, err := w.service.ExportStatus(pid, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to get export status for project %v: %w", pid, err)
	}
	return status, resp, nil
}

//nolint:lll // Wrapper method with long signature
func (w *projectImportExportServiceWrapper) ExportDownload(pid any, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error) {
	data, resp, err := w.service.ExportDownload(pid, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to download export for project %v: %w", pid, err)
	}
	return data, resp, nil
}

//nolint:lll // Wrapper method with long signature
func (w *projectImportExportServiceWrapper) ImportFromFile(archive io.Reader, opt *gitlab.ImportFileOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error) {
	status, resp, err := w.service.ImportFromFile(archive, opt, options...)
	if err != nil {
		path, namespace := extractImportOptions(opt)
		return nil, resp, fmt.Errorf("failed to import project from file (path: %s, namespace: %s): %w",
			path, namespace, err)
	}
	return status, resp, nil
}

// extractImportOptions extracts path and namespace from ImportFileOptions.
func extractImportOptions(opt *gitlab.ImportFileOptions) (string, string) {
	path := "<nil>"
	namespace := "<nil>"

	if opt == nil {
		return path, namespace
	}

	if opt.Path != nil {
		path = *opt.Path
	}
	if opt.Namespace != nil {
		namespace = *opt.Namespace
	}

	return path, namespace
}

//nolint:lll // Wrapper method with long signature
func (w *projectImportExportServiceWrapper) ImportStatus(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ImportStatus, *gitlab.Response, error) {
	status, resp, err := w.service.ImportStatus(pid, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to get import status for project %v: %w", pid, err)
	}
	return status, resp, nil
}

// labelsServiceWrapper wraps the official GitLab labels service.
type labelsServiceWrapper struct {
	service gitlab.LabelsServiceInterface
}

//nolint:lll // Wrapper method with long signature
func (w *labelsServiceWrapper) ListLabels(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
	labels, resp, err := w.service.ListLabels(pid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list labels for project %v: %w", pid, err)
	}
	return labels, resp, nil
}

//nolint:lll // Wrapper method with long signature
func (w *labelsServiceWrapper) CreateLabel(pid any, opt *gitlab.CreateLabelOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Label, *gitlab.Response, error) {
	label, resp, err := w.service.CreateLabel(pid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to create label for project %v: %w", pid, err)
	}
	return label, resp, nil
}

// issuesServiceWrapper wraps the official GitLab issues service.
type issuesServiceWrapper struct {
	service gitlab.IssuesServiceInterface
}

//nolint:lll // Wrapper method with long signature
func (w *issuesServiceWrapper) ListProjectIssues(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
	issues, resp, err := w.service.ListProjectIssues(pid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list issues for project %v: %w", pid, err)
	}
	return issues, resp, nil
}

//nolint:lll // Wrapper method with long signature
func (w *issuesServiceWrapper) CreateIssue(pid any, opt *gitlab.CreateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
	issue, resp, err := w.service.CreateIssue(pid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to create issue for project %v: %w", pid, err)
	}
	return issue, resp, nil
}

//nolint:lll // Wrapper method with long signature
func (w *issuesServiceWrapper) UpdateIssue(pid any, issue int64, opt *gitlab.UpdateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
	updatedIssue, resp, err := w.service.UpdateIssue(pid, issue, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to update issue %d for project %v: %w", issue, pid, err)
	}
	return updatedIssue, resp, nil
}

// notesServiceWrapper wraps the official GitLab notes service.
type notesServiceWrapper struct {
	service gitlab.NotesServiceInterface
}

//nolint:lll // Wrapper method with long signature
func (w *notesServiceWrapper) CreateIssueNote(pid any, issue int64, opt *gitlab.CreateIssueNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error) {
	note, resp, err := w.service.CreateIssueNote(pid, issue, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to create note for issue %d in project %v: %w", issue, pid, err)
	}
	return note, resp, nil
}

// commitsServiceWrapper wraps the official GitLab commits service.
type commitsServiceWrapper struct {
	service gitlab.CommitsServiceInterface
}

//nolint:lll // Wrapper method with long signature
func (w *commitsServiceWrapper) ListCommits(pid any, opt *gitlab.ListCommitsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Commit, *gitlab.Response, error) {
	commits, resp, err := w.service.ListCommits(pid, opt, options...)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list commits for project %v: %w", pid, err)
	}
	return commits, resp, nil
}
