// Package localstorage provides local file system storage implementation.
package localstorage

import (
	"context"
	"fmt"
	"io"
	"os"
)

// LocalStorage implements storage interface for local file system.
type LocalStorage struct {
	dirpath string
}

// NewLocalStorage creates a new LocalStorage instance.
func NewLocalStorage(dirpath string) *LocalStorage {
	return &LocalStorage{
		dirpath: dirpath,
	}
}

// SaveFile saves the file in localstorage with context cancellation support.
// For large files, the copy operation checks for cancellation periodically.
func (s *LocalStorage) SaveFile(ctx context.Context, archiveFilePath string, dstFilename string) error {
	// Check context before starting
	if ctx.Err() != nil {
		return fmt.Errorf("operation cancelled before starting: %w", ctx.Err())
	}

	src, err := os.Open(archiveFilePath) //nolint:gosec // G304: File inclusion is intentional for backup functionality
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", archiveFilePath, err)
	}
	defer func() { _ = src.Close() }()

	// save file in localstorage
	//nolint:gosec // G304: File creation is intentional for backup functionality
	dstPath := s.dirpath + "/" + dstFilename
	fDst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s/%s: %w", s.dirpath, dstFilename, err)
	}
	defer func() { _ = fDst.Close() }()

	// Use context-aware copy with periodic cancellation checks
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		// Check context before each chunk
		if ctx.Err() != nil {
			_ = fDst.Close()
			_ = os.Remove(dstPath) // Clean up partial file
			return fmt.Errorf("copy cancelled: %w", ctx.Err())
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := fDst.Write(buf[0:nr])
			if ew != nil {
				_ = os.Remove(dstPath) // Clean up on write error
				return fmt.Errorf("failed to write to destination: %w", ew)
			}
			if nr != nw {
				_ = os.Remove(dstPath) // Clean up on short write
				return fmt.Errorf("short write: wrote %d bytes, expected %d", nw, nr)
			}
		}
		if er != nil {
			if er != io.EOF {
				_ = os.Remove(dstPath) // Clean up on read error
				return fmt.Errorf("failed to read from source: %w", er)
			}
			break // EOF reached, copy complete
		}
	}

	return nil
}
