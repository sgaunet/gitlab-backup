package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const (
	// metadataAPIPageSize defines the page size for metadata API pagination.
	metadataAPIPageSize = 100
)

// ExportLabels fetches all labels for a project and writes them to a JSON file.
func (s *Service) ExportLabels(ctx context.Context, projectID int64, outputPath string) error {
	err := s.rateLimitMetadataAPI.Wait(ctx) // Rate limiting for metadata API
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRateLimit, err)
	}

	var allLabels []*gitlab.Label
	opt := &gitlab.ListLabelsOptions{
		ListOptions: gitlab.ListOptions{PerPage: metadataAPIPageSize},
	}

	for {
		labels, resp, err := s.client.Labels().ListLabels(projectID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("failed to list labels: %w", err)
		}

		allLabels = append(allLabels, labels...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	log.Debug("ExportLabels", "projectID", projectID, "count", len(allLabels))

	return writeJSONFile(outputPath, allLabels)
}

// ExportIssues fetches all issues for a project (with full history) and writes them to a JSON file.
func (s *Service) ExportIssues(ctx context.Context, projectID int64, outputPath string) error {
	err := s.rateLimitMetadataAPI.Wait(ctx) // Rate limiting for metadata API
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRateLimit, err)
	}

	var allIssues []*gitlab.Issue
	opt := &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{PerPage: metadataAPIPageSize},
	}

	for {
		issues, resp, err := s.client.Issues().ListProjectIssues(projectID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("failed to list issues: %w", err)
		}

		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	log.Debug("ExportIssues", "projectID", projectID, "count", len(allIssues))

	return writeJSONFile(outputPath, allIssues)
}

// writeJSONFile marshals the given data to JSON and writes it to the specified file path.
func writeJSONFile(filePath string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	//nolint:gosec,mnd // G304: File creation is intentional for metadata export
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", filePath, err)
	}

	return nil
}
