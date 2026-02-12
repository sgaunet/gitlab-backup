package constants

// Size Constants
//
// Standard binary size units (powers of 1024, not 1000).
const (
	// Byte is the base unit (included for completeness).
	Byte = 1

	// KB is one kilobyte (1,024 bytes).
	KB = 1024

	// MB is one megabyte (1,024 kilobytes = 1,048,576 bytes).
	MB = 1024 * KB

	// GB is one gigabyte (1,024 megabytes = 1,073,741,824 bytes).
	GB = 1024 * MB

	// TB is one terabyte (1,024 gigabytes).
	TB = 1024 * GB
)

// Buffer Sizes
//
// These control memory allocation for file operations.
// Larger buffers improve throughput but use more memory.
const (
	// CopyBufferSize is the buffer size for local file copy operations.
	// 32KB provides good balance between memory usage and I/O performance.
	CopyBufferSize = 32 * KB

	// DefaultBufferSize is the default buffer size for generic I/O operations.
	DefaultBufferSize = 32 * KB
)

// File Permissions
//
// Standard Unix file permission constants.
const (
	// DefaultFilePermission is the default permission mode for created files (rw-r--r--).
	// Owner can read/write, group/others can read.
	DefaultFilePermission = 0644

	// DefaultDirPermission is the default permission mode for created directories (rwxr-xr-x).
	// Owner can read/write/execute, group/others can read/execute.
	DefaultDirPermission = 0755
)
