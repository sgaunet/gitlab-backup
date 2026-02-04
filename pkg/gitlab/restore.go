package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/time/rate"
	gitlabapi "gitlab.com/gitlab-org/api/client-go"
)

// ImportService provides GitLab project import functionality.
type ImportService struct {
	importExportService ProjectImportExportService
	labelsService       LabelsService
	issuesService       IssuesService
	notesService        NotesService
	rateLimiterImport   *rate.Limiter
	rateLimiterMetadata *rate.Limiter
}

// NewImportService creates a new import service instance.
func NewImportService(importExportService ProjectImportExportService) *ImportService {
	return &ImportService{
		importExportService: importExportService,
		rateLimiterImport: rate.NewLimiter(
			rate.Every(ImportRateLimitIntervalSeconds*time.Second),
			ImportRateLimitBurst,
		),
	}
}

// NewImportServiceWithRateLimiters creates an import service with custom rate limiters.
func NewImportServiceWithRateLimiters(
	importExportService ProjectImportExportService,
	labelsService LabelsService,
	issuesService IssuesService,
	notesService NotesService,
	rateLimiterImport *rate.Limiter,
	rateLimiterMetadata *rate.Limiter,
) *ImportService {
	return &ImportService{
		importExportService: importExportService,
		labelsService:       labelsService,
		issuesService:       issuesService,
		notesService:        notesService,
		rateLimiterImport:   rateLimiterImport,
		rateLimiterMetadata: rateLimiterMetadata,
	}
}

// ImportProject initiates a GitLab project import and waits for completion.
// It respects rate limiting and polls the import status until finished or failed.
//
// Returns the final ImportStatus on success.
// Returns error if import initiation fails, import status becomes "failed", or timeout occurs.
func (s *ImportService) ImportProject(ctx context.Context, archive io.Reader, namespace string, projectPath string) (*gitlabapi.ImportStatus, error) {
	// Wait for rate limit
	if err := s.rateLimiterImport.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Initiate import
	importStatus, _, err := s.importExportService.ImportFromFile(
		archive,
		&gitlabapi.ImportFileOptions{
			Namespace: &namespace,
			Path:      &projectPath,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate import: %w", err)
	}

	// Wait for import to complete (10 minute default timeout)
	finalStatus, err := s.WaitForImport(ctx, importStatus.ID, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("import did not complete successfully: %w", err)
	}

	return finalStatus, nil
}

// WaitForImport polls the import status until it reaches a terminal state (finished or failed).
// It respects the context deadline and rate limiting.
//
// Returns the final ImportStatus when import reaches "finished" state.
// Returns error if import fails, times out, or API errors occur.
func (s *ImportService) WaitForImport(ctx context.Context, projectID int64, timeout time.Duration) (*gitlabapi.ImportStatus, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		// Wait for rate limit
		if err := s.rateLimiterImport.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}

		// Check import status
		status, _, err := s.importExportService.ImportStatus(projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get import status: %w", err)
		}

		// Check terminal states
		switch status.ImportStatus {
		case "finished":
			return status, nil
		case "failed":
			return nil, fmt.Errorf("import failed: %s", status.ImportError)
		case "scheduled", "started":
			// Continue polling
		default:
			return nil, fmt.Errorf("unexpected import status: %s", status.ImportStatus)
		}

		// Wait for next poll or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}

// LabelData represents a label from labels.json.
type LabelData struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	Priority    int64  `json:"priority"`
}

// RestoreLabels restores project labels from a labels.json file.
// It reads the JSON file, creates each label via GitLab API, and handles duplicates gracefully.
//
// Returns (labelsCreated, labelsSkipped, error).
// Duplicate labels (409 Conflict) are counted as skipped, not errors.
// Other errors are logged but don't fail the entire operation (non-fatal).
func (s *ImportService) RestoreLabels(ctx context.Context, projectID int64, labelsJSONPath string) (int, int, error) {
	// Read labels JSON file
	data, err := os.ReadFile(labelsJSONPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read labels file: %w", err)
	}

	// Parse labels
	var labels []LabelData
	if err := json.Unmarshal(data, &labels); err != nil {
		return 0, 0, fmt.Errorf("failed to parse labels JSON: %w", err)
	}

	labelsCreated := 0
	labelsSkipped := 0

	// Create each label
	for _, labelData := range labels {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return labelsCreated, labelsSkipped, ctx.Err()
		default:
		}

		// Wait for rate limit
		if err := s.rateLimiterMetadata.Wait(ctx); err != nil {
			return labelsCreated, labelsSkipped, fmt.Errorf("rate limit wait failed: %w", err)
		}

		// Create label
		_, resp, err := s.labelsService.CreateLabel(
			projectID,
			&gitlabapi.CreateLabelOptions{
				Name:        &labelData.Name,
				Color:       &labelData.Color,
				Description: &labelData.Description,
				Priority:    &labelData.Priority,
			},
		)

		if err != nil {
			// Check for duplicate label (409 Conflict)
			if resp != nil && resp.StatusCode == 409 {
				labelsSkipped++
				log.Debug("Label already exists, skipping", "label", labelData.Name)
				continue
			}
			// Log other errors but continue
			log.Warn("Failed to create label", "label", labelData.Name, "error", err)
			continue
		}

		labelsCreated++
	}

	return labelsCreated, labelsSkipped, nil
}

// IssueData represents an issue from issues.json.
type IssueData struct {
	ID          int64       `json:"id"`
	IID         int64       `json:"iid"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	State       string      `json:"state"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	ClosedAt    *string     `json:"closed_at"`
	Labels      []string    `json:"labels"`
	Author      AuthorData  `json:"author"`
	Assignees   []UserData  `json:"assignees"`
	Weight      *int64      `json:"weight"`
	Notes       []NoteData  `json:"notes"`
}

