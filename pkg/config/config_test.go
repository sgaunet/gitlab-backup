package config_test

import (
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromFile(t *testing.T) {
	// TestNewConfigFromFile tests the NewConfigFromFile function
	t.Run("normal case", func(t *testing.T) {
		cfg, err := config.NewConfigFromFile("testdata/good-cfg.yaml")
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, 123, cfg.GitlabGroupID)
		require.Equal(t, 456, cfg.GitlabProjectID)
		require.Equal(t, "https://gitlab.com", cfg.GitlabURI)
		// require.Equal(t, "/tmp", cfg.TmpDir)
		require.Equal(t, "/data/gitlab", cfg.LocalPath)
		require.Equal(t, "mybucket", cfg.S3cfg.BucketName)
		require.Equal(t, "myregion", cfg.S3cfg.Region)
		require.Equal(t, "myendpoint", cfg.S3cfg.Endpoint)
		require.Equal(t, "mybucketpath", cfg.S3cfg.BucketPath)
		require.Equal(t, false, cfg.NoLogTime)
		require.Equal(t, "echo prebackup", cfg.Hooks.PreBackup)
		require.Equal(t, "echo postbackup", cfg.Hooks.PostBackup)
	})
	t.Run("file not found", func(t *testing.T) {
		_, err := config.NewConfigFromFile("testdata/unknown.yaml")
		require.Error(t, err)
	})
	t.Run("invalid yaml", func(t *testing.T) {
		_, err := config.NewConfigFromFile("testdata/invalid-cfg.yaml")
		require.Error(t, err)
	})
}
func TestNewConfigFromEnv(t *testing.T) {
	t.Run("valid environment variables", func(t *testing.T) {
		t.Setenv("GITLABGROUPID", "123")
		t.Setenv("GITLABPROJECTID", "456")
		t.Setenv("GITLAB_TOKEN", "mytoken")
		t.Setenv("GITLAB_URI", "https://gitlab.example.com")
		t.Setenv("LOCALPATH", "/data/gitlab")
		t.Setenv("TMPDIR", "/tmp")
		t.Setenv("S3ENDPOINT", "myendpoint")
		t.Setenv("S3BUCKETNAME", "mybucket")
		t.Setenv("S3BUCKETPATH", "mybucketpath")
		t.Setenv("S3REGION", "myregion")
		t.Setenv("AWS_ACCESS_KEY_ID", "myaccesskey")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "mysecretkey")
		t.Setenv("NOLOGTIME", "true")

		cfg, err := config.NewConfigFromEnv()
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, 123, cfg.GitlabGroupID)
		require.Equal(t, 456, cfg.GitlabProjectID)
		require.Equal(t, "mytoken", cfg.GitlabToken)
		require.Equal(t, "https://gitlab.example.com", cfg.GitlabURI)
		require.Equal(t, "/data/gitlab", cfg.LocalPath)
		require.Equal(t, "/tmp", cfg.TmpDir)
		require.Equal(t, "myendpoint", cfg.S3cfg.Endpoint)
		require.Equal(t, "mybucket", cfg.S3cfg.BucketName)
		require.Equal(t, "mybucketpath", cfg.S3cfg.BucketPath)
		require.Equal(t, "myregion", cfg.S3cfg.Region)
		require.Equal(t, "myaccesskey", cfg.S3cfg.AccessKey)
		require.Equal(t, "mysecretkey", cfg.S3cfg.SecretKey)
		require.Equal(t, true, cfg.NoLogTime)
	})
}

func TestEmptyGitlabToken(t *testing.T) {
	t.Setenv("GITLABGROUPID", "123")
	t.Setenv("GITLABPROJECTID", "456")
	t.Setenv("GITLAB_URI", "https://gitlab.example.com")
	t.Setenv("LOCALPATH", "/data/gitlab")
	t.Setenv("TMPDIR", "/tmp")
	t.Setenv("S3ENDPOINT", "myendpoint")
	t.Setenv("S3BUCKETNAME", "mybucket")
	t.Setenv("S3BUCKETPATH", "mybucketpath")
	t.Setenv("S3REGION", "myregion")
	t.Setenv("AWS_ACCESS_KEY_ID", "myaccesskey")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "mysecretkey")
	t.Setenv("NOLOGTIME", "true")
	// GITLAB_TOKEN is not set
	t.Setenv("GITLAB_TOKEN", "")
	cfg, err := config.NewConfigFromEnv()
	require.NoError(t, err)
	isValid := cfg.IsConfigValid()
	require.Equal(t, false, isValid)
}
