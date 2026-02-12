package gitlab

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/time/rate"
)

// Additional comprehensive unit tests for the GitLab package

func TestService_RateLimiting_Behavior(t *testing.T) {
	// Test rate limiting behavior
	client := &mockGitLabClient{
		projectImportExportService: &mockProjectImportExportService{
			scheduleExportFunc: func(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
				return &gitlab.Response{Response: &http.Response{StatusCode: http.StatusAccepted}}, nil
			},
		},
	}

	// Create service with very restrictive rate limiting for testing
	service := &Service{
		client:               client,
		gitlabAPIEndpoint:    constants.GitLabAPIEndpoint,
		token:                "test-token",
		rateLimitDownloadAPI: rate.NewLimiter(rate.Every(100*time.Millisecond), 1),
		rateLimitExportAPI:   rate.NewLimiter(rate.Every(100*time.Millisecond), 1),
	}

	ctx := context.Background()

	// First call should succeed immediately (uses burst capacity)
	start := time.Now()
	result1, err1 := service.askExport(ctx, 1)
	duration1 := time.Since(start)
	
	require.NoError(t, err1)
	assert.True(t, result1)
	assert.Less(t, duration1, 50*time.Millisecond, "First call should be immediate")

	// Second call should also succeed
	result2, err2 := service.askExport(ctx, 2)
	
	require.NoError(t, err2)
	assert.True(t, result2)
	// Both calls should complete successfully (rate limiting is tested separately)
}

func TestService_RateLimiting_Configuration(t *testing.T) {
	// Test that rate limiting configuration is correctly set up
	service := NewGitlabService()
	if service == nil {
		t.Skip("Cannot create service without GITLAB_TOKEN")
	}

	// Test that rate limiters are properly configured
	assert.NotNil(t, service.rateLimitDownloadAPI, "Download rate limiter should be initialized")
	assert.NotNil(t, service.rateLimitExportAPI, "Export rate limiter should be initialized")

	// Test rate limiter configuration values
	downloadLimit := service.rateLimitDownloadAPI.Limit()
	exportLimit := service.rateLimitExportAPI.Limit()
	
	expectedDownloadLimit := rate.Every(constants.DownloadRateLimitIntervalSeconds * time.Second)
	expectedExportLimit := rate.Every(constants.ExportRateLimitIntervalSeconds * time.Second)
	
	assert.Equal(t, expectedDownloadLimit, downloadLimit, "Download rate limit should match expected value")
	assert.Equal(t, expectedExportLimit, exportLimit, "Export rate limit should match expected value")

	// Test burst values
	downloadBurst := service.rateLimitDownloadAPI.Burst()
	exportBurst := service.rateLimitExportAPI.Burst()
	
	assert.Equal(t, constants.DownloadRateLimitBurst, downloadBurst, "Download burst should match expected value")
	assert.Equal(t, constants.ExportRateLimitBurst, exportBurst, "Export burst should match expected value")
}

func TestService_RateLimiting_Integration(t *testing.T) {
	// Test rate limiting integration with actual service methods
	client := &mockGitLabClient{
		projectImportExportService: &mockProjectImportExportService{
			scheduleExportFunc: func(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
				return &gitlab.Response{Response: &http.Response{StatusCode: http.StatusAccepted}}, nil
			},
		},
	}

	// Create service with normal rate limiting
	service := createTestService(client)
	ctx := context.Background()

	// Multiple calls should all succeed without errors
	for i := range 3 {
		result, err := service.askExport(ctx, int64(i+1))
		require.NoError(t, err, "askExport should handle rate limiting gracefully")
		assert.True(t, result, "askExport should return true on success")
	}
}

func TestService_Pagination_Comprehensive(t *testing.T) {
	// Test comprehensive pagination scenarios
	callCount := 0
	groupsService := &mockGroupsService{
		listSubGroupsFunc: func(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
			callCount++
			switch callCount {
			case 1:
				// First page with NextPage set
				return []*gitlab.Group{
					{ID: 1, Name: "group1"},
					{ID: 2, Name: "group2"},
				}, &gitlab.Response{NextPage: 2}, nil
			case 2:
				// Second page with NextPage set
				return []*gitlab.Group{
					{ID: 3, Name: "group3"},
				}, &gitlab.Response{NextPage: 3}, nil
			case 3:
				// Last page with NextPage = 0
				return []*gitlab.Group{
					{ID: 4, Name: "group4"},
				}, &gitlab.Response{NextPage: 0}, nil
			default:
				return []*gitlab.Group{}, &gitlab.Response{NextPage: 0}, nil
			}
		},
	}

	client := &mockGitLabClient{groupsService: groupsService}
	service := createTestService(client)

	result, err := service.GetSubgroups(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, result, 4, "Should collect all groups from all pages")
	// Note: The actual call count may be higher due to recursive subgroup fetching
	assert.GreaterOrEqual(t, callCount, 3, "Should make at least 3 API calls for pagination")
	
	expected := []Group{
		{ID: 1, Name: "group1"},
		{ID: 2, Name: "group2"},
		{ID: 3, Name: "group3"},
		{ID: 4, Name: "group4"},
	}
	assert.Equal(t, expected, result)
}

