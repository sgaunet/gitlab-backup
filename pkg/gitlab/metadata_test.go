package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/time/rate"
)

// Mock implementations for Labels and Issues services

type mockLabelsService struct {
	listLabelsFunc  func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error)
	createLabelFunc func(pid any, opt *gitlab.CreateLabelOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Label, *gitlab.Response, error)
}

func (m *mockLabelsService) ListLabels(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
	if m.listLabelsFunc != nil {
		return m.listLabelsFunc(pid, opt, options...)
	}
	return nil, nil, nil
}

func (m *mockLabelsService) CreateLabel(pid any, opt *gitlab.CreateLabelOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Label, *gitlab.Response, error) {
	if m.createLabelFunc != nil {
		return m.createLabelFunc(pid, opt, options...)
	}
	return nil, nil, nil
}

type mockIssuesService struct {
	listProjectIssuesFunc func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error)
	createIssueFunc       func(pid any, opt *gitlab.CreateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error)
	updateIssueFunc       func(pid any, issue int64, opt *gitlab.UpdateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error)
}

func (m *mockIssuesService) ListProjectIssues(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
	if m.listProjectIssuesFunc != nil {
		return m.listProjectIssuesFunc(pid, opt, options...)
	}
	return nil, nil, nil
}

func (m *mockIssuesService) CreateIssue(pid any, opt *gitlab.CreateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
	if m.createIssueFunc != nil {
		return m.createIssueFunc(pid, opt, options...)
	}
	return nil, nil, nil
}

func (m *mockIssuesService) UpdateIssue(pid any, issue int64, opt *gitlab.UpdateIssueOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
	if m.updateIssueFunc != nil {
		return m.updateIssueFunc(pid, issue, opt, options...)
	}
	return nil, nil, nil
}

func TestService_ExportLabels_Success(t *testing.T) {
	// Create mock labels
	mockLabels := []*gitlab.Label{
		{ID: 1, Name: "bug", Color: "#FF0000"},
		{ID: 2, Name: "feature", Color: "#00FF00"},
	}

	labelsService := &mockLabelsService{
		listLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
			return mockLabels, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
		},
	}

	client := &mockGitLabClient{
		labelsService: labelsService,
	}

	service := &Service{
		client:               client,
		rateLimitMetadataAPI: rate.NewLimiter(rate.Inf, 1),
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "labels-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Export labels
	ctx := context.Background()
	err = service.ExportLabels(ctx, 123, tmpFile.Name())
	require.NoError(t, err)

	// Verify file contents
	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var labels []*gitlab.Label
	err = json.Unmarshal(data, &labels)
	require.NoError(t, err)

	assert.Len(t, labels, 2)
	assert.Equal(t, "bug", labels[0].Name)
	assert.Equal(t, "feature", labels[1].Name)
}

func TestService_ExportLabels_Pagination(t *testing.T) {
	// Test pagination with multiple pages
	page1Labels := []*gitlab.Label{
		{ID: 1, Name: "label1"},
	}
	page2Labels := []*gitlab.Label{
		{ID: 2, Name: "label2"},
	}

	callCount := 0
	labelsService := &mockLabelsService{
		listLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
			callCount++
			if callCount == 1 {
				return page1Labels, &gitlab.Response{
					Response: &http.Response{StatusCode: http.StatusOK},
					NextPage: 2,
				}, nil
			}
			return page2Labels, &gitlab.Response{
				Response: &http.Response{StatusCode: http.StatusOK},
				NextPage: 0,
			}, nil
		},
	}

	client := &mockGitLabClient{
		labelsService: labelsService,
	}

	service := &Service{
		client:               client,
		rateLimitMetadataAPI: rate.NewLimiter(rate.Inf, 1),
	}

	tmpFile, err := os.CreateTemp("", "labels-pagination-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	ctx := context.Background()
	err = service.ExportLabels(ctx, 123, tmpFile.Name())
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var labels []*gitlab.Label
	err = json.Unmarshal(data, &labels)
	require.NoError(t, err)

	assert.Len(t, labels, 2)
	assert.Equal(t, 2, callCount, "Should have made 2 API calls for pagination")
}

