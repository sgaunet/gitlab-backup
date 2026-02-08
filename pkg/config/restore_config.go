package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrNotS3Source is returned when trying to parse a non-S3 source as S3.
	ErrNotS3Source = errors.New("not an S3 source")
	// ErrInvalidS3PathFormat is returned when S3 path format is invalid.
	ErrInvalidS3PathFormat = errors.New("invalid S3 path format (expected s3://bucket/key)")
	// ErrInvalidS3Path is returned when S3 bucket or key is empty.
	ErrInvalidS3Path = errors.New("invalid S3 path: bucket and key cannot be empty")
)

// RestoreConfig extends Config with restore-specific fields.
type RestoreConfig struct {
	Config

	// RestoreSource is the path to the archive (local or S3)
	RestoreSource string `env:"RESTORE_SOURCE" yaml:"restoreSource"`
	// RestoreTargetNS is the target namespace/group path
	RestoreTargetNS string `env:"RESTORE_TARGET_NS" yaml:"restoreTargetNS"`
	// RestoreTargetPath is the target project path
	RestoreTargetPath string `env:"RESTORE_TARGET_PATH" yaml:"restoreTargetPath"`
	// RestoreOverwrite allows overwriting existing project content
	RestoreOverwrite bool `env:"RESTORE_OVERWRITE" env-default:"false" yaml:"restoreOverwrite"`
}

// ValidateRestore performs validation specific to restore operations.
func (c *RestoreConfig) ValidateRestore() error {
	// First validate base configuration
	if err := c.Validate(); err != nil {
		return err
	}

	// Validate restore-specific fields
	if err := c.validateRestoreSource(); err != nil {
		return err
	}

	if err := c.validateRestoreTarget(); err != nil {
		return err
	}

	return nil
}

// GetFullProjectPath returns the full project path including namespace.
func (c *RestoreConfig) GetFullProjectPath() string {
	if c.RestoreTargetNS == "" {
		return c.RestoreTargetPath
	}
	return filepath.Join(c.RestoreTargetNS, c.RestoreTargetPath)
}

// IsS3Source returns true if the restore source is an S3 path.
func (c *RestoreConfig) IsS3Source() bool {
	return strings.HasPrefix(c.RestoreSource, "s3://")
}

// ParseS3Source extracts bucket and key from an S3 path (s3://bucket/key).
func (c *RestoreConfig) ParseS3Source() (string, string, error) {
	if !c.IsS3Source() {
		return "", "", ErrNotS3Source
	}

	// Remove s3:// prefix
	path := strings.TrimPrefix(c.RestoreSource, "s3://")

	// Split into bucket and key
	const expectedParts = 2
	parts := strings.SplitN(path, "/", expectedParts)
	if len(parts) != expectedParts {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidS3PathFormat, c.RestoreSource)
	}

	bucketName := parts[0]
	keyPath := parts[1]

	if bucketName == "" || keyPath == "" {
		return "", "", fmt.Errorf("%w in %s", ErrInvalidS3Path, c.RestoreSource)
	}

	return bucketName, keyPath, nil
}

//nolint:err113 // validation errors are intentionally dynamic to include context
func (c *RestoreConfig) validateRestoreSource() error {
	if c.RestoreSource == "" {
		return errors.New("restoreSource is required")
	}

	// Check if it's an S3 path
	if strings.HasPrefix(c.RestoreSource, "s3://") {
		if !c.IsS3ConfigValid() {
			return errors.New("S3 configuration required for S3 archive source")
		}
		return nil
	}

	// For local paths, validate it's a tar.gz file
	if !strings.HasSuffix(c.RestoreSource, ".tar.gz") {
		return errors.New("archive must be a .tar.gz file")
	}

	// Check for path traversal in local paths
	if err := validatePath(c.RestoreSource, "restore source"); err != nil {
		return err
	}

	return nil
}

//nolint:err113 // validation errors are intentionally dynamic to include context
func (c *RestoreConfig) validateRestoreTarget() error {
	if c.RestoreTargetPath == "" {
		return errors.New("restoreTargetPath is required")
	}

	// Validate project path format: alphanumeric, underscores, dots, hyphens
	validProjectPath := regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
	if !validProjectPath.MatchString(c.RestoreTargetPath) {
		return fmt.Errorf(
			"invalid restoreTargetPath: must contain only letters, numbers, underscores, dots, and hyphens, got %s",
			c.RestoreTargetPath,
		)
	}

	// RestoreTargetNS can be empty (restoring to user's personal namespace)
	if c.RestoreTargetNS != "" {
		// Validate namespace path (can contain slashes for nested groups)
		if err := validatePath(c.RestoreTargetNS, "restore target namespace"); err != nil {
			return err
		}

		// Check for valid namespace format
		for part := range strings.SplitSeq(c.RestoreTargetNS, "/") {
			if part == "" {
				return errors.New("restoreTargetNS cannot contain empty path segments")
			}
			if !validProjectPath.MatchString(part) {
				return fmt.Errorf(
					"invalid restoreTargetNS segment: must contain only letters, numbers, underscores, dots, and hyphens, got %s",
					part,
				)
			}
		}
	}

	return nil
}
