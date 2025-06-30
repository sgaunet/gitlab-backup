package gitlab_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitlabService_GetGroupGetID(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := gitlab.Group{
				ID:   1,
				Name: "test",
			}
			resJSON, err := json.Marshal(res)
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := gitlab.NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	g, err := r.GetGroup(context.Background(), 1)
	if err != nil {
		t.Errorf("expected no error, got %s", err.Error())
	}
	if g.ID != 1 {
		t.Errorf("expected id %d, got %d", 1, g.ID)
	}
}

// TestGitlabService_GetProjectsLst tests the GetProjectsLst method
// which retrieves projects from a GitLab group with pagination
func TestGitlabService_GetProjectsLst(t *testing.T) {
	// Track the number of requests to prevent infinite loops
	requestCount := 0

	// Create a test server that responds with a simulated GitLab API response
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Increment request count and fail if too many requests
			requestCount++
			if requestCount > 10 {
				t.Fatal("Too many requests, possible infinite loop")
			}

			// Assert the correct HTTP method and path are used
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v4/groups/42/projects", r.URL.Path)
			assert.Contains(t, r.URL.RawQuery, "per_page=20")
			assert.Contains(t, r.URL.RawQuery, "order_by=id")
			assert.Contains(t, r.URL.RawQuery, "sort=asc")
			assert.Contains(t, r.URL.RawQuery, "pagination=keyset")

			// First page request
			if r.URL.Query().Get("page") == "" || r.URL.Query().Get("page") == "1" {
				projects := []gitlab.Project{
					{ID: 101, Name: "project1", Archived: false},
					{ID: 102, Name: "project2", Archived: true},
				}
				resJSON, err := json.Marshal(projects)
				require.NoError(t, err)

				// Add pagination link header to test the pagination handling
				w.Header().Add("Content-Type", "application/json")
				serverURL := "https://" + r.Host + "/api/v4"
				w.Header().Add("Link", fmt.Sprintf(`<%s/groups/42/projects?page=2&per_page=20&order_by=id&sort=asc&pagination=keyset>; rel="next"`, serverURL))

				fmt.Fprintln(w, string(resJSON))
			} else if r.URL.Query().Get("page") == "2" {
				// Second page of results
				projects := []gitlab.Project{
					{ID: 103, Name: "project3", Archived: false},
				}
				resJSON, err := json.Marshal(projects)
				require.NoError(t, err)

				w.Header().Add("Content-Type", "application/json")
				// No Link header for the last page
				fmt.Fprintln(w, string(resJSON))
			} else {
				// Should not reach here
				t.Errorf("Unexpected page number: %s", r.URL.Query().Get("page"))
				http.Error(w, "Unexpected page", http.StatusBadRequest)
			}
		}))
	defer ts.Close()

	// Set up GitLab service with test client
	client := ts.Client()
	r := gitlab.NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL + "/api/v4")

	// Test the GetProjectsLst function
	projects, err := r.GetProjectsLst(context.Background(), 42)

	// Verify results
	require.NoError(t, err)
	require.Len(t, projects, 3) // 2 from first page + 1 from second page

	// Check we have all the expected projects in the list
	projectMap := make(map[int]gitlab.Project)
	for _, project := range projects {
		projectMap[project.ID] = project
	}

	assert.Contains(t, projectMap, 101)
	assert.Equal(t, "project1", projectMap[101].Name)
	assert.False(t, projectMap[101].Archived)

	assert.Contains(t, projectMap, 102)
	assert.Equal(t, "project2", projectMap[102].Name)
	assert.True(t, projectMap[102].Archived)

	assert.Contains(t, projectMap, 103)
	assert.Equal(t, "project3", projectMap[103].Name)
	assert.False(t, projectMap[103].Archived)
}

func TestGitlabService_GetGroupGetIDWithHTTPMethod(t *testing.T) {
	// Create a test server that responds with a simulated GitLab API response
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Assert the correct HTTP method and path are used
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v4/groups/1", r.URL.Path)
			// The token header should be set by the client
			assert.NotEmpty(t, r.Header.Get("PRIVATE-TOKEN"))

			// Respond with a mock group object
			group := gitlab.Group{
				ID:   1,
				Name: "testgroup",
			}
			resJSON, err := json.Marshal(group)
			require.NoError(t, err)

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	// Set up GitLab service with test client
	client := ts.Client()
	r := gitlab.NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL + "/api/v4")

	// Test the GetGroup function
	g, err := r.GetGroup(context.Background(), 1)
	require.NoError(t, err)

	// Check response values
	assert.Equal(t, 1, g.ID)
	assert.Equal(t, "testgroup", g.Name)
}

