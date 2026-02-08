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
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
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

// Restore executes the complete 5-phase restore workflow.
// It orchestrates validation, download, extraction, and import.
//
// Returns Result with success status, metrics, and any errors encountered.
// Fatal errors stop the workflow; non-fatal errors are collected but allow continuation.
//
//nolint:funlen // Orchestration function complexity is acceptable
func (o *Orchestrator) Restore(ctx context.Context, cfg *config.Config) (*Result, error) {
	startTime := time.Now()
	result := &Result{
		Success: false,
		Metrics: Metrics{},
		Errors:  []Error{},
	}

	// Determine local archive path (may need download from S3)
	localArchivePath := cfg.RestoreSource
	var tempDownloadPath string

	// Phase 1: Validation (skip if --overwrite flag set)
	if err := o.validateProject(ctx, cfg, result); err != nil {
		return result, err
	}

	// Phase 2: Download (S3 only)
	if cfg.StorageType == "s3" {
		downloadedPath, err := o.downloadFromS3(ctx, cfg, result)
		if err != nil {
			return result, err
		}
		localArchivePath = downloadedPath
		tempDownloadPath = downloadedPath
	}

	// Phase 3: Extraction
	o.progress.StartPhase(PhaseExtraction)
	tempDir, err := os.MkdirTemp(cfg.TmpDir, "gitlab-restore-*")
	if err != nil {
		o.progress.FailPhase(PhaseExtraction, err)
		result.addError(PhaseExtraction, "TempDir", err.Error())
		return result, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer o.cleanup(result, tempDir, tempDownloadPath)

	archiveContents, err := storage.ExtractArchive(ctx, localArchivePath, tempDir)
	if err != nil {
		o.progress.FailPhase(PhaseExtraction, err)
		result.addError(PhaseExtraction, "ArchiveExtractor", err.Error())
		return result, fmt.Errorf("archive extraction failed: %w", err)
	}
	o.progress.CompletePhase(PhaseExtraction)

	// Phase 4: Import
	o.progress.StartPhase(PhaseImport)
	importService := gitlab.NewImportServiceWithRateLimiters(
		o.gitlabClient.Client().ProjectImportExport(),
		o.gitlabClient.RateLimitImportAPI(),
	)

	archiveFile, err := os.Open(archiveContents.ProjectExportPath)
	if err != nil {
		o.progress.FailPhase(PhaseImport, err)
		result.addError(PhaseImport, "FileIO", err.Error())
		return result, fmt.Errorf("failed to open archive file: %w", err)
	}
	defer func() {
		_ = archiveFile.Close()
	}()

	importStatus, err := importService.ImportProject(ctx, archiveFile, cfg.RestoreTargetNS, cfg.RestoreTargetPath)
	if err != nil {
		o.progress.FailPhase(PhaseImport, err)
		result.addError(PhaseImport, "GitLabImport", err.Error())
		return result, fmt.Errorf("import failed: %w", err)
	}

	result.ProjectID = importStatus.ID
	result.ProjectURL = fmt.Sprintf("%s/%s/%s", cfg.GitlabURI, cfg.RestoreTargetNS, cfg.RestoreTargetPath)
	o.progress.CompletePhase(PhaseImport)

	// Phase 5: Cleanup (moved from phase 7, now runs in defer at top of function)
	// Calculate final metrics
	result.Metrics.DurationSeconds = int64(time.Since(startTime).Seconds())
	result.Success = !result.hasFatalErrors()

	return result, nil
}

// addError adds a fatal error to the result.
func (r *Result) addError(phase Phase, component string, message string) {
	r.Errors = append(r.Errors, Error{
		Phase:     phase,
		Component: component,
		Message:   message,
		Fatal:     true,
		Timestamp: time.Now(),
	})
}

// addWarning adds a warning to the result.
func (r *Result) addWarning(warning string) {
	r.Warnings = append(r.Warnings, warning)
}

// hasFatalErrors returns true if any errors are fatal.
func (r *Result) hasFatalErrors() bool {
	for _, err := range r.Errors {
		if err.Fatal {
			return true
		}
	}
	return false
}

// ErrProjectNotEmpty is returned when trying to restore to a non-empty project.
var ErrProjectNotEmpty = errors.New("target project is not empty")

// ErrProjectHasContent is returned when project is not empty (has commits, issues, or labels).
var ErrProjectHasContent = errors.New("project is not empty - use --overwrite to skip validation")

// validateProject validates that the target project is empty (if not overwriting).
func (o *Orchestrator) validateProject(ctx context.Context, cfg *config.Config, result *Result) error {
	if !cfg.RestoreOverwrite {
		o.progress.StartPhase(PhaseValidation)

		// Get or create project to obtain ID
		projectFullPath := fmt.Sprintf("%s/%s", cfg.RestoreTargetNS, cfg.RestoreTargetPath)
		project, _, err := o.gitlabClient.Client().Projects().GetProject(projectFullPath, nil)

		//nolint:nilerr // Project not existing is expected and not an error
		if err != nil {
			// Project doesn't exist - this is OK, validation passes
			o.progress.CompletePhase(PhaseValidation)
			return nil
		}

		projectID := project.ID

		// Validate project is empty
		validator := NewValidator(
			o.gitlabClient.Client().Commits(),
			o.gitlabClient.Client().Issues(),
			o.gitlabClient.Client().Labels(),
		)
		emptiness, err := validator.ValidateProjectEmpty(ctx, projectID)
		if err != nil {
			o.progress.FailPhase(PhaseValidation, err)
			result.addError(PhaseValidation, "Validator", err.Error())
			return err
		}

		if !emptiness.IsEmpty() {
			err := fmt.Errorf("%w (commits: %d, issues: %d, labels: %d)",
				ErrProjectHasContent, emptiness.CommitCount, emptiness.IssueCount, emptiness.LabelCount)
			o.progress.FailPhase(PhaseValidation, err)
			result.addError(PhaseValidation, "Validator", err.Error())
			return err
		}
		o.progress.CompletePhase(PhaseValidation)
	} else {
		o.progress.SkipPhase(PhaseValidation, "overwrite flag set")
	}
	return nil
}

// downloadFromS3 downloads the archive from S3 storage.
func (o *Orchestrator) downloadFromS3(ctx context.Context, cfg *config.Config, result *Result) (string, error) {
	o.progress.StartPhase(PhaseDownload)
	downloadedPath, err := o.storage.Get(ctx, cfg.RestoreSource)
	if err != nil {
		o.progress.FailPhase(PhaseDownload, err)
		result.addError(PhaseDownload, "S3Storage", err.Error())
		return "", fmt.Errorf("download failed: %w", err)
	}
	o.progress.CompletePhase(PhaseDownload)
	return downloadedPath, nil
}

// cleanup removes temporary files and directories.
func (o *Orchestrator) cleanup(result *Result, tempDir, tempDownloadPath string) {
	if err := os.RemoveAll(tempDir); err != nil {
		result.addWarning(fmt.Sprintf("Failed to cleanup temp dir %s: %v", tempDir, err))
	}
	if tempDownloadPath != "" {
		if err := os.Remove(tempDownloadPath); err != nil {
			result.addWarning(fmt.Sprintf("Failed to cleanup downloaded archive %s: %v", tempDownloadPath, err))
		}
	}
}
