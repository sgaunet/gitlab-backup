package restore

import (
	"time"
)

// Phase represents the current phase of the restore operation.
type Phase string

const (
	// PhaseValidation validates configuration and target project emptiness.
	PhaseValidation Phase = "validation"
	// PhaseDownload downloads archive from S3 (if applicable).
	PhaseDownload Phase = "download"
	// PhaseExtraction extracts archive contents to temporary directory.
	PhaseExtraction Phase = "extraction"
	// PhaseImport imports the GitLab project repository.
	PhaseImport Phase = "import"
	// PhaseCleanup removes temporary files.
	PhaseCleanup Phase = "cleanup"
	// PhaseComplete indicates successful completion.
	PhaseComplete Phase = "complete"
)

// Result represents the final outcome of a restore operation.
type Result struct {
	// Success indicates whether the restore completed successfully.
	Success bool
	// ProjectID is the ID of the restored project.
	ProjectID int64
	// ProjectURL is the web URL of the restored project.
	ProjectURL string
	// Metrics contains quantitative restore metrics.
	Metrics Metrics
	// Errors contains all errors encountered during restore.
	Errors []Error
	// Warnings contains non-fatal warnings.
	Warnings []string
}

// Metrics tracks quantitative restore operation metrics.
type Metrics struct {
	// BytesDownloaded is the bytes downloaded from S3 (if applicable).
	BytesDownloaded int64
	// BytesExtracted is the bytes extracted from archive.
	BytesExtracted int64
	// DurationSeconds is the total restore duration in seconds.
	DurationSeconds int64
}

// Error represents an error that occurred during restore.
type Error struct {
	// Phase indicates which phase the error occurred in.
	Phase Phase
	// Component identifies the component that failed (e.g., "import", "labels", "issues").
	Component string
	// Message is the error message.
	Message string
	// Fatal indicates whether the error is fatal (stops restore).
	Fatal bool
	// Timestamp is when the error occurred.
	Timestamp time.Time
}

// EmptinessChecks tracks the three-part emptiness validation.
type EmptinessChecks struct {
	// HasCommits indicates whether project has any commits.
	HasCommits bool
	// HasIssues indicates whether project has any issues.
	HasIssues bool
	// HasLabels indicates whether project has any labels.
	HasLabels bool
	// CommitCount is the number of commits found.
	CommitCount int
	// IssueCount is the number of issues found.
	IssueCount int
	// LabelCount is the number of labels found.
	LabelCount int
}

// IsEmpty returns true if the project has no commits, issues, or labels.
func (e *EmptinessChecks) IsEmpty() bool {
	return !e.HasCommits && !e.HasIssues && !e.HasLabels
}

// Progress tracks restore operation progress.
type Progress struct {
	// CurrentPhase is the active phase.
	CurrentPhase Phase
	// CompletedPhases contains successfully completed phases.
	CompletedPhases []Phase
	// StartTime is the restore start timestamp.
	StartTime time.Time
	// PhaseStartTime is the current phase start timestamp.
	PhaseStartTime time.Time
	// Metrics contains current progress metrics.
	Metrics Metrics
}