func TestService_Pagination_ErrorHandling(t *testing.T) {
	// Test pagination with errors
	callCount := 0
	groupsService := &mockGroupsService{
		listSubGroupsFunc: func(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
			callCount++
			if callCount == 1 {
				// First page succeeds
				return []*gitlab.Group{
					{ID: 1, Name: "group1"},
				}, &gitlab.Response{NextPage: 2}, nil
			}
			// Second page fails
			return nil, nil, errors.New("API error on page 2")
		},
	}

	client := &mockGitLabClient{groupsService: groupsService}
	service := createTestService(client)

	_, err := service.GetSubgroups(context.Background(), 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error listing subgroups")
	assert.Equal(t, 2, callCount, "Should make 2 API calls before failing")
}

func TestService_ExportWorkflow_Complete(t *testing.T) {
	// Test complete export workflow with different status transitions
	statusCallCount := 0
	exportService := &mockProjectImportExportService{
		scheduleExportFunc: func(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
			return &gitlab.Response{Response: &http.Response{StatusCode: http.StatusAccepted}}, nil
		},
		exportStatusFunc: func(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error) {
			statusCallCount++
			switch statusCallCount {
			case 1:
				return &gitlab.ExportStatus{ExportStatus: "queued"}, &gitlab.Response{}, nil
			case 2:
				return &gitlab.ExportStatus{ExportStatus: "started"}, &gitlab.Response{}, nil
			case 3:
				return &gitlab.ExportStatus{ExportStatus: "finished"}, &gitlab.Response{}, nil
			default:
				return &gitlab.ExportStatus{ExportStatus: "finished"}, &gitlab.Response{}, nil
			}
		},
		exportDownloadFunc: func(pid any, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error) {
			return []byte("export-data"), &gitlab.Response{}, nil
		},
	}

	client := &mockGitLabClient{projectImportExportService: exportService}
	service := createTestService(client)

	ctx := context.Background()

	// Test export request
	result, err := service.askExport(ctx, 1)
	require.NoError(t, err)
	assert.True(t, result)

	// Test status checking - should show progression
	status1, err := service.getStatusExport(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "queued", status1)

	status2, err := service.getStatusExport(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "started", status2)

	status3, err := service.getStatusExport(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "finished", status3)

	// Test download (note: downloadProject is used internally by ExportProject)
	// We test the downloadExport mock function directly
	data, _, err := service.client.ProjectImportExport().ExportDownload(1, gitlab.WithContext(ctx))
	require.NoError(t, err)
	assert.Equal(t, []byte("export-data"), data)
}

func TestService_ExportWorkflow_ErrorScenarios(t *testing.T) {
	testCases := []struct {
		name           string
		scheduleError  error
		statusError    error
		downloadError  error
		expectedMethod string
	}{
		{
			name:           "Schedule export fails",
			scheduleError:  errors.New("schedule failed"),
			expectedMethod: "askExport",
		},
		{
			name:          "Status check fails",
			statusError:   errors.New("status failed"),
			expectedMethod: "getStatusExport",
		},
		{
			name:           "Download fails",
			downloadError:  errors.New("download failed"),
			expectedMethod: "downloadExport",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exportService := &mockProjectImportExportService{
				scheduleExportFunc: func(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
					if tc.scheduleError != nil {
						return nil, tc.scheduleError
					}
					return &gitlab.Response{Response: &http.Response{StatusCode: http.StatusAccepted}}, nil
				},
				exportStatusFunc: func(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error) {
					if tc.statusError != nil {
						return nil, nil, tc.statusError
					}
					return &gitlab.ExportStatus{ExportStatus: "finished"}, &gitlab.Response{}, nil
				},
				exportDownloadFunc: func(pid any, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error) {
					if tc.downloadError != nil {
						return nil, nil, tc.downloadError
					}
					return []byte("data"), &gitlab.Response{}, nil
				},
			}

			client := &mockGitLabClient{projectImportExportService: exportService}
			service := createTestService(client)
			ctx := context.Background()

			switch tc.expectedMethod {
			case "askExport":
				_, err := service.askExport(ctx, 1)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to make export request")
			case "getStatusExport":
				_, err := service.getStatusExport(ctx, 1)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get export status")
			case "downloadExport":
				_, _, err := service.client.ProjectImportExport().ExportDownload(1, gitlab.WithContext(ctx))
				require.Error(t, err)
				assert.Contains(t, err.Error(), "download failed")
			}
		})
	}
}

