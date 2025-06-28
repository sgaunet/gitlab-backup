package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// get sends a GET request to the Gitlab API to the given path.
func (r *Service) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	req.Header.Set("Private-Token", r.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GET request: %w", err)
	}
	return resp, nil
}

// post sends a POST request to the Gitlab API to the given path.
func (r *Service) post(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Private-Token", r.token)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute POST request: %w", err)
	}
	return resp, nil
}

// makeRequest makes a GET request and handles common response processing.
func (r *Service) makeRequest(ctx context.Context, url, resourceType string) ([]byte, error) {
	resp, err := r.get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("error retrieving %s: %w", resourceType, err)
	}
	defer func() { _ = resp.Body.Close() }()
	
	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errMsg ErrorMessage
		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &errMsg); err != nil {
			// If we can't unmarshal the error message, return a generic error
			return nil, fmt.Errorf("error retrieving %s: %w: %d", resourceType, ErrHTTPStatusCode, resp.StatusCode)
		}
		return nil, fmt.Errorf("error retrieving %s: %w: %s", resourceType, ErrGitlabAPI, errMsg.Message)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}