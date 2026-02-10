// Package main provides the gitlab-backup command-line tool for backing up GitLab projects and groups.
// Copyright (C) 2021  Sylvain Gaunet

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sgaunet/gitlab-backup/pkg/app"
	"github.com/sgaunet/gitlab-backup/pkg/config"
)

var version = "development"

// cliFlags holds command-line flag values.
type cliFlags struct {
	groupID   int64
	projectID int64
	output    string
	timeout   int
	tmpdir    string
	gitlabURL string
}

func printVersion() {
	fmt.Println(version)
}

func printConfiguration() {
	c, err := config.NewConfigFromEnv()
	if err != nil {
		c = &config.Config{}
	}
	c.Usage()

	fmt.Println("--------------------------------------------------")
	fmt.Println("Gitlab-backup configuration:")
	fmt.Print(c.Redacted())
	os.Exit(0)
}

func loadConfiguration(cfgFile string) *config.Config {
	var cfg *config.Config
	var err error

	if len(cfgFile) > 0 {
		// For file config, don't validate yet (CLI overrides may fix issues)
		cfg, err = config.NewConfigFromFileNoValidate(cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading config file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Try loading from environment
		const defaultExportTimeout = 10
		cfg, err = config.NewConfigFromEnv()
		if err != nil {
			// If env loading fails, start with empty config with defaults
			cfg = &config.Config{
				GitlabURI:         "https://gitlab.com",
				TmpDir:            "/tmp",
				ExportTimeoutMins: defaultExportTimeout,
			}
		}
	}
	return cfg
}

// applyCliOverrides applies command-line flag values to the configuration.
func applyCliOverrides(cfg *config.Config, flags cliFlags) {
	// Apply group/project ID overrides
	if flags.groupID > 0 {
		cfg.GitlabGroupID = flags.groupID
	}
	if flags.projectID > 0 {
		cfg.GitlabProjectID = flags.projectID
	}

	// Apply storage override
	if flags.output != "" {
		cfg.LocalPath = flags.output
	}

	// Apply configuration overrides (only if explicitly set, -1 is sentinel for timeout)
	if flags.timeout >= 0 {
		cfg.ExportTimeoutMins = flags.timeout
	}
	if flags.tmpdir != "" {
		cfg.TmpDir = flags.tmpdir
	}
	if flags.gitlabURL != "" {
		cfg.GitlabURI = flags.gitlabURL
	}
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gitlab-backup [OPTIONS]\n\n")
		fmt.Fprintf(os.Stderr, "Backup GitLab projects and groups\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  # Backup single project to local storage\n")
		fmt.Fprintf(os.Stderr, "  gitlab-backup --project-id 123 --output /backup\n\n")
		fmt.Fprintf(os.Stderr, "  # Backup group with custom timeout\n")
		fmt.Fprintf(os.Stderr, "  gitlab-backup --group-id 456 --output /backup --timeout 30\n\n")
		fmt.Fprintf(os.Stderr, "  # Override config file values\n")
		fmt.Fprintf(os.Stderr, "  gitlab-backup -c config.yaml --timeout 20\n\n")
		fmt.Fprintf(os.Stderr, "  # Backup to S3 (S3 config must be in config file)\n")
		fmt.Fprintf(os.Stderr, "  gitlab-backup -c s3-config.yaml --project-id 789\n\n")
		fmt.Fprintf(os.Stderr, "CONFIGURATION PRECEDENCE:\n")
		fmt.Fprintf(os.Stderr, "  CLI flags > Config file > Environment variables\n\n")
		fmt.Fprintf(os.Stderr, "REQUIRED SETTINGS:\n")
		fmt.Fprintf(os.Stderr, "  - GitLab Token: GITLAB_TOKEN env var or gitlabtoken in config\n")
		fmt.Fprintf(os.Stderr, "  - Target: --group-id OR --project-id (or in config/env)\n")
		fmt.Fprintf(os.Stderr, "  - Storage: --output for local or S3 config in file\n")
	}
}

//nolint:funlen // Main function complexity is acceptable for CLI entry point
func main() {
	// Define flags
	configFile := flag.String("config", "", "Path to configuration file (YAML)")
	flag.StringVar(configFile, "c", "", "Path to configuration file (YAML) (shorthand)")

	groupID := flag.Int64("group-id", 0, "GitLab group ID to backup")
	projectID := flag.Int64("project-id", 0, "GitLab project ID to backup")
	output := flag.String("output", "", "Output directory for local storage")
	timeout := flag.Int("timeout", -1, "Export timeout in minutes (default: 10)")
	tmpdir := flag.String("tmpdir", "", "Temporary directory (default: /tmp)")
	gitlabURL := flag.String("gitlab-url", "", "GitLab API endpoint (default: https://gitlab.com)")

	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.BoolVar(showVersion, "v", false, "Show version and exit (shorthand)")
	showHelp := flag.Bool("help", false, "Show help and exit")
	flag.BoolVar(showHelp, "h", false, "Show help and exit (shorthand)")
	printCfg := flag.Bool("cfg", false, "Print configuration and exit")

	flag.Parse()

	// Handle utility flags first
	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	if *printCfg {
		printConfiguration()
	}

	// Load base configuration
	cfg := loadConfiguration(*configFile)

	// Apply CLI overrides
	flags := cliFlags{
		groupID:   *groupID,
		projectID: *projectID,
		output:    *output,
		timeout:   *timeout,
		tmpdir:    *tmpdir,
		gitlabURL: *gitlabURL,
	}
	applyCliOverrides(cfg, flags)

	// Validate final configuration (after all overrides)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Create context for app initialization and execution
	ctx := context.Background()

	// Initialize app
	app, err := app.NewApp(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	l := initTrace(os.Getenv("DEBUGLEVEL"), cfg.NoLogTime)
	app.SetLogger(l)
	err = app.Run(ctx)

	if err != nil {
		l.Error("error(s) occurred", "error", err)
		os.Exit(1)
	}
}
