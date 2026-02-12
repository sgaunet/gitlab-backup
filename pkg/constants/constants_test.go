package constants_test

import (
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/constants"
)

func TestConstantValues(t *testing.T) {
	// Verify rate limits match GitLab documented limits
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"DownloadRateLimitBurst", constants.DownloadRateLimitBurst, 5},
		{"ExportRateLimitBurst", constants.ExportRateLimitBurst, 6},
		{"ImportRateLimitBurst", constants.ImportRateLimitBurst, 6},
		{"DownloadRateLimitIntervalSeconds", constants.DownloadRateLimitIntervalSeconds, 60},
		{"ExportRateLimitIntervalSeconds", constants.ExportRateLimitIntervalSeconds, 60},
		{"ImportRateLimitIntervalSeconds", constants.ImportRateLimitIntervalSeconds, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestSizeConstants(t *testing.T) {
	// Verify size calculations
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"KB", constants.KB, 1024},
		{"MB", constants.MB, 1024 * 1024},
		{"GB", constants.GB, 1024 * 1024 * 1024},
		{"TB", constants.TB, 1024 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestTimeoutConstants(t *testing.T) {
	// Verify timeouts are sensible
	if constants.DefaultExportTimeoutMins < 1 {
		t.Error("DefaultExportTimeoutMins should be at least 1 minute")
	}
	if constants.MaxExportTimeoutMinutes > 24*60 {
		t.Error("MaxExportTimeoutMinutes should not exceed 24 hours")
	}
	if constants.ImportTimeoutMinutes < 1 {
		t.Error("ImportTimeoutMinutes should be at least 1 minute")
	}
}

func TestBufferSizeConstants(t *testing.T) {
	// Verify buffer sizes are reasonable
	if constants.CopyBufferSize != 32*constants.KB {
		t.Errorf("CopyBufferSize = %d, want %d (32KB)", constants.CopyBufferSize, 32*constants.KB)
	}
	if constants.DefaultBufferSize != 32*constants.KB {
		t.Errorf("DefaultBufferSize = %d, want %d (32KB)", constants.DefaultBufferSize, 32*constants.KB)
	}
}

func TestFilePermissionConstants(t *testing.T) {
	// Verify file permissions are standard Unix permissions
	if constants.DefaultFilePermission != 0644 {
		t.Errorf("DefaultFilePermission = %o, want 0644", constants.DefaultFilePermission)
	}
	if constants.DefaultDirPermission != 0755 {
		t.Errorf("DefaultDirPermission = %o, want 0755", constants.DefaultDirPermission)
	}
}

func TestS3ValidationConstants(t *testing.T) {
	// Verify S3 bucket naming constraints match AWS rules
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"S3BucketNameMinLength", constants.S3BucketNameMinLength, 3},
		{"S3BucketNameMaxLength", constants.S3BucketNameMaxLength, 63},
		{"S3RegionMinLength", constants.S3RegionMinLength, 2},
		{"S3RegionMaxLength", constants.S3RegionMaxLength, 20},
		{"S3PathMinParts", constants.S3PathMinParts, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestOutputConstants(t *testing.T) {
	// Verify output formatting constants
	if constants.SeparatorWidth != 60 {
		t.Errorf("SeparatorWidth = %d, want 60", constants.SeparatorWidth)
	}
}

func TestEndpointConstants(t *testing.T) {
	// Verify GitLab endpoints are correct
	if constants.GitLabAPIEndpoint != "https://gitlab.com/api/v4" {
		t.Errorf("GitLabAPIEndpoint = %s, want https://gitlab.com/api/v4", constants.GitLabAPIEndpoint)
	}
	if constants.GitLabBaseURL != "https://gitlab.com" {
		t.Errorf("GitLabBaseURL = %s, want https://gitlab.com", constants.GitLabBaseURL)
	}
}

func TestRedactionConstants(t *testing.T) {
	// Verify redaction placeholder
	if constants.RedactedValue != "***REDACTED***" {
		t.Errorf("RedactedValue = %s, want ***REDACTED***", constants.RedactedValue)
	}
}
