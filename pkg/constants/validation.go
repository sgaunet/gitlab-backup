package constants

// Configuration Timeouts and Limits.
const (
	// MaxExportTimeoutMinutes is the maximum allowed export timeout in minutes.
	// Prevents unreasonably long timeouts (e.g., typos in config).
	// Maximum: 24 hours = 1440 minutes.
	MaxExportTimeoutMinutes = 1440

	// DefaultTimeoutSeconds is the default timeout for HTTP operations in seconds.
	// Default: 1 hour = 3600 seconds.
	DefaultTimeoutSeconds = 3600
)

// AWS S3 Validation Constants
//
// These limits are defined by AWS S3 bucket naming rules.
// Reference: https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html
const (
	// S3BucketNameMinLength is the minimum allowed S3 bucket name length.
	S3BucketNameMinLength = 3

	// S3BucketNameMaxLength is the maximum allowed S3 bucket name length.
	S3BucketNameMaxLength = 63

	// S3RegionMinLength is the minimum allowed AWS region string length.
	// Shortest region is "us-east-1" (9 chars), but allow 2 for validation flexibility.
	S3RegionMinLength = 2

	// S3RegionMaxLength is the maximum allowed AWS region string length.
	// Longest region is ~20 characters (e.g., "ap-southeast-3").
	S3RegionMaxLength = 20
)

// S3 Path Parsing.
const (
	// S3PathMinParts is the minimum number of path components for S3 paths.
	// Format: bucket/key → 2 parts.
	S3PathMinParts = 2
)

// Configuration Redaction.
const (
	// RedactedValue is the placeholder for redacted credentials in logs/output.
	RedactedValue = "***REDACTED***"
)
