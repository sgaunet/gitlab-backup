package app

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupSummary_RecordAndCounts(t *testing.T) {
	s := newBackupSummary()

	s.recordSuccess("alpha", 2*time.Second)
	s.recordSkipped("beta")
	s.recordFailure("gamma", errors.New("timeout"), 5*time.Second)
	s.recordSuccess("delta", 1*time.Second)

	total, succeeded, skipped, failed := s.counts()
	assert.Equal(t, 4, total)
	assert.Equal(t, 2, succeeded)
	assert.Equal(t, 1, skipped)
	assert.Equal(t, 1, failed)
}

func TestBackupSummary_HasFailures(t *testing.T) {
	t.Run("no failures", func(t *testing.T) {
		s := newBackupSummary()
		s.recordSuccess("a", time.Second)
		s.recordSkipped("b")
		assert.False(t, s.hasFailures())
	})

	t.Run("with failure", func(t *testing.T) {
		s := newBackupSummary()
		s.recordSuccess("a", time.Second)
		s.recordFailure("b", errors.New("err"), time.Second)
		assert.True(t, s.hasFailures())
	})

	t.Run("empty summary", func(t *testing.T) {
		s := newBackupSummary()
		assert.False(t, s.hasFailures())
	})
}

func TestBackupSummary_ConcurrentAccess(t *testing.T) {
	s := newBackupSummary()
	var wg sync.WaitGroup
	const n = 100

	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			switch i % 3 {
			case 0:
				s.recordSuccess("proj", time.Millisecond)
			case 1:
				s.recordSkipped("proj")
			case 2:
				s.recordFailure("proj", errors.New("err"), time.Millisecond)
			}
		}(i)
	}
	wg.Wait()

	total, succeeded, skipped, failed := s.counts()
	assert.Equal(t, n, total)
	require.Equal(t, n, succeeded+skipped+failed)
}

// noopLogger satisfies the Logger interface without producing output.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}

func TestBackupSummary_PrintSummary(t *testing.T) {
	s := newBackupSummary()
	s.recordSuccess("alpha", 2*time.Second)
	s.recordSkipped("beta")
	s.recordFailure("gamma", errors.New("boom"), 3*time.Second)

	// Must not panic.
	require.NotPanics(t, func() {
		s.printSummary(noopLogger{})
	})
}
