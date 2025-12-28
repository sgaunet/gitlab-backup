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
	// With validation enabled, this should fail
	require.Error(t, err)
	require.Contains(t, err.Error(), "gitlabToken is required")
	require.Nil(t, cfg)
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

func TestConfigRedacted(t *testing.T) {
	// Test that the Redacted method properly redacts sensitive fields
	cfg := &config.Config{
		GitlabGroupID:   123,
		GitlabProjectID: 456,
		GitlabToken:     "super-secret-token",
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
			AccessKey:  "my-secret-access-key",
			SecretKey:  "my-secret-secret-key",
		},
	}

	redacted := cfg.Redacted()

	// Check that sensitive fields are redacted
	require.Contains(t, redacted, "gitlabToken: '***REDACTED***'")
	require.Contains(t, redacted, "accessKey: '***REDACTED***'")
	require.Contains(t, redacted, "secretKey: '***REDACTED***'")

	// Check that sensitive fields are NOT in the output
	require.NotContains(t, redacted, "super-secret-token")
	require.NotContains(t, redacted, "my-secret-access-key")
	require.NotContains(t, redacted, "my-secret-secret-key")

	// Check that non-sensitive fields are still present
	require.Contains(t, redacted, "gitlabGroupID: 123")
	require.Contains(t, redacted, "gitlabProjectID: 456")
	require.Contains(t, redacted, "gitlabURI: https://gitlab.com")
	require.Contains(t, redacted, "localpath: /data/gitlab")
	require.Contains(t, redacted, "tmpdir: /tmp")
	require.Contains(t, redacted, "prebackup: echo prebackup")
	require.Contains(t, redacted, "postbackup: echo postbackup")
	require.Contains(t, redacted, "endpoint: myendpoint")
	require.Contains(t, redacted, "bucketName: mybucket")
	require.Contains(t, redacted, "bucketPath: mybucketpath")
	require.Contains(t, redacted, "region: myregion")
	require.Contains(t, redacted, "noLogTime: false")
}

func TestConfigRedactedEmptySecrets(t *testing.T) {
	// Test that the Redacted method handles empty secrets gracefully
	cfg := &config.Config{
		GitlabGroupID:   123,
		GitlabProjectID: 456,
		GitlabToken:     "", // empty token
		GitlabURI:       "https://gitlab.com",
		LocalPath:       "/data/gitlab",
		S3cfg: config.S3Config{
			Endpoint:   "myendpoint",
			BucketName: "mybucket",
			BucketPath: "mybucketpath",
			Region:     "myregion",
			AccessKey:  "", // empty access key
			SecretKey:  "", // empty secret key
		},
	}

	redacted := cfg.Redacted()

	// Empty fields should remain empty, not redacted
	require.Contains(t, redacted, "gitlabToken: \"\"")
	require.Contains(t, redacted, "accessKey: \"\"")
	require.Contains(t, redacted, "secretKey: \"\"")

	// Should NOT contain the redaction marker
	require.NotContains(t, redacted, "***REDACTED***")
}

func TestConfigValidate_Success(t *testing.T) {
	// Test successful validation with local storage
	cfg := &config.Config{
		GitlabGroupID:     123,
		GitlabToken:       "test-token",
		GitlabURI:         "https://gitlab.com",
		LocalPath:         "/tmp",
		TmpDir:            "/tmp",
		ExportTimeoutMins: 10,
	}

	err := cfg.Validate()
	require.NoError(t, err)
}

