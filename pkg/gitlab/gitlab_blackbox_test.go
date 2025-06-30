package gitlab_test

import (
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
)

func TestSetLogger(t *testing.T) {
	// Test setting a nil logger (should handle gracefully)
	gitlab.SetLogger(nil)
}