// AuthorData represents an issue/note author.
type AuthorData struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// UserData represents a user (assignee, etc.).
type UserData struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// NoteData represents an issue comment.
type NoteData struct {
	ID        int64      `json:"id"`
	Body      string     `json:"body"`
	Author    AuthorData `json:"author"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

// RestoreIssues restores project issues and notes from an issues.json file.
// It creates issues with preserved timestamps, sets states, and creates notes.
//
// If withSudo is true, it uses sudo to impersonate issue/note authors.
// Returns (issuesCreated, notesCreated, error).
// Errors are logged but don't fail the entire operation (non-fatal).
func (s *ImportService) RestoreIssues(ctx context.Context, projectID int64, issuesJSONPath string, withSudo bool) (int, int, error) {
	// Read issues JSON file
	data, err := os.ReadFile(issuesJSONPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read issues file: %w", err)
	}

	// Parse issues
	var issues []IssueData
	if err := json.Unmarshal(data, &issues); err != nil {
		return 0, 0, fmt.Errorf("failed to parse issues JSON: %w", err)
	}

	issuesCreated := 0
	notesCreated := 0

	// Create each issue
	for _, issueData := range issues {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return issuesCreated, notesCreated, ctx.Err()
		default:
		}

		// Create issue with preserved created_at
		createdIssue, issueErr := s.createIssueWithMetadata(ctx, projectID, issueData, withSudo)
		if issueErr != nil {
			log.Warn("Failed to create issue", "title", issueData.Title, "error", issueErr)
			continue
		}
		issuesCreated++

		// If issue should be closed, update state
		if issueData.State == "closed" {
			if err := s.closeIssue(ctx, projectID, createdIssue.IID); err != nil {
				log.Warn("Failed to close issue", "iid", createdIssue.IID, "error", err)
			}
		}

		// Restore notes for this issue
		noteCount, err := s.restoreIssueNotes(ctx, projectID, createdIssue.IID, issueData.Notes, withSudo)
		if err != nil {
			log.Warn("Failed to restore some notes", "iid", createdIssue.IID, "error", err)
		}
		notesCreated += noteCount
	}

	return issuesCreated, notesCreated, nil
}

// createIssueWithMetadata creates an issue with preserved metadata.
func (s *ImportService) createIssueWithMetadata(ctx context.Context, projectID int64, issueData IssueData, withSudo bool) (*gitlabapi.Issue, error) {
	// Wait for rate limit
	if err := s.rateLimiterMetadata.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Parse created_at timestamp
	createdAt, err := time.Parse(time.RFC3339, issueData.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	// Build assignee IDs
	var assigneeIDs []int64
	for _, assignee := range issueData.Assignees {
		assigneeIDs = append(assigneeIDs, assignee.ID)
	}

	// Build label names (GitLab API uses label names, not IDs)
	labelPtr := gitlabapi.LabelOptions(issueData.Labels)

	// Create issue options
	opts := &gitlabapi.CreateIssueOptions{
		Title:       &issueData.Title,
		Description: &issueData.Description,
		Labels:      &labelPtr,
		AssigneeIDs: &assigneeIDs,
		CreatedAt:   &createdAt,
	}

	if issueData.Weight != nil {
		opts.Weight = issueData.Weight
	}

	// Use sudo if enabled
	var requestOpts []gitlabapi.RequestOptionFunc
	if withSudo {
		requestOpts = append(requestOpts, gitlabapi.WithSudo(int(issueData.Author.ID)))
	}

	// Create issue
	issue, _, err := s.issuesService.CreateIssue(projectID, opts, requestOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	return issue, nil
}

// closeIssue closes an issue by updating its state.
func (s *ImportService) closeIssue(ctx context.Context, projectID int64, issueIID int64) error {
	// Wait for rate limit
	if err := s.rateLimiterMetadata.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait failed: %w", err)
	}

	stateEvent := "close"
	_, _, err := s.issuesService.UpdateIssue(
		projectID,
		issueIID,
		&gitlabapi.UpdateIssueOptions{
			StateEvent: &stateEvent,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	return nil
}

// restoreIssueNotes creates notes for an issue.
func (s *ImportService) restoreIssueNotes(ctx context.Context, projectID int64, issueIID int64, notes []NoteData, withSudo bool) (int, error) {
	notesCreated := 0

	for _, noteData := range notes {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return notesCreated, ctx.Err()
		default:
		}

		// Wait for rate limit
		if err := s.rateLimiterMetadata.Wait(ctx); err != nil {
			return notesCreated, fmt.Errorf("rate limit wait failed: %w", err)
		}

		// Parse created_at timestamp
		createdAt, err := time.Parse(time.RFC3339, noteData.CreatedAt)
		if err != nil {
			log.Warn("Failed to parse note created_at", "error", err)
			continue
		}

		// Build request options
		var requestOpts []gitlabapi.RequestOptionFunc
		if withSudo {
			requestOpts = append(requestOpts, gitlabapi.WithSudo(int(noteData.Author.ID)))
		}

		// Create note
		_, _, err = s.notesService.CreateIssueNote(
			projectID,
			issueIID,
			&gitlabapi.CreateIssueNoteOptions{
				Body:      &noteData.Body,
				CreatedAt: &createdAt,
			},
			requestOpts...,
		)
		if err != nil {
			log.Warn("Failed to create note", "error", err)
			continue
		}

		notesCreated++
	}

	return notesCreated, nil
}
