package gitlab

import (
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/constants"
)

// Note: getNextLink tests removed since function was removed
// Pagination is now handled by the official GitLab Go client

func TestNewGitlabService(t *testing.T) {
	// Test that NewGitlabService creates a service with default values
	service := NewGitlabService()
	if service == nil {
		t.Error("expected service to be created, got nil")
	}
	if service.gitlabAPIEndpoint != constants.GitLabAPIEndpoint {
		t.Errorf("expected default endpoint %s, got %s", constants.GitLabAPIEndpoint, service.gitlabAPIEndpoint)
	}
	if service.client == nil {
		t.Error("expected GitLab client to be initialized")
	}
}

func TestGitlabService_SetGitlabEndpoint(t *testing.T) {
	service := NewGitlabService()
	if service == nil {
		t.Fatal("expected service to be created, got nil")
	}

	customEndpoint := "https://gitlab.example.com/api/v4"
	service.SetGitlabEndpoint(customEndpoint)

	if service.gitlabAPIEndpoint != customEndpoint {
		t.Errorf("expected endpoint %s, got %s", customEndpoint, service.gitlabAPIEndpoint)
	}
}

func TestGitlabService_SetToken(t *testing.T) {
	service := NewGitlabService()
	if service == nil {
		t.Fatal("expected service to be created, got nil")
	}

	testToken := "test-token-123"
	service.SetToken(testToken)

	if service.token != testToken {
		t.Errorf("expected token %s, got %s", testToken, service.token)
	}
}