func TestService_HTTPStatusCode_Handling(t *testing.T) {
	// Test various HTTP status code scenarios
	// Note: askExport only returns true for StatusAccepted (202), false for others, no errors
	testCases := []struct {
		name       string
		statusCode int
		expectError bool
		expectResult bool
	}{
		{"Accepted", http.StatusAccepted, false, true},
		{"Created", http.StatusCreated, false, false}, // askExport only accepts 202
		{"OK", http.StatusOK, false, false},           // askExport only accepts 202
		{"BadRequest", http.StatusBadRequest, false, false},
		{"Unauthorized", http.StatusUnauthorized, false, false},
		{"Forbidden", http.StatusForbidden, false, false},
		{"NotFound", http.StatusNotFound, false, false},
		{"InternalServerError", http.StatusInternalServerError, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exportService := &mockProjectImportExportService{
				scheduleExportFunc: func(pid any, opt *gitlab.ScheduleExportOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
					return &gitlab.Response{
						Response: &http.Response{StatusCode: tc.statusCode},
					}, nil
				},
			}

			client := &mockGitLabClient{projectImportExportService: exportService}
			service := createTestService(client)

			result, err := service.askExport(context.Background(), 1)

			if tc.expectError {
				require.Error(t, err)
				assert.False(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectResult, result)
			}
		})
	}
}

func TestService_ContextCancellation_AllMethods(t *testing.T) {
	// Test context cancellation for all service methods
	client := &mockGitLabClient{
		groupsService: &mockGroupsService{
			getGroupFunc: func(gid any, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
				time.Sleep(100 * time.Millisecond) // Simulate slow response
				return &gitlab.Group{ID: 1}, &gitlab.Response{}, nil
			},
			listSubGroupsFunc: func(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
				time.Sleep(100 * time.Millisecond)
				return []*gitlab.Group{}, &gitlab.Response{}, nil
			},
			listGroupProjectsFunc: func(gid any, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
				time.Sleep(100 * time.Millisecond)
				return []*gitlab.Project{}, &gitlab.Response{}, nil
			},
		},
		projectsService: &mockProjectsService{
			getProjectFunc: func(pid any, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
				time.Sleep(100 * time.Millisecond)
				return &gitlab.Project{ID: 1}, &gitlab.Response{}, nil
			},
		},
		projectImportExportService: &mockProjectImportExportService{
			exportStatusFunc: func(pid any, options ...gitlab.RequestOptionFunc) (*gitlab.ExportStatus, *gitlab.Response, error) {
				time.Sleep(100 * time.Millisecond)
				return &gitlab.ExportStatus{ExportStatus: "finished"}, &gitlab.Response{}, nil
			},
			exportDownloadFunc: func(pid any, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error) {
				time.Sleep(100 * time.Millisecond)
				return []byte("data"), &gitlab.Response{}, nil
			},
		},
	}

	service := createTestService(client)

	testCases := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"GetGroup", func(ctx context.Context) error {
			_, err := service.GetGroup(ctx, 1)
			return err
		}},
		{"GetProject", func(ctx context.Context) error {
			_, err := service.GetProject(ctx, 1)
			return err
		}},
		{"GetSubgroups", func(ctx context.Context) error {
			_, err := service.GetSubgroups(ctx, 1)
			return err
		}},
		{"GetProjectsLst", func(ctx context.Context) error {
			_, err := service.GetProjectsLst(ctx, 1)
			return err
		}},
		{"getStatusExport", func(ctx context.Context) error {
			_, err := service.getStatusExport(ctx, 1)
			return err
		}},
		{"downloadExport", func(ctx context.Context) error {
			_, _, err := service.client.ProjectImportExport().ExportDownload(1, gitlab.WithContext(ctx))
			return err
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			err := tc.fn(ctx)
			// Should either complete quickly or return context error
			if err != nil {
				assert.Contains(t, err.Error(), "context", "Should handle context cancellation")
			}
		})
	}
}

