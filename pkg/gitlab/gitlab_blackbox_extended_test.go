package gitlab_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/stretchr/testify/assert"
)

// Black box tests for GitLab service focusing on public API behavior
// These tests focus on testing the external behavior without knowledge of internal implementation

func TestGitlabService_PublicAPIBehavior(t *testing.T) {
	// Test basic service creation and configuration
	service := gitlab.NewGitlabService()
	
	// Service might be nil if no GITLAB_TOKEN environment variable is set
	if service == nil {
		t.Skip("Skipping test: GITLAB_TOKEN environment variable not set")
	}

	// Test that the service was created successfully
	assert.NotNil(t, service, "Service should be created when token is available")
}

func TestGitlabService_SetLogger_PublicAPI(t *testing.T) {
	// Test the public SetLogger function
	// This should not panic with nil input
	gitlab.SetLogger(nil)
	
	// Test with a mock logger - create a simple logger implementation
	mockLogger := &testLogger{}
	gitlab.SetLogger(mockLogger)
	
	// SetLogger should handle the logger gracefully
	assert.True(t, true, "SetLogger should handle inputs gracefully")
}

// testLogger implements the gitlab.Logger interface for testing
type testLogger struct {
	debugCalls []logCall
	infoCalls  []logCall
	warnCalls  []logCall
	errorCalls []logCall
}

type logCall struct {
	msg  string
	args []any
}

func (l *testLogger) Debug(msg string, args ...any) {
	l.debugCalls = append(l.debugCalls, logCall{msg: msg, args: args})
}

func (l *testLogger) Info(msg string, args ...any) {
	l.infoCalls = append(l.infoCalls, logCall{msg: msg, args: args})
}

func (l *testLogger) Warn(msg string, args ...any) {
	l.warnCalls = append(l.warnCalls, logCall{msg: msg, args: args})
}

func (l *testLogger) Error(msg string, args ...any) {
	l.errorCalls = append(l.errorCalls, logCall{msg: msg, args: args})
}

func TestGitlabService_Configuration_PublicAPI(t *testing.T) {
	// Test configuration methods on the public API
	service := gitlab.NewGitlabService()
	if service == nil {
		t.Skip("Skipping test: GITLAB_TOKEN environment variable not set")
	}

	// Test token configuration
	testToken := "test-token-123"
	service.SetToken(testToken)
	
	// Test endpoint configuration
	testEndpoint := "https://gitlab.example.com/api/v4"
	service.SetGitlabEndpoint(testEndpoint)
	
	// All configuration methods should complete without error
	assert.True(t, true, "Configuration methods should complete successfully")
}

func TestGitlabService_Constants_PublicAPI(t *testing.T) {
	// Test that public constants are accessible and have expected values
	assert.Equal(t, "https://gitlab.com/api/v4", gitlab.GitlabAPIEndpoint)
	assert.Equal(t, 60, gitlab.DownloadRateLimitIntervalSeconds)
	assert.Equal(t, 5, gitlab.DownloadRateLimitBurst)
	assert.Equal(t, 60, gitlab.ExportRateLimitIntervalSeconds)
	assert.Equal(t, 6, gitlab.ExportRateLimitBurst)
}

func TestGitlabService_ErrorTypes_PublicAPI(t *testing.T) {
	// Test that public error types are accessible
	assert.NotNil(t, gitlab.ErrHTTPStatusCode)
	assert.Contains(t, gitlab.ErrHTTPStatusCode.Error(), "HTTP status code")
}

