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

// SaveFile saves the file in localstorage.
func (s *LocalStorage) SaveFile(_ context.Context, archiveFilePath string, dstFilename string) error {
	src, err := os.Open(archiveFilePath) //nolint:gosec // G304: File inclusion is intentional for backup functionality
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", archiveFilePath, err)
	}
	defer func() { _ = src.Close() }()

	// save file in localstorage
	//nolint:gosec // G304: File creation is intentional for backup functionality
	fDst, err := os.Create(s.dirpath + "/" + dstFilename)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s/%s: %w", s.dirpath, dstFilename, err)
	}
	defer func() { _ = fDst.Close() }()
	_, err = io.Copy(fDst, src)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	return nil
}
