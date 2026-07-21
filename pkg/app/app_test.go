package app

import (
	"context"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/constants"
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
	if got := app.gitlabService.GitlabEndpoint(); got != cfg.GitlabURI {
		t.Errorf("expected endpoint %q to be applied from config, got %q", cfg.GitlabURI, got)
	}
}
