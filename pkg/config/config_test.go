package config_test

import (
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/hooks"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromFile(t *testing.T) {
	// TestNewConfigFromFile tests the NewConfigFromFile function
	t.Run("normal case", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "mytoken")
		cfg, err := config.NewConfigFromFile("testdata/good-cfg.yaml")
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, int64(123), cfg.GitlabGroupID)
		require.Equal(t, int64(456), cfg.GitlabProjectID)
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
		require.Equal(t, int64(123), cfg.GitlabGroupID)
		require.Equal(t, int64(456), cfg.GitlabProjectID)
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

func TestIsS3ConfigValid(t *testing.T) {
	t.Run("valid S3 config", func(t *testing.T) {
		cfg := &config.Config{
			S3cfg: config.S3Config{
				BucketPath: "mybucketpath",
				Region:     "myregion",
				BucketName: "mybucket", // these are not checked in IsS3ConfigValid
				Endpoint:   "myendpoint", // these are not checked in IsS3ConfigValid
			},
		}
		require.True(t, cfg.IsS3ConfigValid())
	})

	t.Run("invalid S3 config - empty BucketPath", func(t *testing.T) {
		cfg := &config.Config{
			S3cfg: config.S3Config{
				BucketPath: "", // empty BucketPath makes it invalid
				Region:     "myregion",
			},
		}
		require.False(t, cfg.IsS3ConfigValid())
	})

	t.Run("invalid S3 config - empty Region", func(t *testing.T) {
		cfg := &config.Config{
			S3cfg: config.S3Config{
				BucketPath: "mybucketpath",
				Region:     "", // empty Region makes it invalid
			},
		}
		require.False(t, cfg.IsS3ConfigValid())
	})

	t.Run("invalid S3 config - both empty", func(t *testing.T) {
		cfg := &config.Config{
			S3cfg: config.S3Config{
				BucketPath: "", 
				Region:     "",
			},
		}
		require.False(t, cfg.IsS3ConfigValid())
	})
}

func TestIsLocalConfigValid(t *testing.T) {
	t.Run("valid local config", func(t *testing.T) {
		cfg := &config.Config{
			LocalPath: "/data/gitlab",
		}
		require.True(t, cfg.IsLocalConfigValid())
	})

	t.Run("invalid local config - empty LocalPath", func(t *testing.T) {
		cfg := &config.Config{
			LocalPath: "", // empty LocalPath makes it invalid
		}
		require.False(t, cfg.IsLocalConfigValid())
	})
}

func TestIsConfigValid(t *testing.T) {
	t.Run("valid config with group ID and local storage", func(t *testing.T) {
		cfg := &config.Config{
			GitlabGroupID: 123,
			GitlabToken:   "mytoken",
			LocalPath:     "/data/gitlab",
		}
		require.True(t, cfg.IsConfigValid())
	})

	t.Run("valid config with project ID and S3 storage", func(t *testing.T) {
		cfg := &config.Config{
			GitlabProjectID: 456,
			GitlabToken:     "mytoken",
			S3cfg: config.S3Config{
				BucketPath: "mybucketpath",
				Region:     "myregion",
			},
		}
		require.True(t, cfg.IsConfigValid())
	})

	t.Run("invalid config - no group or project ID", func(t *testing.T) {
		cfg := &config.Config{
			GitlabGroupID:   0,
			GitlabProjectID: 0,
			GitlabToken:     "mytoken",
			LocalPath:       "/data/gitlab",
		}
		require.False(t, cfg.IsConfigValid())
	})

	t.Run("invalid config - empty token", func(t *testing.T) {
		cfg := &config.Config{
			GitlabGroupID: 123,
			GitlabToken:   "", // empty token makes it invalid
			LocalPath:     "/data/gitlab",
		}
		require.False(t, cfg.IsConfigValid())
	})

	t.Run("invalid config - no storage options", func(t *testing.T) {
		cfg := &config.Config{
			GitlabGroupID: 123,
			GitlabToken:   "mytoken",
			LocalPath:     "", // empty local path
			S3cfg: config.S3Config{ // invalid S3 config
				BucketPath: "",
				Region:     "",
			},
		}
		require.False(t, cfg.IsConfigValid())
	})
}

func TestConfigString(t *testing.T) {
	// Test that the String method returns a YAML representation of the config
	cfg := &config.Config{
		GitlabGroupID:   123,
		GitlabProjectID: 456,
		GitlabToken:     "mytoken",
		GitlabURI:       "https://gitlab.com",
		LocalPath:       "/data/gitlab",
		TmpDir:          "/tmp",
		NoLogTime:       false,
		Hooks: hooks.Hooks{
			PreBackup:  "echo prebackup",
			PostBackup: "echo postbackup",
		},
		S3cfg: config.S3Config{
			Endpoint:   "myendpoint",
			BucketName: "mybucket",
			BucketPath: "mybucketpath",
			Region:     "myregion",
			AccessKey:  "myaccesskey",
			SecretKey:  "mysecretkey",
		},
	}

	str := cfg.String()

	// Check that the string contains all the expected values
	require.Contains(t, str, "gitlabGroupID: 123")
	require.Contains(t, str, "gitlabProjectID: 456")
	require.Contains(t, str, "gitlabToken: mytoken")
	require.Contains(t, str, "gitlabURI: https://gitlab.com")
	require.Contains(t, str, "localpath: /data/gitlab")
	require.Contains(t, str, "tmpdir: /tmp")
	require.Contains(t, str, "prebackup: echo prebackup")
	require.Contains(t, str, "postbackup: echo postbackup")
	require.Contains(t, str, "endpoint: myendpoint")
	require.Contains(t, str, "bucketName: mybucket")
	require.Contains(t, str, "bucketPath: mybucketpath")
	require.Contains(t, str, "region: myregion")
	require.Contains(t, str, "accessKey: myaccesskey")
	require.Contains(t, str, "secretKey: mysecretkey")
	require.Contains(t, str, "noLogTime: false")
}

func TestConfigUsage(t *testing.T) {
	// Just test that the Usage method doesn't panic
	cfg := &config.Config{}

	// This should not panic
	require.NotPanics(t, func() {
		cfg.Usage()
	})
}

func TestNewConfigFromEnvEdgeCases(t *testing.T) {
	// Test invalid group ID
	t.Run("invalid group ID format", func(t *testing.T) {
		t.Setenv("GITLABGROUPID", "not_a_number")
		t.Setenv("GITLAB_TOKEN", "mytoken")
		_, err := config.NewConfigFromEnv()
		require.Error(t, err)
	})

	// Test invalid project ID
	t.Run("invalid project ID format", func(t *testing.T) {
		t.Setenv("GITLABGROUPID", "123")
		t.Setenv("GITLABPROJECTID", "not_a_number")
		t.Setenv("GITLAB_TOKEN", "mytoken")
		_, err := config.NewConfigFromEnv()
		require.Error(t, err)
	})

	// Test invalid boolean value for NoLogTime
	t.Run("invalid boolean value", func(t *testing.T) {
		t.Setenv("GITLABGROUPID", "123")
		t.Setenv("GITLAB_TOKEN", "mytoken")
		t.Setenv("NOLOGTIME", "not_a_boolean")
		_, err := config.NewConfigFromEnv()
		require.Error(t, err)
	})
}
