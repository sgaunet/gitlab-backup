package app

import (
	"fmt"
	"sync"
	"time"
)

// projectStatus represents the outcome of a single project backup.
type projectStatus int

const (
	statusSuccess projectStatus = iota
	statusSkipped
	statusFailed
)

// projectResult holds the outcome of a single project backup.
type projectResult struct {
	name     string
	status   projectStatus
	err      error
	duration time.Duration
}

// backupSummary collects results from concurrent project backups.
type backupSummary struct {
	mu        sync.Mutex
	results   []projectResult
	startTime time.Time
}

// newBackupSummary creates a new summary with the clock started.
func newBackupSummary() *backupSummary {
	return &backupSummary{
		startTime: time.Now(),
	}
}

func (s *backupSummary) recordSuccess(name string, d time.Duration) {
	s.mu.Lock()
	s.results = append(s.results, projectResult{name: name, status: statusSuccess, duration: d})
	s.mu.Unlock()
}

func (s *backupSummary) recordSkipped(name string) {
	s.mu.Lock()
	s.results = append(s.results, projectResult{name: name, status: statusSkipped})
	s.mu.Unlock()
}

func (s *backupSummary) recordFailure(name string, err error, d time.Duration) {
	s.mu.Lock()
	s.results = append(s.results, projectResult{name: name, status: statusFailed, err: err, duration: d})
	s.mu.Unlock()
}

func (s *backupSummary) hasFailures() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.results {
		if r.status == statusFailed {
			return true
		}
	}
	return false
}

func (s *backupSummary) counts() (int, int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total, succeeded, skipped, failed int
	for _, r := range s.results {
		total++
		switch r.status {
		case statusSuccess:
			succeeded++
		case statusSkipped:
			skipped++
		case statusFailed:
			failed++
		}
	}
	return total, succeeded, skipped, failed
}

func (s *backupSummary) printSummary(log Logger) {
	total, succeeded, skipped, failed := s.counts()
	duration := time.Since(s.startTime).Truncate(time.Second)

	const percent = 100.0
	var rate float64
	if total > 0 {
		rate = float64(succeeded) / float64(total) * percent
	}

	log.Info("[BACKUP SUMMARY] completed",
		"total", total,
		"succeeded", succeeded,
		"skipped", skipped,
		"failed", failed,
		"success_rate", fmt.Sprintf("%.1f%%", rate),
		"duration", duration.String(),
	)

	s.mu.Lock()
	results := make([]projectResult, len(s.results))
	copy(results, s.results)
	s.mu.Unlock()

	for _, r := range results {
		switch r.status {
		case statusSuccess:
			log.Info("[BACKUP SUMMARY] succeeded", "project", r.name, "duration", r.duration.Truncate(time.Second).String())
		case statusSkipped:
			log.Info("[BACKUP SUMMARY] skipped (archived)", "project", r.name)
		case statusFailed:
			log.Error("[BACKUP SUMMARY] failed",
				"project", r.name,
				"error", r.err.Error(),
				"duration", r.duration.Truncate(time.Second).String(),
			)
		}
	}
}
