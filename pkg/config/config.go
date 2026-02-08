// Package config provides configuration management for GitLab backup application.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/sgaunet/gitlab-backup/pkg/hooks"
	"gopkg.in/yaml.v3"
)

const (
	redactedValue           = "***REDACTED***"
	maxExportTimeoutMinutes = 1440 // 24 hours
	minRegionLength         = 2
)

// S3Config holds the configuration for S3 storage backend.
type S3Config struct {
	Endpoint   string `env:"S3ENDPOINT"            env-default:""   yaml:"endpoint"`
	BucketName string `env:"S3BUCKETNAME"          env-default:""   yaml:"bucketName"`
	BucketPath string `env:"S3BUCKETPATH"          env-default:""   yaml:"bucketPath"`
	Region     string `env:"S3REGION"              env-default:""   yaml:"region"`
	AccessKey  string `env:"AWS_ACCESS_KEY_ID"     yaml:"accessKey"`
	SecretKey  string `env:"AWS_SECRET_ACCESS_KEY" yaml:"secretKey"`
}

// Config holds the application configuration.
type Config struct {
	GitlabGroupID      int64       `env:"GITLABGROUPID"      env-default:"0"                  yaml:"gitlabGroupID"`
	GitlabProjectID    int64       `env:"GITLABPROJECTID"    env-default:"0"                  yaml:"gitlabProjectID"`
	GitlabToken        string      `env:"GITLAB_TOKEN"       yaml:"gitlabToken"`
	GitlabURI          string      `env:"GITLAB_URI"         env-default:"https://gitlab.com" yaml:"gitlabURI"`
	LocalPath          string      `env:"LOCALPATH"          env-default:""                   yaml:"localpath"`
	TmpDir             string      `env:"TMPDIR"             env-default:"/tmp"               yaml:"tmpdir"`
	ExportTimeoutMins  int         `env:"EXPORT_TIMEOUT_MIN" env-default:"10"                 yaml:"exportTimeoutMins"`
	Hooks              hooks.Hooks `yaml:"hooks"`
	S3cfg              S3Config    `yaml:"s3cfg"`
	NoLogTime          bool        `env:"NOLOGTIME"          env-default:"false"              yaml:"noLogTime"`
	// Restore-specific fields (set via CLI flags, not config file)
	RestoreSource      string `yaml:"-"` // Archive path (local or s3://)
	RestoreTargetNS    string `yaml:"-"` // Target namespace/group
	RestoreTargetPath  string `yaml:"-"` // Target project path
	RestoreOverwrite   bool   `yaml:"-"` // Overwrite existing project content
	StorageType        string `yaml:"-"` // Storage type: "local" or "s3"
}

// NewConfigFromFile returns a new Config struct from the given file.
func NewConfigFromFile(filePath string) (*Config, error) {
	var cfg Config
	err := cleanenv.ReadConfig(filePath, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from file %s: %w", filePath, err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// NewConfigFromFileNoValidate loads config without validation (for CLI override pattern).
func NewConfigFromFileNoValidate(filePath string) (*Config, error) {
	var cfg Config
	err := cleanenv.ReadConfig(filePath, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from file %s: %w", filePath, err)
	}
	return &cfg, nil
}

// NewConfigFromEnv returns a new Config struct from the environment variables.
// It does not perform validation - validation happens after CLI overrides in main().
func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from environment: %w", err)
	}

	// Don't validate yet - CLI overrides may fix validation errors
	return &cfg, nil
}

// IsS3ConfigValid returns true if the S3 config is valid.
func (c *Config) IsS3ConfigValid() bool {
	return len(c.S3cfg.BucketPath) > 0 && len(c.S3cfg.Region) > 0
}

// IsLocalConfigValid returns true if the local config is valid.
func (c *Config) IsLocalConfigValid() bool {
	return len(c.LocalPath) > 0
}

// IsConfigValid returns true if the config is valid.
func (c *Config) IsConfigValid() bool {
	valid := c.GitlabGroupID > 0 || c.GitlabProjectID > 0
	return (c.IsS3ConfigValid() || c.IsLocalConfigValid()) && valid && len(c.GitlabToken) > 0
}

