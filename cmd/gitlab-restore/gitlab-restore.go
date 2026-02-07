package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sgaunet/gitlab-backup/pkg/app/restore"
	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/sgaunet/gitlab-backup/pkg/storage/localstorage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/s3storage"
)

var (
	version = "development" // Set by GoReleaser ldflags
)

func main() {
	// Define flags
	configFile := flag.String("config", "", "Path to configuration file (YAML)")
	archive := flag.String("archive", "", "Archive path (local path or s3://bucket/key)")
	namespace := flag.String("namespace", "", "Target GitLab namespace/group")
	project := flag.String("project", "", "Target GitLab project name")
	overwrite := flag.Bool("overwrite", false, "Overwrite existing project content (use with caution)")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("gitlab-restore version %s\n", version)
		os.Exit(0)
	}

	// Validate required flags
	if *configFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --config flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if *archive == "" {
		fmt.Fprintln(os.Stderr, "Error: --archive flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if *namespace == "" {
		fmt.Fprintln(os.Stderr, "Error: --namespace flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if *project == "" {
		fmt.Fprintln(os.Stderr, "Error: --project flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.NewConfigFromFile(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Override config with CLI flags
	cfg.RestoreSource = *archive
	cfg.RestoreTargetNS = *namespace
	cfg.RestoreTargetPath = *project
	cfg.RestoreOverwrite = *overwrite

	// Determine storage type from archive path
	if strings.HasPrefix(*archive, "s3://") {
		cfg.StorageType = "s3"
	} else {
		cfg.StorageType = "local"
	}

	// Setup context with cancellation (Ctrl+C handling)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\n[RESTORE] Interrupted - cleaning up...")
		cancel()
	}()

	// Initialize GitLab client
	gitlabClient := gitlab.NewGitlabServiceWithTimeout(cfg.ExportTimeoutMins)
	if gitlabClient == nil {
		fmt.Fprintf(os.Stderr, "Error initializing GitLab client\n")
		os.Exit(1)
	}
	gitlabClient.SetToken(cfg.GitlabToken)
	gitlabClient.SetGitlabEndpoint(cfg.GitlabURI)

	// Initialize storage
	var storage restore.Storage
	if cfg.StorageType == "s3" {
		s3Store, err := s3storage.NewS3Storage(cfg.S3cfg.Region, cfg.S3cfg.Endpoint, cfg.S3cfg.BucketName, cfg.S3cfg.BucketPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing S3 storage: %v\n", redactCredentials(err.Error(), cfg))
			os.Exit(1)
		}
		storage = &s3StorageAdapter{s3Store}
	} else {
		storage = &localStorageAdapter{localstorage.NewLocalStorage(cfg.LocalPath)}
	}

	// Create restore orchestrator
	orchestrator := restore.NewOrchestrator(gitlabClient, storage, cfg)

	// Execute restore
	result, err := orchestrator.Restore(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during restore: %v\n", redactCredentials(err.Error(), cfg))
		os.Exit(1)
	}

	// Print results
	printRestoreResult(result, cfg)

	// Exit with appropriate code
	if !result.Success {
		os.Exit(1)
	}
}

// redactCredentials removes sensitive information from error messages
func redactCredentials(message string, cfg *config.Config) string {
	redacted := message

	// Redact GitLab token
	if cfg.GitlabToken != "" {
		redacted = strings.ReplaceAll(redacted, cfg.GitlabToken, "***REDACTED***")
	}

	// Redact AWS credentials if present
	if cfg.S3cfg.AccessKey != "" {
		redacted = strings.ReplaceAll(redacted, cfg.S3cfg.AccessKey, "***REDACTED***")
	}
	if cfg.S3cfg.SecretKey != "" {
		redacted = strings.ReplaceAll(redacted, cfg.S3cfg.SecretKey, "***REDACTED***")
	}

	return redacted
}

// s3StorageAdapter adapts S3Storage to the restore.Storage interface.
type s3StorageAdapter struct {
	*s3storage.S3Storage
}

// Get downloads a file from S3 and returns the local path.
func (a *s3StorageAdapter) Get(ctx context.Context, key string) (string, error) {
	// Extract S3 key from s3:// URL if needed
	s3Key := key
	if strings.HasPrefix(key, "s3://") {
		// Format: s3://bucket/key
		parts := strings.SplitN(strings.TrimPrefix(key, "s3://"), "/", 2)
		if len(parts) == 2 {
			s3Key = parts[1]
		}
	}

	// Download to temporary file
	tempFile, err := os.CreateTemp("", "gitlab-restore-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Use S3Storage's GetFile method
	if err := a.S3Storage.GetFile(ctx, s3Key, tempFile.Name()); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to download from S3: %w", err)
	}

	return tempFile.Name(), nil
}

// localStorageAdapter adapts LocalStorage to the restore.Storage interface.
type localStorageAdapter struct {
	*localstorage.LocalStorage
}

// Get returns the local file path (already local, no download needed).
func (a *localStorageAdapter) Get(ctx context.Context, key string) (string, error) {
	// For local storage, the key IS the path
	return key, nil
}

// printRestoreResult displays the final restore outcome
func printRestoreResult(result *restore.RestoreResult, cfg *config.Config) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	if result.Success {
		fmt.Println("✓ RESTORE SUCCESSFUL")
	} else {
		fmt.Println("✗ RESTORE FAILED")
	}
	fmt.Println(strings.Repeat("=", 60))

	// Print project information
	if result.ProjectID != 0 {
		fmt.Printf("\nProject ID: %d\n", result.ProjectID)
		fmt.Printf("Project URL: %s\n", redactCredentials(result.ProjectURL, cfg))
	}

	// Print metrics
	fmt.Println("\nMetrics:")
	fmt.Printf("  Duration: %ds\n", result.Metrics.DurationSeconds)

	if result.Metrics.BytesDownloaded > 0 {
		fmt.Printf("  Downloaded: %.2f MB\n", float64(result.Metrics.BytesDownloaded)/(1024*1024))
	}
	if result.Metrics.BytesExtracted > 0 {
		fmt.Printf("  Extracted: %.2f MB\n", float64(result.Metrics.BytesExtracted)/(1024*1024))
	}

	// Print errors if any
	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fatalStr := ""
			if err.Fatal {
				fatalStr = " [FATAL]"
			}
			fmt.Printf("  [%s]%s %s: %s\n", err.Phase, fatalStr, err.Component, redactCredentials(err.Message, cfg))
		}
	}

	// Print warnings if any
	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  %s\n", redactCredentials(warning, cfg))
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}
