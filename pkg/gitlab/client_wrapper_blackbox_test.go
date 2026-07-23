package gitlab_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlabAPI "gitlab.com/gitlab-org/api/client-go"
)

// newWrappedClient builds a real GitLab client pointed at srv and wraps it with
// the production NewGitLabClientWrapper, so the retry wrappers are exercised
// end-to-end over HTTP without touching gitlab.com.
func newWrappedClient(t *testing.T, srv *httptest.Server) gitlab.GitLabClient {
	t.Helper()
	// WithoutRetries disables the client-go built-in retryablehttp backoff so the
	// wrapper's own retry logic (retry.go) is the only retry under test — keeping
	// 5xx-based tests fast and deterministic.
	c, err := gitlabAPI.NewClient("test-token", gitlabAPI.WithBaseURL(srv.URL), gitlabAPI.WithoutRetries())
	require.NoError(t, err)
	return gitlab.NewGitLabClientWrapper(c)
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// apiRouter returns a handler that serves minimal valid responses for every
// endpoint the wrappers touch. Order matters: more specific suffixes first.
func apiRouter() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v4/")
		switch {
		case r.Method == http.MethodGet && path == "groups/456":
			writeJSON(w, http.StatusOK, `{"id":456,"name":"grp"}`)
		case r.Method == http.MethodGet && path == "groups/456/subgroups":
			writeJSON(w, http.StatusOK, `[{"id":789,"name":"sub"}]`)
		case r.Method == http.MethodGet && path == "groups/456/projects":
			writeJSON(w, http.StatusOK, `[{"id":1,"name":"p1"}]`)
		case r.Method == http.MethodGet && path == "projects/123/export/download":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("BINARY-EXPORT"))
		case r.Method == http.MethodPost && path == "projects/123/export":
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && path == "projects/123/export":
			writeJSON(w, http.StatusOK, `{"export_status":"finished"}`)
		case r.Method == http.MethodPost && path == "projects/import":
			writeJSON(w, http.StatusCreated, `{"id":77,"import_status":"scheduled"}`)
		case r.Method == http.MethodGet && path == "projects/123/import":
			writeJSON(w, http.StatusOK, `{"id":77,"import_status":"finished"}`)
		case r.Method == http.MethodGet && path == "projects/123/repository/commits":
			writeJSON(w, http.StatusOK, `[{"id":"abc123"}]`)
		case r.Method == http.MethodGet && path == "projects/123/labels":
			writeJSON(w, http.StatusOK, `[{"id":10,"name":"bug"}]`)
		case r.Method == http.MethodPost && path == "projects/123/labels":
			writeJSON(w, http.StatusCreated, `{"id":10,"name":"bug"}`)
		case r.Method == http.MethodPost && path == "projects/123/issues/5/notes":
			writeJSON(w, http.StatusCreated, `{"id":99,"body":"note"}`)
		case r.Method == http.MethodGet && path == "projects/123/issues":
			writeJSON(w, http.StatusOK, `[{"id":50,"iid":5}]`)
		case r.Method == http.MethodPost && path == "projects/123/issues":
			writeJSON(w, http.StatusCreated, `{"id":50,"iid":5}`)
		case r.Method == http.MethodPut && path == "projects/123/issues/5":
			writeJSON(w, http.StatusOK, `{"id":50,"iid":5,"title":"updated"}`)
		case r.Method == http.MethodGet && path == "projects/123":
			writeJSON(w, http.StatusOK, `{"id":123,"name":"proj"}`)
		default:
			writeJSON(w, http.StatusNotFound, `{"message":"404 Not Found"}`)
		}
	}
}

func TestWrapper_Groups(t *testing.T) {
	srv := httptest.NewServer(apiRouter())
	defer srv.Close()
	client := newWrappedClient(t, srv)
	ctx := context.Background()

	group, _, err := client.Groups().GetGroup(ctx, 456, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(456), group.ID)

	subs, _, err := client.Groups().ListSubGroups(ctx, 456, nil)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	assert.Equal(t, int64(789), subs[0].ID)

	projects, _, err := client.Groups().ListGroupProjects(ctx, 456, nil)
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, int64(1), projects[0].ID)
}

func TestWrapper_Projects(t *testing.T) {
	srv := httptest.NewServer(apiRouter())
	defer srv.Close()
	client := newWrappedClient(t, srv)

	project, _, err := client.Projects().GetProject(context.Background(), 123, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(123), project.ID)
}