func (c *Config) String() string {
	cyaml, err := yaml.Marshal(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	return string(cyaml)
}

// Redacted returns a YAML representation of the config with sensitive fields redacted.
func (c *Config) Redacted() string {
	redacted := *c
	if redacted.GitlabToken != "" {
		redacted.GitlabToken = redactedValue
	}
	if redacted.S3cfg.AccessKey != "" {
		redacted.S3cfg.AccessKey = redactedValue
	}
	if redacted.S3cfg.SecretKey != "" {
		redacted.S3cfg.SecretKey = redactedValue
	}
	cyaml, err := yaml.Marshal(redacted)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	return string(cyaml)
}

// Validate performs comprehensive validation of configuration parameters.
func (c *Config) Validate() error {
	// Validate basic configuration requirements
	if err := c.validateBasicConfig(); err != nil {
		return err
	}

	// Validate timeout range
	if err := c.validateTimeout(); err != nil {
		return err
	}

	// Validate TmpDir
	if err := c.validateTmpDir(); err != nil {
		return err
	}

	// Validate GitLab URI
	if err := c.validateGitlabURI(); err != nil {
		return err
	}

	// Validate storage configuration
	if err := c.validateStorageConfig(); err != nil {
		return err
	}

	return nil
}

//nolint:err113,funcorder // validation errors provide user context; grouped with Validate()
func (c *Config) validateBasicConfig() error {
	// Must have exactly one of group ID or project ID
	if c.GitlabGroupID <= 0 && c.GitlabProjectID <= 0 {
		return errors.New(
			"either gitlabGroupID or gitlabProjectID must be set " +
				"(use --group-id or --project-id flag, config file, or environment variable)",
		)
	}

	// Cannot have both group ID and project ID
	if c.GitlabGroupID > 0 && c.GitlabProjectID > 0 {
		return errors.New("cannot specify both gitlabGroupID and gitlabProjectID")
	}

	// Must have GitLab token
	if c.GitlabToken == "" {
		return errors.New(
			"gitlabToken is required " +
				"(set via config file or GITLAB_TOKEN environment variable)",
		)
	}

	// Must have either S3 or local storage configured
	if !c.IsS3ConfigValid() && !c.IsLocalConfigValid() {
		return errors.New(
			"no storage configured: " +
				"use --output for local storage or configure S3 in config file",
		)
	}

	return nil
}

//nolint:err113,funcorder // validation errors are dynamic for context; grouped with Validate()
func (c *Config) validateTimeout() error {
	if c.ExportTimeoutMins < 1 {
		return fmt.Errorf("exportTimeoutMins must be at least 1 minute, got %d", c.ExportTimeoutMins)
	}
	if c.ExportTimeoutMins > maxExportTimeoutMinutes {
		return fmt.Errorf(
			"exportTimeoutMins must not exceed %d minutes (24 hours), got %d",
			maxExportTimeoutMinutes, c.ExportTimeoutMins,
		)
	}
	return nil
}

//nolint:err113,funcorder // validation errors are dynamic for context; grouped with Validate()
func (c *Config) validateTmpDir() error {
	// Check if directory exists
	stat, err := os.Stat(c.TmpDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("tmpdir %s does not exist", c.TmpDir)
		}
		return fmt.Errorf("tmpdir %s is not accessible: %w", c.TmpDir, err)
	}

	// Check if it's a directory
	if !stat.IsDir() {
		return fmt.Errorf("tmpdir %s is not a directory", c.TmpDir)
	}

	// Check if directory is writable by attempting to create a temp file
	testFile := filepath.Join(c.TmpDir, ".write-test")
	//nolint:gosec // intentional write test to verify directory permissions
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("tmpdir %s is not writable: %w", c.TmpDir, err)
	}
	_ = f.Close()
	_ = os.Remove(testFile)

	return nil
}