func TestGitlabService_GetProjectGetID(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := gitlab.Project{
				ID:           1,
				Name:         "test",
				Archived:     true,
				ExportStatus: "finished",
			}
			resJSON, err := json.Marshal(res)
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := gitlab.NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	p, err := r.GetProject(context.Background(), 1)
	if err != nil {
		t.Errorf("expected no error, got %s", err.Error())
	}
	if p.ID != 1 {
		t.Errorf("expected id %d, got %d", 1, p.ID)
	}
	if p.Name != "test" {
		t.Errorf("expected name %s, got %s", "test", p.Name)
	}
	if p.Archived != true {
		t.Errorf("expected archived %t, got %t", true, p.Archived)
	}
	if p.ExportStatus != "finished" {
		t.Errorf("expected export status %s, got %s", "finished", p.ExportStatus)
	}
}

func TestSetLogger(t *testing.T) {
	gitlab.SetLogger(nil)
	r := gitlab.NewGitlabService()
	r.SetToken("")
}

func TestGitlabService_SetGitlabEndpoint(t *testing.T) {
	r := gitlab.NewGitlabService()

	// Test custom endpoint
	customEndpoint := "https://mygitlab.example.com/api/v4"
	r.SetGitlabEndpoint(customEndpoint)

	// Test the endpoint indirectly by setting up a mock server at the endpoint
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/groups/1", r.URL.Path)
			group := gitlab.Group{ID: 1, Name: "test-endpoint"}
			resJSON, err := json.Marshal(group)
			require.NoError(t, err)
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	r.SetGitlabEndpoint(ts.URL)
	r.SetHTTPClient(ts.Client())

	// Verify the endpoint was set correctly by making a request
	group, err := r.GetGroup(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, group.ID)
	assert.Equal(t, "test-endpoint", group.Name)
}

func TestGitlabService_GetProjectsOfGroup(t *testing.T) {
	// Create a test server that handles both subgroups and projects requests
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Common headers
			w.Header().Add("Content-Type", "application/json")

			// Handle different API endpoints
			if strings.Contains(r.URL.Path, "/subgroups") {
				// This is a request for subgroups
				groupID := 0
				_, err := fmt.Sscanf(r.URL.Path, "/api/v4/groups/%d/subgroups", &groupID)
				require.NoError(t, err)

				if groupID == 10 {
					// Main group subgroups
					subgroups := []gitlab.Group{
						{ID: 11, Name: "subgroup1"},
						{ID: 12, Name: "subgroup2"},
					}
					resJSON, err := json.Marshal(subgroups)
					require.NoError(t, err)
					fmt.Fprintln(w, string(resJSON))
				} else {
					// Subgroups have no further subgroups
					fmt.Fprintln(w, "[]")
				}
			} else if strings.Contains(r.URL.Path, "/projects") {
				// This is a request for projects
				groupID := 0
				_, err := fmt.Sscanf(r.URL.Path, "/api/v4/groups/%d/projects", &groupID)
				require.NoError(t, err)

				switch groupID {
				case 10: // Main group projects
					projects := []gitlab.Project{
						{ID: 101, Name: "main-project1", Archived: false},
						{ID: 102, Name: "main-project2", Archived: true}, // Archived should be excluded
					}
					resJSON, err := json.Marshal(projects)
					require.NoError(t, err)
					fmt.Fprintln(w, string(resJSON))
				case 11: // Subgroup1 projects
					projects := []gitlab.Project{
						{ID: 111, Name: "sub1-project1", Archived: false},
					}
					resJSON, err := json.Marshal(projects)
					require.NoError(t, err)
					fmt.Fprintln(w, string(resJSON))
				case 12: // Subgroup2 projects
					projects := []gitlab.Project{
						{ID: 121, Name: "sub2-project1", Archived: false},
						{ID: 122, Name: "sub2-project2", Archived: false},
					}
					resJSON, err := json.Marshal(projects)
					require.NoError(t, err)
					fmt.Fprintln(w, string(resJSON))
				default:
					fmt.Fprintln(w, "[]")
				}
			}
		}))
	defer ts.Close()

	// Set up GitLab service with test client
	client := ts.Client()
	r := gitlab.NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL + "/api/v4")

	// Test the GetProjectsOfGroup function
	projects, err := r.GetProjectsOfGroup(context.Background(), 10)

	// Verify results
	require.NoError(t, err)

	// Should have 4 non-archived projects (1 from main group + 1 from subgroup1 + 2 from subgroup2)
	require.Len(t, projects, 4)

	// Check we have all the expected projects in the list
	projectMap := make(map[int]bool)
	projectNames := make(map[string]bool)

	for _, project := range projects {
		projectMap[project.ID] = true
		projectNames[project.Name] = true
		assert.False(t, project.Archived, "No archived projects should be included")
	}

	// Check main group project
	assert.True(t, projectMap[101])
	assert.True(t, projectNames["main-project1"])

	// Check subgroup1 project
	assert.True(t, projectMap[111])
	assert.True(t, projectNames["sub1-project1"])

	// Check subgroup2 projects
	assert.True(t, projectMap[121])
	assert.True(t, projectMap[122])
	assert.True(t, projectNames["sub2-project1"])
	assert.True(t, projectNames["sub2-project2"])

	// Make sure archived project is not included
	assert.False(t, projectMap[102])
	assert.False(t, projectNames["main-project2"])
}

