package main

import (
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestApplyCliOverrides_GroupID(t *testing.T) {
	baseCfg := &config.Config{
		GitlabGroupID:   100,
		GitlabProjectID: 0,
	}
	flags := cliFlags{
		groupID: 200,
	}

	applyCliOverrides(baseCfg, flags)

	assert.Equal(t, int64(200), baseCfg.GitlabGroupID)
	assert.Equal(t, int64(0), baseCfg.GitlabProjectID)
}

func TestApplyCliOverrides_ProjectID(t *testing.T) {
	baseCfg := &config.Config{
		GitlabProjectID: 100,
		GitlabGroupID:   0,
	}
	flags := cliFlags{
		projectID: 200,
	}

	applyCliOverrides(baseCfg, flags)

	assert.Equal(t, int64(200), baseCfg.GitlabProjectID)
	assert.Equal(t, int64(0), baseCfg.GitlabGroupID)
}

func TestApplyCliOverrides_Output(t *testing.T) {
	baseCfg := &config.Config{
		LocalPath: "/old/path",
	}
	flags := cliFlags{
		output: "/new/path",
	}

	applyCliOverrides(baseCfg, flags)

	assert.Equal(t, "/new/path", baseCfg.LocalPath)
}

func TestApplyCliOverrides_Timeout(t *testing.T) {
	baseCfg := &config.Config{
		ExportTimeoutMins: 10,
	}
	flags := cliFlags{
		timeout: 30,
	}

	applyCliOverrides(baseCfg, flags)

	assert.Equal(t, 30, baseCfg.ExportTimeoutMins)
}

func TestApplyCliOverrides_TimeoutNotSet(t *testing.T) {
	baseCfg := &config.Config{
		ExportTimeoutMins: 10,
	}
	flags := cliFlags{
		timeout: -1, // Sentinel value for "not set"
	}

	applyCliOverrides(baseCfg, flags)

	assert.Equal(t, 10, baseCfg.ExportTimeoutMins) // Should remain unchanged
}

func TestApplyCliOverrides_TmpDir(t *testing.T) {
	baseCfg := &config.Config{
		TmpDir: "/tmp",
	}
	flags := cliFlags{
		tmpdir: "/custom/tmp",
	}

	applyCliOverrides(baseCfg, flags)

	assert.Equal(t, "/custom/tmp", baseCfg.TmpDir)
}

func TestApplyCliOverrides_GitLabURL(t *testing.T) {
	baseCfg := &config.Config{
		GitlabURI: "https://gitlab.com",
	}
	flags := cliFlags{
		gitlabURL: "https://gitlab.example.com",
	}

	applyCliOverrides(baseCfg, flags)

	assert.Equal(t, "https://gitlab.example.com", baseCfg.GitlabURI)
}

func TestApplyCliOverrides_MultipleOverrides(t *testing.T) {
	baseCfg := &config.Config{
		GitlabProjectID:   100,
		LocalPath:         "/old/path",
		ExportTimeoutMins: 10,
		TmpDir:            "/tmp",
		GitlabURI:         "https://gitlab.com",
	}
	flags := cliFlags{
		projectID: 200,
		output:    "/new/path",
		timeout:   30,
		tmpdir:    "/custom/tmp",
		gitlabURL: "https://gitlab.example.com",
	}

	applyCliOverrides(baseCfg, flags)

	assert.Equal(t, int64(200), baseCfg.GitlabProjectID)
	assert.Equal(t, "/new/path", baseCfg.LocalPath)
	assert.Equal(t, 30, baseCfg.ExportTimeoutMins)
	assert.Equal(t, "/custom/tmp", baseCfg.TmpDir)
	assert.Equal(t, "https://gitlab.example.com", baseCfg.GitlabURI)
}

func TestApplyCliOverrides_NoOverrides(t *testing.T) {
	baseCfg := &config.Config{
		GitlabProjectID:   100,
		LocalPath:         "/old/path",
		ExportTimeoutMins: 10,
		TmpDir:            "/tmp",
		GitlabURI:         "https://gitlab.com",
	}
	flags := cliFlags{
		projectID: 0,  // Not set
		groupID:   0,  // Not set
		output:    "", // Not set
		timeout:   -1, // Not set
		tmpdir:    "",
		gitlabURL: "",
	}

	applyCliOverrides(baseCfg, flags)

	// All values should remain unchanged
	assert.Equal(t, int64(100), baseCfg.GitlabProjectID)
	assert.Equal(t, "/old/path", baseCfg.LocalPath)
	assert.Equal(t, 10, baseCfg.ExportTimeoutMins)
	assert.Equal(t, "/tmp", baseCfg.TmpDir)
	assert.Equal(t, "https://gitlab.com", baseCfg.GitlabURI)
}
