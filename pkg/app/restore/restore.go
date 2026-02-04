package restore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/sgaunet/gitlab-backup/pkg/storage"
)

// Storage interface defines the storage operations needed for restore.
type Storage interface {
	// Get retrieves a file from storage and returns the local path.
	Get(ctx context.Context, key string) (string, error)
}

// Orchestrator coordinates the restore workflow across all phases.
type Orchestrator struct {
	gitlabClient *gitlab.Service
	storage      Storage
	progress     ProgressReporter
}

// NewOrchestrator creates a new restore orchestrator.
func NewOrchestrator(gitlabClient *gitlab.Service, storage Storage, cfg *config.Config) *Orchestrator {
	// Create logger
	var logger *slog.Logger
	if cfg.NoLogTime {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	return &Orchestrator{
		gitlabClient: gitlabClient,
		storage:      storage,
		progress:     NewConsoleProgressReporter(logger),
	}
}

// Restore executes the complete 7-phase restore workflow.
// It orchestrates validation, download, extraction, import, and metadata restoration.
//
// Returns RestoreResult with success status, metrics, and any errors encountered.
// Fatal errors stop the workflow; non-fatal errors are collected but allow continuation.
func (o *Orchestrator) Restore(ctx context.Context, cfg *config.Config) (*RestoreResult, error) {
	startTime := time.Now()
	result := &RestoreResult{
		Success: false,
		Metrics: RestoreMetrics{},
		Errors:  []RestoreError{},
	}

	// Determine local archive path (may need download from S3)
	localArchivePath := cfg.RestoreSource
	var tempDownloadPath string

	// Phase 1: Validation (skip if --overwrite flag set)
	if !cfg.RestoreOverwrite {
		o.progress.StartPhase(PhaseValidation)

		// Get or create project to obtain ID
		projectFullPath := fmt.Sprintf("%s/%s", cfg.RestoreTargetNS, cfg.RestoreTargetPath)
		project, _, err := o.gitlabClient.Client().Projects().GetProject(projectFullPath, nil)

		var projectID int64
		if err != nil {
			// Project doesn't exist - this is OK, validation passes
			o.progress.CompletePhase(PhaseValidation)
		} else {
			projectID = int64(project.ID)

			// Validate project is empty
			validator := NewValidator(
				o.gitlabClient.Client().Commits(),
				o.gitlabClient.Client().Issues(),
				o.gitlabClient.Client().Labels(),
			)
			emptiness, err := validator.ValidateProjectEmpty(ctx, projectID)
			if err != nil {
				o.progress.FailPhase(PhaseValidation, err)
				result.addError(PhaseValidation, "Validator", err.Error(), true)
				return result, err
			}

			if !emptiness.IsEmpty() {
				err := fmt.Errorf("project is not empty (commits: %d, issues: %d, labels: %d) - use --overwrite to skip validation",
					emptiness.CommitCount, emptiness.IssueCount, emptiness.LabelCount)
				o.progress.FailPhase(PhaseValidation, err)
				result.addError(PhaseValidation, "Validator", err.Error(), true)
				return result, err
			}
			o.progress.CompletePhase(PhaseValidation)
		}
	} else {
		o.progress.SkipPhase(PhaseValidation, "overwrite flag set")
	}

	// Phase 2: Download (S3 only)
	if cfg.StorageType == "s3" {
		o.progress.StartPhase(PhaseDownload)
		downloadedPath, err := o.storage.Get(ctx, cfg.RestoreSource)
		if err != nil {
			o.progress.FailPhase(PhaseDownload, err)
			result.addError(PhaseDownload, "S3Storage", err.Error(), true)
			return result, err
		}
		localArchivePath = downloadedPath
		tempDownloadPath = downloadedPath
		o.progress.CompletePhase(PhaseDownload)
	}

	// Phase 3: Extraction
	o.progress.StartPhase(PhaseExtraction)
	tempDir, err := os.MkdirTemp(cfg.TmpDir, "gitlab-restore-*")
	if err != nil {
		o.progress.FailPhase(PhaseExtraction, err)
		result.addError(PhaseExtraction, "TempDir", err.Error(), true)
		return result, err
	}
	defer func() {
		// Phase 7: Cleanup (always runs)
		if err := os.RemoveAll(tempDir); err != nil {
			result.addWarning(fmt.Sprintf("Failed to cleanup temp dir %s: %v", tempDir, err))
		}
		if tempDownloadPath != "" {
			if err := os.Remove(tempDownloadPath); err != nil {
				result.addWarning(fmt.Sprintf("Failed to cleanup downloaded archive %s: %v", tempDownloadPath, err))
			}
		}
	}()

	archiveContents, err := storage.ExtractArchive(ctx, localArchivePath, tempDir)
	if err != nil {
		o.progress.FailPhase(PhaseExtraction, err)
		result.addError(PhaseExtraction, "ArchiveExtractor", err.Error(), true)
		return result, err
	}
	o.progress.CompletePhase(PhaseExtraction)

	// Phase 4: Import
	o.progress.StartPhase(PhaseImport)
	importService := gitlab.NewImportServiceWithRateLimiters(
		o.gitlabClient.Client().ProjectImportExport(),
		o.gitlabClient.Client().Labels(),
		o.gitlabClient.Client().Issues(),
		o.gitlabClient.Client().Notes(),
		o.gitlabClient.RateLimitImportAPI(),
		o.gitlabClient.RateLimitMetadataAPI(),
	)

	archiveFile, err := os.Open(archiveContents.ProjectExportPath)
	if err != nil {
		o.progress.FailPhase(PhaseImport, err)
		result.addError(PhaseImport, "FileIO", err.Error(), true)
		return result, err
	}
	defer archiveFile.Close()

	importStatus, err := importService.ImportProject(ctx, archiveFile, cfg.RestoreTargetNS, cfg.RestoreTargetPath)
	if err != nil {
		o.progress.FailPhase(PhaseImport, err)
		result.addError(PhaseImport, "GitLabImport", err.Error(), true)
		return result, err
	}

	result.ProjectID = importStatus.ID
	result.ProjectURL = fmt.Sprintf("%s/%s/%s", cfg.GitlabURI, cfg.RestoreTargetNS, cfg.RestoreTargetPath)
	o.progress.CompletePhase(PhaseImport)

	// Phase 5: Labels (non-fatal)
	if cfg.RestoreLabels && archiveContents.HasLabels() {
		o.progress.StartPhase(PhaseLabels)
		labelsCreated, labelsSkipped, err := importService.RestoreLabels(ctx, importStatus.ID, archiveContents.LabelsJSONPath)
		if err != nil {
			o.progress.FailPhase(PhaseLabels, err)
			result.addError(PhaseLabels, "LabelRestore", err.Error(), false)
		} else {
			result.Metrics.LabelsRestored = labelsCreated
			result.Metrics.LabelsSkipped = labelsSkipped
			o.progress.CompletePhase(PhaseLabels)
		}
	} else if !cfg.RestoreLabels {
		o.progress.SkipPhase(PhaseLabels, "disabled via flag")
	} else {
		o.progress.SkipPhase(PhaseLabels, "no labels.json in archive")
	}

	// Phase 6: Issues (non-fatal)
	if cfg.RestoreIssues && archiveContents.HasIssues() {
		o.progress.StartPhase(PhaseIssues)
		issuesCreated, notesCreated, err := importService.RestoreIssues(ctx, importStatus.ID, archiveContents.IssuesJSONPath, cfg.RestoreWithSudo)
		if err != nil {
			o.progress.FailPhase(PhaseIssues, err)
			result.addError(PhaseIssues, "IssueRestore", err.Error(), false)
		} else {
			result.Metrics.IssuesRestored = issuesCreated
			result.Metrics.NotesRestored = notesCreated
			o.progress.CompletePhase(PhaseIssues)
		}
	} else if !cfg.RestoreIssues {
		o.progress.SkipPhase(PhaseIssues, "disabled via flag")
	} else {
		o.progress.SkipPhase(PhaseIssues, "no issues.json in archive")
	}

	// Calculate final metrics
	result.Metrics.DurationSeconds = int64(time.Since(startTime).Seconds())
	result.Success = !result.hasFatalErrors()

	return result, nil
}

// addError adds an error to the result.
func (r *RestoreResult) addError(phase RestorePhase, component string, message string, fatal bool) {
	r.Errors = append(r.Errors, RestoreError{
		Phase:     phase,
		Component: component,
		Message:   message,
		Fatal:     fatal,
		Timestamp: time.Now(),
	})
}

// addWarning adds a warning to the result.
func (r *RestoreResult) addWarning(warning string) {
	r.Warnings = append(r.Warnings, warning)
}

// hasFatalErrors returns true if any errors are fatal.
func (r *RestoreResult) hasFatalErrors() bool {
	for _, err := range r.Errors {
		if err.Fatal {
			return true
		}
	}
	return false
}

// ErrProjectNotEmpty is returned when trying to restore to a non-empty project.
var ErrProjectNotEmpty = errors.New("target project is not empty")
