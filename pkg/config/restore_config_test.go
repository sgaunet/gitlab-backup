package config_test

import (
	"path/filepath"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/stretchr/testify/require"
)

// baseValidRestoreConfig returns a RestoreConfig whose embedded Config passes
// the base backup Validate() (which ValidateRestore calls first). TmpDir points
// at a real writable directory so validateTmpDir succeeds. Restore-specific
// fields are left to the caller.
func baseValidRestoreConfig(t *testing.T) *config.RestoreConfig {
	t.Helper()
	dir := t.TempDir()
	return &config.RestoreConfig{
		Config: config.Config{
			GitlabGroupID:     123,
			GitlabToken:       "test-token",
			GitlabURI:         "https://gitlab.com",
			LocalPath:         dir,
			TmpDir:            dir,
			ExportTimeoutMins: 10,
			ImportTimeoutMins: 60,
		},
	}
}

func TestRestoreConfig_IsS3Source(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{"s3 source", "s3://bucket/key", true},
		{"local tar.gz", "/data/archive.tar.gz", false},
		{"empty", "", false},
		{"s3 prefix only", "s3://", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &config.RestoreConfig{RestoreSource: tc.source}
			require.Equal(t, tc.want, c.IsS3Source())
		})
	}
}

func TestRestoreConfig_ParseS3Source(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c := &config.RestoreConfig{RestoreSource: "s3://my-bucket/path/to/archive.tar.gz"}
		bucket, key, err := c.ParseS3Source()
		require.NoError(t, err)
		require.Equal(t, "my-bucket", bucket)
		require.Equal(t, "path/to/archive.tar.gz", key)
	})

	t.Run("not an s3 source", func(t *testing.T) {
		c := &config.RestoreConfig{RestoreSource: "/local/archive.tar.gz"}
		_, _, err := c.ParseS3Source()
		require.ErrorIs(t, err, config.ErrNotS3Source)
	})

	t.Run("missing key separator", func(t *testing.T) {
		c := &config.RestoreConfig{RestoreSource: "s3://bucketonly"}
		_, _, err := c.ParseS3Source()
		require.ErrorIs(t, err, config.ErrInvalidS3PathFormat)
	})

	t.Run("empty bucket", func(t *testing.T) {
		c := &config.RestoreConfig{RestoreSource: "s3:///key"}
		_, _, err := c.ParseS3Source()
		require.ErrorIs(t, err, config.ErrInvalidS3Path)
	})

	t.Run("empty key", func(t *testing.T) {
		c := &config.RestoreConfig{RestoreSource: "s3://bucket/"}
		_, _, err := c.ParseS3Source()
		require.ErrorIs(t, err, config.ErrInvalidS3Path)
	})
}

func TestRestoreConfig_GetFullProjectPath(t *testing.T) {
	t.Run("no namespace", func(t *testing.T) {
		c := &config.RestoreConfig{RestoreTargetPath: "myproject"}
		require.Equal(t, "myproject", c.GetFullProjectPath())
	})

	t.Run("with namespace", func(t *testing.T) {
		c := &config.RestoreConfig{RestoreTargetNS: "group/sub", RestoreTargetPath: "myproject"}
		require.Equal(t, filepath.Join("group/sub", "myproject"), c.GetFullProjectPath())
	})
}

func TestRestoreConfig_ValidateRestore_Source(t *testing.T) {
	t.Run("empty source", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		c.RestoreTargetPath = "proj"
		c.RestoreSource = ""
		err := c.ValidateRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "restoreSource is required")
	})

	t.Run("s3 source without valid s3 config", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		c.RestoreTargetPath = "proj"
		c.RestoreSource = "s3://bucket/archive.tar.gz"
		// No S3cfg set -> IsS3ConfigValid() is false.
		err := c.ValidateRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "S3 configuration required")
	})

	t.Run("local source not tar.gz", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		c.RestoreTargetPath = "proj"
		c.RestoreSource = "/data/archive.zip"
		err := c.ValidateRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), ".tar.gz file")
	})

	t.Run("local source with path traversal", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		c.RestoreTargetPath = "proj"
		c.RestoreSource = "../../etc/archive.tar.gz"
		err := c.ValidateRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "traversal")
	})

	t.Run("valid local source", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		c.RestoreTargetPath = "proj"
		c.RestoreSource = filepath.Join(c.LocalPath, "archive.tar.gz")
		require.NoError(t, c.ValidateRestore())
	})

	t.Run("valid s3 source", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		c.RestoreTargetPath = "proj"
		c.RestoreSource = "s3://my-backup-bucket/archive.tar.gz"
		c.S3cfg = config.S3Config{
			BucketName: "my-backup-bucket",
			BucketPath: "backups",
			Region:     "us-east-1",
		}
		require.NoError(t, c.ValidateRestore())
	})
}

func TestRestoreConfig_ValidateRestore_Target(t *testing.T) {
	validSource := func(c *config.RestoreConfig) {
		c.RestoreSource = filepath.Join(c.LocalPath, "archive.tar.gz")
	}

	t.Run("empty target path", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		validSource(c)
		c.RestoreTargetPath = ""
		err := c.ValidateRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "restoreTargetPath is required")
	})

	t.Run("invalid target path characters", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		validSource(c)
		c.RestoreTargetPath = "invalid path"
		err := c.ValidateRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid restoreTargetPath")
	})

	t.Run("valid nested namespace", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		validSource(c)
		c.RestoreTargetPath = "proj"
		c.RestoreTargetNS = "group/subgroup"
		require.NoError(t, c.ValidateRestore())
	})

	t.Run("namespace with empty segment", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		validSource(c)
		c.RestoreTargetPath = "proj"
		c.RestoreTargetNS = "group//sub"
		err := c.ValidateRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty path segments")
	})

	t.Run("namespace with invalid segment", func(t *testing.T) {
		c := baseValidRestoreConfig(t)
		validSource(c)
		c.RestoreTargetPath = "proj"
		c.RestoreTargetNS = "group/bad segment"
		err := c.ValidateRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid restoreTargetNS")
	})
}

func TestConfig_ValidateForRestore(t *testing.T) {
	baseValid := func(t *testing.T) *config.Config {
		t.Helper()
		dir := t.TempDir()
		return &config.Config{
			GitlabToken:       "test-token",
			GitlabURI:         "https://gitlab.com",
			TmpDir:            dir,
			ExportTimeoutMins: 10,
			ImportTimeoutMins: 60,
		}
	}

	t.Run("happy path does not require group or project id", func(t *testing.T) {
		c := baseValid(t)
		require.NoError(t, c.ValidateForRestore())
	})

	t.Run("missing token", func(t *testing.T) {
		c := baseValid(t)
		c.GitlabToken = ""
		err := c.ValidateForRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "gitlabToken is required")
	})

	t.Run("invalid gitlab uri scheme", func(t *testing.T) {
		c := baseValid(t)
		c.GitlabURI = "ftp://gitlab.example.com"
		err := c.ValidateForRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "http or https scheme")
	})

	t.Run("tmpdir does not exist", func(t *testing.T) {
		c := baseValid(t)
		c.TmpDir = filepath.Join(t.TempDir(), "does-not-exist")
		err := c.ValidateForRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "tmpdir")
	})

	t.Run("timeout too low", func(t *testing.T) {
		c := baseValid(t)
		c.ExportTimeoutMins = 0
		err := c.ValidateForRestore()
		require.Error(t, err)
		require.Contains(t, err.Error(), "exportTimeoutMins")
	})
}