func TestService_ExportLabels_Error(t *testing.T) {
	labelsService := &mockLabelsService{
		listLabelsFunc: func(pid any, opt *gitlab.ListLabelsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
			return nil, nil, errors.New("API error")
		},
	}

	client := &mockGitLabClient{
		labelsService: labelsService,
	}

	service := &Service{
		client:               client,
		rateLimitMetadataAPI: rate.NewLimiter(rate.Inf, 1),
	}

	tmpFile, err := os.CreateTemp("", "labels-error-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	ctx := context.Background()
	err = service.ExportLabels(ctx, 123, tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list labels")
}

func TestService_ExportIssues_Success(t *testing.T) {
	mockIssues := []*gitlab.Issue{
		{ID: 1, Title: "Issue 1", State: "opened"},
		{ID: 2, Title: "Issue 2", State: "closed"},
	}

	issuesService := &mockIssuesService{
		listProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
			return mockIssues, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
		},
	}

	client := &mockGitLabClient{
		issuesService: issuesService,
	}

	service := &Service{
		client:               client,
		rateLimitMetadataAPI: rate.NewLimiter(rate.Inf, 1),
	}

	tmpFile, err := os.CreateTemp("", "issues-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	ctx := context.Background()
	err = service.ExportIssues(ctx, 123, tmpFile.Name())
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var issues []*gitlab.Issue
	err = json.Unmarshal(data, &issues)
	require.NoError(t, err)

	assert.Len(t, issues, 2)
	assert.Equal(t, "Issue 1", issues[0].Title)
	assert.Equal(t, "Issue 2", issues[1].Title)
}

func TestService_ExportIssues_Pagination(t *testing.T) {
	page1Issues := []*gitlab.Issue{
		{ID: 1, Title: "Issue 1"},
	}
	page2Issues := []*gitlab.Issue{
		{ID: 2, Title: "Issue 2"},
	}

	callCount := 0
	issuesService := &mockIssuesService{
		listProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
			callCount++
			if callCount == 1 {
				return page1Issues, &gitlab.Response{
					Response: &http.Response{StatusCode: http.StatusOK},
					NextPage: 2,
				}, nil
			}
			return page2Issues, &gitlab.Response{
				Response: &http.Response{StatusCode: http.StatusOK},
				NextPage: 0,
			}, nil
		},
	}

	client := &mockGitLabClient{
		issuesService: issuesService,
	}

	service := &Service{
		client:               client,
		rateLimitMetadataAPI: rate.NewLimiter(rate.Inf, 1),
	}

	tmpFile, err := os.CreateTemp("", "issues-pagination-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	ctx := context.Background()
	err = service.ExportIssues(ctx, 123, tmpFile.Name())
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var issues []*gitlab.Issue
	err = json.Unmarshal(data, &issues)
	require.NoError(t, err)

	assert.Len(t, issues, 2)
	assert.Equal(t, 2, callCount, "Should have made 2 API calls for pagination")
}

func TestService_ExportIssues_Error(t *testing.T) {
	issuesService := &mockIssuesService{
		listProjectIssuesFunc: func(pid any, opt *gitlab.ListProjectIssuesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
			return nil, nil, errors.New("API error")
		},
	}

	client := &mockGitLabClient{
		issuesService: issuesService,
	}

	service := &Service{
		client:               client,
		rateLimitMetadataAPI: rate.NewLimiter(rate.Inf, 1),
	}

	tmpFile, err := os.CreateTemp("", "issues-error-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	ctx := context.Background()
	err = service.ExportIssues(ctx, 123, tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list issues")
}

func TestWriteJSONFile_Success(t *testing.T) {
	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	tmpFile, err := os.CreateTemp("", "json-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = writeJSONFile(tmpFile.Name(), data)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var result map[string]string
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	assert.Equal(t, data, result)
}

func TestWriteJSONFile_InvalidPath(t *testing.T) {
	data := map[string]string{"key": "value"}

	err := writeJSONFile("/invalid/path/file.json", data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write JSON file")
}
