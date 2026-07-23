package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/constants"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
)

// TestNewApp_AppliesGitlabConfig is a regression test for the bug where the
// backup command ignored gitlabURI/gitlabToken from the resolved config: NewApp
// built the GitLab client with defaults only (https://gitlab.com + the
// GITLAB_TOKEN env var). NewApp must now apply the connection settings from cfg
// so callers cannot forget to wire them up.
func TestNewApp_AppliesGitlabConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		LocalPath:         dir,
		GitlabToken:       "test-token",
		GitlabURI:         "https://gitlab.example.com/api/v4",
		ExportTimeoutMins: constants.DefaultExportTimeoutMins,
	}

	app, err := NewApp(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}
	if app.gitlabService == nil {
		t.Fatal("expected gitlabService to be initialized")
	}
	// NewApp builds a concrete *gitlab.Service; GitlabEndpoint is not part of the
	// injectable BackupService interface, so reach it via a type assertion.
	svc, ok := app.gitlabService.(*gitlab.Service)
	if !ok {
		t.Fatalf("expected gitlabService to be *gitlab.Service, got %T", app.gitlabService)
	}
	if got := svc.GitlabEndpoint(); got != cfg.GitlabURI {
		t.Errorf("expected endpoint %q to be applied from config, got %q", cfg.GitlabURI, got)
	}
}

// TestNewApp_HonorsConfiguredToken verifies that the gitlabToken from the
// resolved config is loaded into the GitLab client and actually sent on API
// requests (as the PRIVATE-TOKEN header), targeting the configured gitlabURI.
// It complements TestNewApp_AppliesGitlabConfig (which only checks the stored
// endpoint) by exercising the real request path end-to-end against a stub
// server, so the token half of the config wiring cannot silently regress.
func TestNewApp_HonorsConfiguredToken(t *testing.T) {
	const wantToken = "cfg-token"

	var (
		hit      bool
		gotToken string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		gotToken = r.Header.Get("PRIVATE-TOKEN")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	cfg := &config.Config{
		LocalPath:         t.TempDir(),
		GitlabToken:       wantToken,
		GitlabURI:         srv.URL,
		ExportTimeoutMins: constants.DefaultExportTimeoutMins,
	}

	app, err := NewApp(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}

	// Drive a real request through the app's GitLab service and inspect the wire.
	// The returned error is irrelevant here: we only care that the request was
	// sent to the configured endpoint carrying the configured token. GetGroup is
	// not on the injectable BackupService interface, so reach it via the concrete
	// service NewApp built.
	svc, ok := app.gitlabService.(*gitlab.Service)
	if !ok {
		t.Fatalf("expected gitlabService to be *gitlab.Service, got %T", app.gitlabService)
	}
	_, _ = svc.GetGroup(context.Background(), 42)

	if !hit {
		t.Fatal("expected the request to reach the configured gitlabURI, but the stub server was never called")
	}
	if gotToken != wantToken {
		t.Errorf("expected PRIVATE-TOKEN header %q loaded from config, got %q", wantToken, gotToken)
	}
}