func TestConfigValidate_S3Storage(t *testing.T) {
	// Test successful validation with S3 storage
	cfg := &config.Config{
		GitlabGroupID:     123,
		GitlabToken:       "test-token",
		GitlabURI:         "https://gitlab.example.com",
		TmpDir:            "/tmp",
		ExportTimeoutMins: 30,
		S3cfg: config.S3Config{
			BucketName: "my-backup-bucket",
			BucketPath: "backups",
			Region:     "us-east-1",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)
}

func TestConfigValidate_MissingGroupAndProject(t *testing.T) {
	cfg := &config.Config{
		GitlabToken:       "test-token",
		GitlabURI:         "https://gitlab.com",
		LocalPath:         "/tmp",
		TmpDir:            "/tmp",
		ExportTimeoutMins: 10,
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "either gitlabGroupID or gitlabProjectID must be set")
}

func TestConfigValidate_MissingToken(t *testing.T) {
	cfg := &config.Config{
		GitlabGroupID:     123,
		GitlabURI:         "https://gitlab.com",
		LocalPath:         "/tmp",
		TmpDir:            "/tmp",
		ExportTimeoutMins: 10,
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "gitlabToken is required")
}

func TestConfigValidate_NoStorage(t *testing.T) {
	cfg := &config.Config{
		GitlabGroupID:     123,
		GitlabToken:       "test-token",
		GitlabURI:         "https://gitlab.com",
		TmpDir:            "/tmp",
		ExportTimeoutMins: 10,
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "either S3 storage or local storage must be configured")
}

func TestConfigValidate_TimeoutTooLow(t *testing.T) {
	cfg := &config.Config{
		GitlabGroupID:     123,
		GitlabToken:       "test-token",
		GitlabURI:         "https://gitlab.com",
		LocalPath:         "/tmp",
		TmpDir:            "/tmp",
		ExportTimeoutMins: 0,
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "exportTimeoutMins must be at least 1 minute")
}

func TestConfigValidate_TimeoutTooHigh(t *testing.T) {
	cfg := &config.Config{
		GitlabGroupID:     123,
		GitlabToken:       "test-token",
		GitlabURI:         "https://gitlab.com",
		LocalPath:         "/tmp",
		TmpDir:            "/tmp",
		ExportTimeoutMins: 1500,
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "exportTimeoutMins must not exceed 1440 minutes")
}

func TestConfigValidate_TmpDirNotExists(t *testing.T) {
	cfg := &config.Config{
		GitlabGroupID:     123,
		GitlabToken:       "test-token",
		GitlabURI:         "https://gitlab.com",
		LocalPath:         "/tmp",
		TmpDir:            "/nonexistent/directory",
		ExportTimeoutMins: 10,
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "tmpdir")
	require.Contains(t, err.Error(), "does not exist")
}

func TestConfigValidate_InvalidGitlabURI(t *testing.T) {
	testCases := []struct {
		name string
		uri  string
		err  string
	}{
		{
			name: "invalid URL format",
			uri:  "not-a-url",
			err:  "gitlabURI must use http or https scheme",
		},
		{
			name: "ftp scheme",
			uri:  "ftp://gitlab.com",
			err:  "gitlabURI must use http or https scheme",
		},
		{
			name: "no host",
			uri:  "https://",
			err:  "gitlabURI must have a host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				GitlabGroupID:     123,
				GitlabToken:       "test-token",
				GitlabURI:         tc.uri,
				LocalPath:         "/tmp",
				TmpDir:            "/tmp",
				ExportTimeoutMins: 10,
			}

			err := cfg.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.err)
		})
	}
}

