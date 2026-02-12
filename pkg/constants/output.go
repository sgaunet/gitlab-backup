package constants

// CLI Output Formatting
//
// These constants control the visual formatting of CLI output.
const (
	// SeparatorWidth is the character width of console separators/dividers.
	// Used for visual section breaks in restore results and status output.
	SeparatorWidth = 60
)

// File Size Display
//
// These constants are used for converting byte counts to human-readable formats.
const (
	// BytesPerKB is the number of bytes in one kilobyte (1,024).
	// Use storage.KB instead for new code.
	//
	// Deprecated: Use constants.KB from storage.go.
	BytesPerKB = 1024

	// BytesPerMB is the number of bytes in one megabyte (1,048,576).
	// Use storage.MB instead for new code.
	//
	// Deprecated: Use constants.MB from storage.go.
	BytesPerMB = BytesPerKB * BytesPerKB
)