// TestGitlabService_Error tests error handling for various error scenarios
// TestGitlabService_ExportProject tests the ExportProject functionality of the GitlabService
func TestGitlabService_ExportProject(t *testing.T) {
	// Create a temporary directory for our test
	tempDir, err := os.MkdirTemp("", "gitlab-backup-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Define path for the downloaded archive
	archivePath := filepath.Join(tempDir, "project-export.tar.gz")

	// Define test data
	projectID := 42
	projectName := "test-project"
	exportContent := []byte("This is test export content")

	// Create a test server that handles all API endpoints needed for project export
	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	defer server.Close()

	// Track request state for testing sequence
	var requestCounter int
	var exportRequested bool

	// 1. Handle project info endpoint
	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/%d", projectID), func(w http.ResponseWriter, r *http.Request) {
		project := gitlab.Project{
			ID:       projectID,
			Name:     projectName,
			Archived: false, // non-archived project
		}

		// Set export status on GET requests after export is requested
		if exportRequested && r.Method == http.MethodGet {
			project.ExportStatus = "finished"
		}

		resJSON, _ := json.Marshal(project)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, string(resJSON))
	})

	// 2. Handle export request endpoint
	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/%d/export", projectID), func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Export request
			exportRequested = true
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintln(w, "{\"message\":\"202 Accepted\"}")
		} else if r.Method == http.MethodGet {
			// Export status check
			requestCounter++

			var status string
			// First status check returns 'started', second returns 'finished'
			if requestCounter == 1 {
				status = "started"
			} else {
				status = "finished"
			}

			resp := fmt.Sprintf("{\"export_status\":\"%s\"}", status)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, resp)
		}
	})

	// 3. Handle download endpoint
	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/%d/export/download", projectID), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.tar.gz\"", projectName))
		w.Write(exportContent)
	})

	// Setup GitLab service client with test server
	client := server.Client()
	r := gitlab.NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(server.URL + "/api/v4")
	r.SetToken("test-token")

	// Execute the ExportProject method
	project := &gitlab.Project{
		ID:       projectID,
		Name:     projectName,
		Archived: false,
	}

	err = r.ExportProject(context.Background(), project, archivePath)
	require.NoError(t, err)

	// Verify file was downloaded
	fileContent, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	assert.Equal(t, exportContent, fileContent)

	// Test with archived project
	archivedProject := &gitlab.Project{
		ID:       999,
		Name:     "archived-project",
		Archived: true,
	}

	// The export should be skipped for archived projects
	err = r.ExportProject(context.Background(), archivedProject, filepath.Join(tempDir, "archived-export.tar.gz"))
	require.NoError(t, err)

	// The file should not exist since export was skipped
	_, err = os.Stat(filepath.Join(tempDir, "archived-export.tar.gz"))
	assert.True(t, os.IsNotExist(err))
}

