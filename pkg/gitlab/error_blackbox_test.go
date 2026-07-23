package gitlab_test

import (
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalErrorMessage(t *testing.T) {
	t.Run("valid api error message", func(t *testing.T) {
		err := gitlab.UnmarshalErrorMessage([]byte(`{"message":"boom"}`))
		require.Error(t, err)
		require.ErrorIs(t, err, gitlab.ErrGitlabAPI)
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("invalid json", func(t *testing.T) {
		err := gitlab.UnmarshalErrorMessage([]byte(`not-json`))
		require.Error(t, err)
		require.ErrorIs(t, err, gitlab.ErrUnmarshalJSON)
	})
}