func TestConfigValidate_S3BucketName(t *testing.T) {
	testCases := []struct {
		name       string
		bucketName string
		shouldFail bool
		errMsg     string
	}{
		{
			name:       "valid bucket name",
			bucketName: "my-backup-bucket",
			shouldFail: false,
		},
		{
			name:       "valid with numbers",
			bucketName: "backup-123",
			shouldFail: false,
		},
		{
			name:       "too short",
			bucketName: "ab",
			shouldFail: true,
			errMsg:     "bucket name must be between 3 and 63 characters",
		},
		{
			name:       "too long",
			bucketName: "this-is-a-very-long-bucket-name-that-exceeds-the-maximum-allowed-length-of-63-characters",
			shouldFail: true,
			errMsg:     "bucket name must be between 3 and 63 characters",
		},
		{
			name:       "uppercase letters",
			bucketName: "MyBucket",
			shouldFail: true,
			errMsg:     "bucket name must be lowercase",
		},
		{
			name:       "consecutive dots",
			bucketName: "my..bucket",
			shouldFail: true,
			errMsg:     "bucket name cannot contain consecutive dots",
		},
		{
			name:       "IP address format",
			bucketName: "192.168.1.1",
			shouldFail: true,
			errMsg:     "bucket name cannot be formatted as an IP address",
		},
		{
			name:       "starts with hyphen",
			bucketName: "-mybucket",
			shouldFail: true,
			errMsg:     "bucket name must be DNS-compliant",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				GitlabGroupID:     123,
				GitlabToken:       "test-token",
				GitlabURI:         "https://gitlab.com",
				TmpDir:            "/tmp",
				ExportTimeoutMins: 10,
				S3cfg: config.S3Config{
					BucketName: tc.bucketName,
					BucketPath: "backups",
					Region:     "us-east-1",
				},
			}

			err := cfg.Validate()
			if tc.shouldFail {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigValidate_S3Region(t *testing.T) {
	testCases := []struct {
		name       string
		region     string
		bucketPath string
		shouldFail bool
		errMsg     string
	}{
		{
			name:       "valid AWS region",
			region:     "us-east-1",
			bucketPath: "backups",
			shouldFail: false,
		},
		{
			name:       "valid custom region",
			region:     "local",
			bucketPath: "backups",
			shouldFail: false,
		},
		{
			name:       "empty region makes S3 invalid",
			region:     "",
			bucketPath: "",
			shouldFail: true,
			errMsg:     "either S3 storage or local storage must be configured",
		},
		{
			name:       "uppercase letters",
			region:     "US-EAST-1",
			bucketPath: "backups",
			shouldFail: true,
			errMsg:     "region must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:       "special characters",
			region:     "us_east_1",
			bucketPath: "backups",
			shouldFail: true,
			errMsg:     "region must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:       "too short",
			region:     "a",
			bucketPath: "backups",
			shouldFail: true,
			errMsg:     "region name too short",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				GitlabGroupID:     123,
				GitlabToken:       "test-token",
				GitlabURI:         "https://gitlab.com",
				TmpDir:            "/tmp",
				ExportTimeoutMins: 10,
				S3cfg: config.S3Config{
					BucketName: "my-bucket",
					BucketPath: tc.bucketPath,
					Region:     tc.region,
				},
			}

			err := cfg.Validate()
			if tc.shouldFail {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigValidate_PathTraversal(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		pathType   string
		shouldFail bool
	}{
		{
			name:       "valid local path",
			path:       "/data/backups",
			pathType:   "local",
			shouldFail: false,
		},
		{
			name:       "valid S3 path",
			path:       "backups/gitlab",
			pathType:   "s3",
			shouldFail: false,
		},
		{
			name:       "path traversal in local path",
			path:       "/tmp/../etc/passwd",
			pathType:   "local",
			shouldFail: true,
		},
		{
			name:       "path traversal in S3 path",
			path:       "backups/../../secrets",
			pathType:   "s3",
			shouldFail: true,
		},
		{
			name:       "relative path traversal",
			path:       "../outside",
			pathType:   "s3",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				GitlabGroupID:     123,
				GitlabToken:       "test-token",
				GitlabURI:         "https://gitlab.com",
				TmpDir:            "/tmp",
				ExportTimeoutMins: 10,
			}

			if tc.pathType == "local" {
				cfg.LocalPath = tc.path
			} else {
				cfg.S3cfg = config.S3Config{
					BucketName: "my-bucket",
					BucketPath: tc.path,
					Region:     "us-east-1",
				}
			}

			err := cfg.Validate()
			if tc.shouldFail {
				require.Error(t, err)
				require.Contains(t, err.Error(), "path traversal")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