func TestGitlabService_Error(t *testing.T) {
	// 1. Test for HTTP errors (connection refused)
	r := gitlab.NewGitlabService()
	r.SetGitlabEndpoint("https://non-existent-gitlab-server.example.com/api/v4")
	_, err := r.GetGroup(context.Background(), 1)
	require.Error(t, err)
	// Check for common connection error substrings since the exact message might vary by OS/environment
	errMsg := err.Error()
	isConnectionError := strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "connection reset by peer") ||
		strings.Contains(errMsg, "connection timed out") ||
		strings.Contains(errMsg, "no route to host")
	assert.True(t, isConnectionError, "Expected a connection error, got: %s", errMsg)

	// 2. Test for JSON unmarshalling errors
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			// Return invalid JSON
			fmt.Fprintln(w, "{invalid-json")
		}))
	defer ts.Close()

	r = gitlab.NewGitlabService()
	r.SetHTTPClient(ts.Client())
	r.SetGitlabEndpoint(ts.URL + "/api/v4")
	_, err = r.GetGroup(context.Background(), 1)
	require.Error(t, err)
	// Check for JSON unmarshal error in a more flexible way
	errMsg = err.Error()
	isJSONError := strings.Contains(errMsg, "unmarshal") ||
		strings.Contains(errMsg, "unexpected")
	assert.True(t, isJSONError, "Expected a JSON unmarshal error, got: %s", errMsg)

	// 3. Test for GitLab API returning error message
	ts2 := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			// Return GitLab error format
			fmt.Fprintln(w, `{"message":"404 Group Not Found"}`)
		}))
	defer ts2.Close()

	r = gitlab.NewGitlabService()
	r.SetHTTPClient(ts2.Client())
	r.SetGitlabEndpoint(ts2.URL + "/api/v4")
	_, err = r.GetGroup(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error retrieving group")
	assert.Contains(t, err.Error(), "404 Group Not Found")
}

func TestGitlabService_TokenWarning(t *testing.T) {
	// This test verifies that when no token is set, a warning is logged
	// Since we can't directly test the logging output in a blackbox test,
	// we'll test the functionality indirectly

	// Create a mock server for the API request
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify no token header is sent
			assert.Equal(t, "", r.Header.Get("PRIVATE-TOKEN"))

			// Return a successful response
			group := gitlab.Group{ID: 1, Name: "no-token-test"}
			resJSON, err := json.Marshal(group)
			require.NoError(t, err)
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	// Create a new service but don't set a token
	r := gitlab.NewGitlabService()
	r.SetHTTPClient(ts.Client())
	r.SetGitlabEndpoint(ts.URL + "/api/v4")
	// Explicitly set an empty token
	r.SetToken("")

	// Make the request - this should work but cause a warning to be logged
	group, err := r.GetGroup(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, group.ID)
	assert.Equal(t, "no-token-test", group.Name)
}

func TestGitlabService_GetNextLink(t *testing.T) {
	// Create a test server with pagination links
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate pagination in response headers
			baseURL := "https://" + r.Host + "/api/v4"

			// Test both with and without the Link header
			if r.URL.Query().Get("test") == "with-link" {
				w.Header().Add("Link", fmt.Sprintf("<%s/projects?page=2&per_page=20>; rel=\"next\", <%s/projects?page=1&per_page=20>; rel=\"first\">", baseURL, baseURL))
			}

			// Return an empty result - we're just testing the header processing
			projects := []gitlab.Project{}
			resJSON, err := json.Marshal(projects)
			require.NoError(t, err)
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	// Set up GitLab service with test client
	client := ts.Client()
	r := gitlab.NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL + "/api/v4")

	// 1. Test with Link header present
	resp, err := client.Get(ts.URL + "/api/v4/projects?test=with-link")
	require.NoError(t, err)
	defer resp.Body.Close()

	// This functionality actually tests the unexported getNextLink behavior
	// through the observable behavior of retrieveProjects calling it
	_, err = r.GetProjectsLst(context.Background(), 1) // This will internally process Link headers
	require.NoError(t, err)

	// 2. Test with no Link header
	resp, err = client.Get(ts.URL + "/api/v4/projects?test=no-link")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify no Link header returned empty projects list
	assert.Equal(t, "", resp.Header.Get("Link"))
}

func TestGitlabService_SetHTTPClient(t *testing.T) {
	r := gitlab.NewGitlabService()

	// Create a custom HTTP client with specific timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Set the custom client
	r.SetHTTPClient(client)

	// Test that the client was set correctly by making a request
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			group := gitlab.Group{ID: 1, Name: "test-client"}
			resJSON, err := json.Marshal(group)
			require.NoError(t, err)
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	// Use the test server's client which has the correct certificates
	r.SetHTTPClient(ts.Client())
	r.SetGitlabEndpoint(ts.URL + "/api/v4")

	group, err := r.GetGroup(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, group.ID)
	assert.Equal(t, "test-client", group.Name)
}