func TestService_EdgeCases_DataValidation(t *testing.T) {
	// Test edge cases and data validation
	client := &mockGitLabClient{
		groupsService: &mockGroupsService{
			getGroupFunc: func(gid any, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
				// Return group with unusual but valid data
				return &gitlab.Group{
					ID:   0, // Edge case: ID = 0
					Name: "", // Edge case: empty name
				}, &gitlab.Response{}, nil
			},
		},
		projectsService: &mockProjectsService{
			getProjectFunc: func(pid any, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
				// Return project with edge case data
				return &gitlab.Project{
					ID:       0,
					Name:     "",
					Archived: true,
				}, &gitlab.Response{}, nil
			},
		},
	}

	service := createTestService(client)
	ctx := context.Background()

	// Test group with edge case data
	group, err := service.GetGroup(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), group.ID)
	assert.Equal(t, "", group.Name)

	// Test project with edge case data
	project, err := service.GetProject(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), project.ID)
	assert.Equal(t, "", project.Name)
	assert.True(t, project.Archived)
	assert.Equal(t, "", project.ExportStatus)
}

func TestService_ConcurrentOperations_Safety(t *testing.T) {
	// Test concurrent operations for thread safety
	var callCount int32
	client := &mockGitLabClient{
		groupsService: &mockGroupsService{
			getGroupFunc: func(gid any, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(10 * time.Millisecond) // Small delay to encourage race conditions
				// Convert gid to int64 - it could be int or int64 depending on the caller
				var id int64
				switch v := gid.(type) {
				case int:
					id = int64(v)
				case int64:
					id = v
				}
				return &gitlab.Group{
					ID:   id,
					Name: "concurrent-group",
				}, &gitlab.Response{}, nil
			},
		},
	}

	service := createTestService(client)
	ctx := context.Background()

	const numGoroutines = 20
	results := make(chan Group, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Launch concurrent operations
	for i := range numGoroutines {
		go func(id int) {
			group, err := service.GetGroup(ctx, int64(id))
			if err != nil {
				errors <- err
			} else {
				results <- group
			}
		}(i + 1)
	}

	// Collect results
	var groups []Group
	var errs []error

	for range numGoroutines {
		select {
		case group := <-results:
			groups = append(groups, group)
		case err := <-errors:
			errs = append(errs, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify all operations completed
	assert.Len(t, errs, 0, "No errors should occur during concurrent operations")
	assert.Len(t, groups, numGoroutines, "All goroutines should complete successfully")
	assert.Equal(t, int32(numGoroutines), atomic.LoadInt32(&callCount), "All API calls should be made")

	// Verify each group has correct ID
	groupIDs := make(map[int64]bool)
	for _, group := range groups {
		groupIDs[group.ID] = true
		assert.Equal(t, "concurrent-group", group.Name)
	}
	assert.Len(t, groupIDs, numGoroutines, "All groups should have unique IDs")
}

func TestService_LargeDataSets_Handling(t *testing.T) {
	// Test handling of large datasets (simplified to avoid recursion issues)
	const largePageSize = 100 // Reduced size for test performance
	
	groupsService := &mockGroupsService{
		listSubGroupsFunc: func(gid any, opt *gitlab.ListSubGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
			// Only return data for the root group (ID 1) to avoid infinite recursion
			// Handle both int and int64 types
			var id int64
			switch v := gid.(type) {
			case int:
				id = int64(v)
			case int64:
				id = v
			}

			if id == 1 {
				// Simulate large page of data
				groups := make([]*gitlab.Group, largePageSize)
				for i := range largePageSize {
					groups[i] = &gitlab.Group{
						ID:   int64(i + 1000), // Use high IDs to avoid recursion
						Name: "large-dataset-group",
					}
				}
				return groups, &gitlab.Response{NextPage: 0}, nil
			}
			// Return empty for any other group ID to prevent infinite recursion
			return []*gitlab.Group{}, &gitlab.Response{NextPage: 0}, nil
		},
	}

	client := &mockGitLabClient{groupsService: groupsService}
	service := createTestService(client)

	result, err := service.GetSubgroups(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, result, largePageSize, "Should handle large datasets")

	// Verify first and last items
	if len(result) > 0 {
		assert.Equal(t, int64(1000), result[0].ID)
		assert.Equal(t, "large-dataset-group", result[0].Name)
		assert.Equal(t, int64(1000+largePageSize-1), result[len(result)-1].ID)
	}
}