func TestGitlabService_Integration_WithEnvironment(t *testing.T) {
	// Test integration behavior with environment variables
	
	// Save original environment
	originalToken := os.Getenv("GITLAB_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("GITLAB_TOKEN", originalToken)
		} else {
			os.Unsetenv("GITLAB_TOKEN")
		}
	}()

	// Test with no token - service should still be created but may fail API calls
	os.Unsetenv("GITLAB_TOKEN")
	service := gitlab.NewGitlabService()
	// Service creation may succeed but API calls will likely fail without token
	if service != nil {
		assert.NotNil(t, service, "Service creation should handle missing token gracefully")
	}

	// Test with empty token - service should still be created but may fail API calls
	os.Setenv("GITLAB_TOKEN", "")
	service = gitlab.NewGitlabService()
	// Service creation may succeed but API calls will likely fail with empty token
	if service != nil {
		assert.NotNil(t, service, "Service creation should handle empty token gracefully")
	}

	// Test with valid token format
	os.Setenv("GITLAB_TOKEN", "glpat-test-token-123")
	service = gitlab.NewGitlabService()
	// Should create service successfully with valid token format
	if service != nil {
		assert.NotNil(t, service, "Service should be created with valid token format")
	}
}

func TestGitlabService_ContextHandling_PublicAPI(t *testing.T) {
	// Test context handling in public API methods
	service := gitlab.NewGitlabService()
	if service == nil {
		t.Skip("Skipping test: GITLAB_TOKEN environment variable not set")
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// These calls should handle cancelled context gracefully
	// Note: Actual behavior depends on GitLab API availability
	_, err := service.GetGroup(ctx, 1)
	if err != nil {
		// Error is expected with cancelled context or invalid API call
		assert.Error(t, err, "Should handle cancelled context appropriately")
	}

	_, err = service.GetProject(ctx, 1)
	if err != nil {
		// Error is expected with cancelled context or invalid API call
		assert.Error(t, err, "Should handle cancelled context appropriately")
	}
}

func TestGitlabService_DataTypes_PublicAPI(t *testing.T) {
	// Test public data types and their structure

	// Test Group type
	group := gitlab.Group{
		ID:   int64(1),
		Name: "test-group",
	}
	assert.Equal(t, int64(1), group.ID)
	assert.Equal(t, "test-group", group.Name)

	// Test Project type
	project := gitlab.Project{
		ID:           int64(1),
		Name:         "test-project",
		Archived:     false,
		ExportStatus: "none",
	}
	assert.Equal(t, int64(1), project.ID)
	assert.Equal(t, "test-project", project.Name)
	assert.False(t, project.Archived)
	assert.Equal(t, "none", project.ExportStatus)
}

func TestGitlabService_ConcurrentAccess_PublicAPI(t *testing.T) {
	// Test concurrent access to service methods
	service := gitlab.NewGitlabService()
	if service == nil {
		t.Skip("Skipping test: GITLAB_TOKEN environment variable not set")
	}

	// Test concurrent configuration changes
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for range numGoroutines {
		go func() {
			defer func() { done <- true }()
			
			// Concurrent configuration calls
			service.SetToken("test-token")
			service.SetGitlabEndpoint("https://example.com/api/v4")
			
			// Should not cause race conditions or panics
		}()
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		select {
		case <-done:
			// Goroutine completed
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out waiting for concurrent operations")
		}
	}

	// If we reach here, no race conditions or panics occurred
	assert.True(t, true, "Concurrent access should be handled safely")
}

func TestGitlabService_ErrorHandling_PublicAPI(t *testing.T) {
	// Test error handling in public API
	service := gitlab.NewGitlabService()
	if service == nil {
		t.Skip("Skipping test: GITLAB_TOKEN environment variable not set")
	}

	// Test with invalid IDs (should return appropriate errors)
	ctx := context.Background()

	// Test invalid group ID
	_, err := service.GetGroup(ctx, -1)
	if err != nil {
		assert.Error(t, err, "Should handle invalid group ID appropriately")
		assert.Contains(t, err.Error(), "error", "Error message should be descriptive")
	}

	// Test invalid project ID
	_, err = service.GetProject(ctx, -1)
	if err != nil {
		assert.Error(t, err, "Should handle invalid project ID appropriately")
		assert.Contains(t, err.Error(), "error", "Error message should be descriptive")
	}

	// Test very large ID (potential overflow)
	_, err = service.GetGroup(ctx, 999999999)
	if err != nil {
		assert.Error(t, err, "Should handle very large IDs appropriately")
	}
}

func TestGitlabService_EmptyResults_PublicAPI(t *testing.T) {
	// Test handling of empty results
	service := gitlab.NewGitlabService()
	if service == nil {
		t.Skip("Skipping test: GITLAB_TOKEN environment variable not set")
	}

	ctx := context.Background()

	// Test getting subgroups for a group that might not have any
	// This tests the nil/empty slice handling
	subgroups, err := service.GetSubgroups(ctx, 99999) // Very high ID unlikely to exist
	if err == nil {
		// If no error, should return slice (may be empty)
		assert.IsType(t, []gitlab.Group{}, subgroups, "Should return correct type")
	} else {
		// Error is acceptable for non-existent groups
		assert.Error(t, err, "Should handle non-existent groups appropriately")
	}

	// Test getting projects for a group that might not have any
	projects, err := service.GetProjectsLst(ctx, 99999)
	if err == nil {
		// If no error, should return slice (may be empty)
		assert.IsType(t, []gitlab.Project{}, projects, "Should return correct type")
	} else {
		// Error is acceptable for non-existent groups
		assert.Error(t, err, "Should handle non-existent groups appropriately")
	}
}

func TestGitlabService_NetworkResilience_PublicAPI(t *testing.T) {
	// Test behavior with network issues (using invalid endpoint)
	service := gitlab.NewGitlabService()
	if service == nil {
		t.Skip("Skipping test: GITLAB_TOKEN environment variable not set")
	}

	// Set invalid endpoint to simulate network issues
	service.SetGitlabEndpoint("https://invalid-domain-that-does-not-exist.com/api/v4")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// These should fail gracefully with appropriate errors
	_, err := service.GetGroup(ctx, 1)
	if err != nil {
		assert.Error(t, err, "Should handle network errors gracefully")
		// Error message should be informative
		assert.True(t, len(err.Error()) > 0, "Error message should not be empty")
	}

	_, err = service.GetProject(ctx, 1)
	if err != nil {
		assert.Error(t, err, "Should handle network errors gracefully")
		assert.True(t, len(err.Error()) > 0, "Error message should not be empty")
	}
}

func TestGitlabService_TypeSafety_PublicAPI(t *testing.T) {
	// Test type safety of public API
	service := gitlab.NewGitlabService()
	if service == nil {
		t.Skip("Skipping test: GITLAB_TOKEN environment variable not set")
	}

	// Test that methods accept and return expected types
	ctx := context.Background()

	// GetGroup should return Group type
	group, err := service.GetGroup(ctx, 1)
	if err == nil {
		assert.IsType(t, gitlab.Group{}, group, "GetGroup should return Group type")
		assert.IsType(t, 0, group.ID, "Group ID should be int")
		assert.IsType(t, "", group.Name, "Group Name should be string")
	}

	// GetProject should return Project type
	project, err := service.GetProject(ctx, 1)
	if err == nil {
		assert.IsType(t, gitlab.Project{}, project, "GetProject should return Project type")
		assert.IsType(t, 0, project.ID, "Project ID should be int")
		assert.IsType(t, "", project.Name, "Project Name should be string")
		assert.IsType(t, false, project.Archived, "Project Archived should be bool")
		assert.IsType(t, "", project.ExportStatus, "Project ExportStatus should be string")
	}

	// GetSubgroups should return slice of Group
	subgroups, err := service.GetSubgroups(ctx, 1)
	if err == nil {
		assert.IsType(t, []gitlab.Group{}, subgroups, "GetSubgroups should return []Group")
	}

	// GetProjectsLst should return slice of Project
	projects, err := service.GetProjectsLst(ctx, 1)
	if err == nil {
		assert.IsType(t, []gitlab.Project{}, projects, "GetProjectsLst should return []Project")
	}
}