//nolint:err113,funcorder // validation errors are dynamic for context; grouped with Validate()
func (c *Config) validateGitlabURI() error {
	parsedURL, err := url.Parse(c.GitlabURI)
	if err != nil {
		return fmt.Errorf("invalid gitlabURI %s: %w", c.GitlabURI, err)
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("gitlabURI must use http or https scheme, got %s", parsedURL.Scheme)
	}

	// Check host
	if parsedURL.Host == "" {
		return errors.New("gitlabURI must have a host")
	}

	return nil
}

//nolint:funcorder // grouped with Validate()
func (c *Config) validateStorageConfig() error {
	if c.IsS3ConfigValid() {
		if err := c.validateS3Config(); err != nil {
			return err
		}
	}

	if c.IsLocalConfigValid() {
		if err := c.validateLocalPath(); err != nil {
			return err
		}
	}

	return nil
}

//nolint:funcorder // grouped with Validate()
func (c *Config) validateS3Config() error {
	// Validate bucket name
	if err := validateS3BucketName(c.S3cfg.BucketName); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	// Validate region
	if err := validateS3Region(c.S3cfg.Region); err != nil {
		return fmt.Errorf("invalid S3 region: %w", err)
	}

	// Validate bucket path doesn't contain path traversal
	if err := validatePath(c.S3cfg.BucketPath, "S3 bucket path"); err != nil {
		return err
	}

	return nil
}

//nolint:funcorder // grouped with Validate()
func (c *Config) validateLocalPath() error {
	return validatePath(c.LocalPath, "local path")
}

//nolint:err113 // validation errors are intentionally dynamic to include context
func validateS3BucketName(bucketName string) error {
	if bucketName == "" {
		// Empty bucket name is allowed if S3 is not being used
		return nil
	}

	// Check length (3-63 characters)
	if len(bucketName) < 3 || len(bucketName) > 63 {
		return fmt.Errorf("bucket name must be between 3 and 63 characters, got %d", len(bucketName))
	}

	// Check if it's all lowercase (AWS requirement)
	if bucketName != strings.ToLower(bucketName) {
		return errors.New("bucket name must be lowercase")
	}

	// DNS-compliant naming: lowercase letters, numbers, dots, hyphens
	// Cannot start or end with dot or hyphen
	// Cannot have consecutive dots
	validBucketName := regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)
	if !validBucketName.MatchString(bucketName) {
		return errors.New("bucket name must be DNS-compliant (lowercase letters, numbers, dots, hyphens only)")
	}

	// Cannot have consecutive dots
	if strings.Contains(bucketName, "..") {
		return errors.New("bucket name cannot contain consecutive dots")
	}

	// Cannot be formatted as IP address
	ipPattern := regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)
	if ipPattern.MatchString(bucketName) {
		return errors.New("bucket name cannot be formatted as an IP address")
	}

	return nil
}

//nolint:err113 // validation errors are intentionally dynamic to include context
func validateS3Region(region string) error {
	if region == "" {
		return errors.New("region cannot be empty")
	}

	// AWS region format: lowercase letters, numbers, and hyphens
	validRegion := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !validRegion.MatchString(region) {
		return errors.New("region must contain only lowercase letters, numbers, and hyphens")
	}

	// Typical AWS region pattern (e.g., us-east-1, eu-west-2)
	// This is lenient to allow custom regions
	if len(region) < minRegionLength {
		return errors.New("region name too short")
	}

	return nil
}

//nolint:err113 // validation errors are intentionally dynamic to include context
func validatePath(path, pathType string) error {
	if path == "" {
		return nil // Empty paths are handled by IsConfigValid
	}

	// Check for path traversal sequences in the original path
	if strings.Contains(path, "..") {
		return fmt.Errorf("%s contains path traversal sequence: %s", pathType, path)
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// For relative paths, ensure they don't try to escape the working directory
	if !filepath.IsAbs(path) {
		// After cleaning, relative paths should not start with ../
		if strings.HasPrefix(cleanPath, "..") {
			return fmt.Errorf("%s attempts to traverse outside working directory: %s", pathType, path)
		}
	}

	return nil
}

// Usage prints the usage of the config.
func (c *Config) Usage() {
	f := cleanenv.Usage(c, nil)
	f()
}
