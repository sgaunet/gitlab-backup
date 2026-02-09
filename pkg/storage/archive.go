package storage

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrArchiveNotFound is returned when archive file doesn't exist.
	ErrArchiveNotFound = errors.New("archive not found")
	// ErrArchiveIsDirectory is returned when archive path points to a directory.
	ErrArchiveIsDirectory = errors.New("archive path is a directory")
	// ErrArchiveEmpty is returned when archive file is empty.
	ErrArchiveEmpty = errors.New("archive is empty")
)

// Archive represents a backup archive created by gitlab-backup.
type Archive struct {
	// Path is the local filesystem path or S3 key.
	Path string
	// StorageType indicates where the archive is stored (local or s3).
	StorageType string
	// Size is the archive file size in bytes.
	Size int64
	// ChecksumMD5 is the archive integrity checksum.
	ChecksumMD5 string
	// Contents is the extracted archive structure.
	Contents *ArchiveContents
}

// ArchiveContents represents the contents of a backup archive.
type ArchiveContents struct {
	// ProjectExportPath is the path to GitLab native export archive.
	ProjectExportPath string
	// ExtractionDir is the temporary directory (unused but kept for API compatibility).
	ExtractionDir string
}

// ExtractArchive validates and returns the archive path for GitLab import.
// All archives created by gitlab-backup are direct GitLab exports and require no extraction.
//
// Returns ArchiveContents with the archive path.
// The ctx parameter is kept for API compatibility but not currently used.
func ExtractArchive(_ context.Context, archivePath string, destDir string) (*ArchiveContents, error) {
	// All archives are direct GitLab exports - no extraction needed
	return &ArchiveContents{
		ProjectExportPath: archivePath,
		ExtractionDir:     destDir,
	}, nil
}

// ValidateArchive validates that a file is a valid tar.gz archive.
// It checks the file exists, is readable, and has valid gzip/tar format.
// It does NOT extract the archive.
func ValidateArchive(archivePath string) error {
	// Check file exists
	info, err := os.Stat(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrArchiveNotFound, archivePath)
		}
		return fmt.Errorf("failed to stat archive: %w", err)
	}

	// Check it's a regular file
	if info.IsDir() {
		return fmt.Errorf("%w: %s", ErrArchiveIsDirectory, archivePath)
	}

	// Check file size is reasonable (not empty)
	if info.Size() == 0 {
		return fmt.Errorf("%w: %s", ErrArchiveEmpty, archivePath)
	}

	// Open file
	//nolint:gosec // G304: Archive path is provided by caller and validated
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Validate gzip format by creating reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("invalid gzip format: %w", err)
	}
	defer func() {
		_ = gzr.Close()
	}()

	// Validate tar format by reading first header
	tr := tar.NewReader(gzr)
	_, err = tr.Next()
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid tar format: %w", err)
	}

	return nil
}