func TestWrapper_ProjectImportExport(t *testing.T) {
	srv := httptest.NewServer(apiRouter())
	defer srv.Close()
	client := newWrappedClient(t, srv)
	ctx := context.Background()
	ie := client.ProjectImportExport()

	resp, err := ie.ScheduleExport(ctx, 123, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	status, _, err := ie.ExportStatus(ctx, 123)
	require.NoError(t, err)
	assert.Equal(t, "finished", status.ExportStatus)

	var buf bytes.Buffer
	_, err = ie.ExportDownloadStream(ctx, 123, &buf)
	require.NoError(t, err)
	assert.Equal(t, "BINARY-EXPORT", buf.String())

	imp, _, err := ie.ImportFromFile(ctx, bytes.NewReader([]byte("archive")), &gitlabAPI.ImportFileOptions{
		Namespace: gitlabAPI.Ptr("ns"),
		Path:      gitlabAPI.Ptr("proj"),
		Name:      gitlabAPI.Ptr("proj"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(77), imp.ID)

	impStatus, _, err := ie.ImportStatus(ctx, 123)
	require.NoError(t, err)
	assert.Equal(t, "finished", impStatus.ImportStatus)
}

func TestWrapper_ImportFromFile_ErrorEnrichesContext(t *testing.T) {
	// A failing import must surface the path/namespace in the wrapped error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusBadRequest, `{"message":"bad request"}`)
	}))
	defer srv.Close()
	client := newWrappedClient(t, srv)

	_, _, err := client.ProjectImportExport().ImportFromFile(context.Background(), bytes.NewReader([]byte("x")),
		&gitlabAPI.ImportFileOptions{Namespace: gitlabAPI.Ptr("myns"), Path: gitlabAPI.Ptr("mypath")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mypath")
	assert.Contains(t, err.Error(), "myns")
}

func TestWrapper_LabelsIssuesNotesCommits(t *testing.T) {
	srv := httptest.NewServer(apiRouter())
	defer srv.Close()
	client := newWrappedClient(t, srv)
	ctx := context.Background()

	labels, _, err := client.Labels().ListLabels(ctx, 123, nil)
	require.NoError(t, err)
	assert.Len(t, labels, 1)

	label, _, err := client.Labels().CreateLabel(ctx, 123, &gitlabAPI.CreateLabelOptions{
		Name:  gitlabAPI.Ptr("bug"),
		Color: gitlabAPI.Ptr("#ff0000"),
	})
	require.NoError(t, err)
	assert.Equal(t, "bug", label.Name)

	issues, _, err := client.Issues().ListProjectIssues(ctx, 123, nil)
	require.NoError(t, err)
	assert.Len(t, issues, 1)

	issue, _, err := client.Issues().CreateIssue(ctx, 123, &gitlabAPI.CreateIssueOptions{Title: gitlabAPI.Ptr("t")})
	require.NoError(t, err)
	assert.Equal(t, int64(5), issue.IID)

	updated, _, err := client.Issues().UpdateIssue(ctx, 123, 5, &gitlabAPI.UpdateIssueOptions{
		Title: gitlabAPI.Ptr("updated"),
	})
	require.NoError(t, err)
	assert.Equal(t, "updated", updated.Title)

	note, _, err := client.Notes().CreateIssueNote(ctx, 123, 5, &gitlabAPI.CreateIssueNoteOptions{
		Body: gitlabAPI.Ptr("note"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(99), note.ID)

	commits, _, err := client.Commits().ListCommits(ctx, 123, nil)
	require.NoError(t, err)
	require.Len(t, commits, 1)
	assert.Equal(t, "abc123", commits[0].ID)
}

func TestWrapper_RetryThenSuccess(t *testing.T) {
	// First response is a retryable 500; the retry wrapper must recover on the
	// second attempt and return the successful payload.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			writeJSON(w, http.StatusInternalServerError, `{"message":"boom"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"id":123,"name":"proj"}`)
	}))
	defer srv.Close()
	client := newWrappedClient(t, srv)

	project, _, err := client.Projects().GetProject(context.Background(), 123, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(123), project.ID)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls), "should have retried exactly once")
}

func TestWrapper_NonRetryableErrorReturnsImmediately(t *testing.T) {
	// A 404 is not retryable: the wrapper must return after a single call.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		writeJSON(w, http.StatusNotFound, `{"message":"404 Not Found"}`)
	}))
	defer srv.Close()
	client := newWrappedClient(t, srv)

	_, _, err := client.Projects().GetProject(context.Background(), 123, nil)
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "404 must not be retried")
}

func TestWrapper_RetryStopsOnContextCancellation(t *testing.T) {
	// Persistent 500 keeps the wrapper retrying; a short context deadline makes
	// the retry loop abort via the ctx.Done branch instead of backing off ~1s.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusInternalServerError, `{"message":"still failing"}`)
	}))
	defer srv.Close()
	client := newWrappedClient(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := client.Projects().GetProject(ctx, 123, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWrapper_ScheduleExport_RetryThenSuccess(t *testing.T) {
	// Exercises retryResponseOnly's retry loop (ScheduleExport returns Response only).
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			writeJSON(w, http.StatusServiceUnavailable, `{"message":"try later"}`)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	client := newWrappedClient(t, srv)

	resp, err := client.ProjectImportExport().ScheduleExport(context.Background(), 123, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}
