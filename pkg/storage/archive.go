package storage

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	dirPermissions = 0o755
)

var (
	// ErrPathTraversal is returned when an archive contains path traversal attempts.
	ErrPathTraversal = errors.New("path traversal detected in archive")
	// ErrMissingExportFile is returned when archive doesn't contain required GitLab export file.
	ErrMissingExportFile = errors.New(
		"archive does not contain required GitLab export file (*-gitlab.tar.gz or project.tar.gz)",
	)
	// ErrArchiveNotFound is returned when archive file doesn't exist.
	ErrArchiveNotFound = errors.New("archive not found")
	// ErrArchiveIsDirectory is returned when archive path points to a directory.
	ErrArchiveIsDirectory = errors.New("archive path is a directory")
	// ErrArchiveEmpty is returned when archive file is empty.
	ErrArchiveEmpty = errors.New("archive is empty")
)

// Archive represents a composite backup archive created by gitlab-backup.
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

// ArchiveContents represents the extracted contents of a backup archive.
type ArchiveContents struct {
	// ProjectExportPath is the path to GitLab native export (*-gitlab.tar.gz or project.tar.gz).
	ProjectExportPath string
	// ExtractionDir is the temporary directory for extracted files.
	ExtractionDir string
}

// ExtractArchive extracts a tar.gz archive to a destination directory.
// It implements path traversal protection to prevent malicious archives
// from writing outside the destination directory.
//
// Returns ArchiveContents with paths to extracted files.
// Validates that GitLab export file (*-gitlab.tar.gz or project.tar.gz) is present in the archive.
//
func ExtractArchive(ctx context.Context, archivePath string, destDir string) (*ArchiveContents, error) {
	// Open and initialize tar reader
	tr, cleanup, err := openTarArchive(archivePath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Initialize archive contents
	contents := &ArchiveContents{
		ExtractionDir: destDir,
	}

	// Extract all files from archive
	if err := extractAllFiles(ctx, tr, destDir, contents); err != nil {
		return nil, err
	}

	// Validate required files
	if contents.ProjectExportPath == "" {
		return nil, ErrMissingExportFile
	}

	return contents, nil
}

// openTarArchive opens a tar.gz archive and returns a tar reader and cleanup function.
func openTarArchive(archivePath string) (*tar.Reader, func(), error) {
	// Open archive file
	//nolint:gosec // G304: Archive path is validated by caller
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open archive: %w", err)
	}

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	cleanup := func() {
		_ = gzr.Close()
		_ = file.Close()
	}

	return tar.NewReader(gzr), cleanup, nil
}

// extractAllFiles extracts all files from the tar reader to the destination directory.
func extractAllFiles(ctx context.Context, tr *tar.Reader, destDir string, contents *ArchiveContents) error {
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("extraction cancelled: %w", ctx.Err())
		default:
		}

		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Validate and extract entry
		if err := extractTarEntry(tr, header, destDir, contents); err != nil {
			return err
		}
	}
	return nil
}

// extractTarEntry extracts a single tar entry to the destination directory.
func extractTarEntry(tr *tar.Reader, header *tar.Header, destDir string, contents *ArchiveContents) error {
	// Validate path
	targetPath, err := validateTarPath(header.Name, destDir)
	if err != nil {
		return err
	}

	// Extract based on file type
	switch header.Typeflag {
	case tar.TypeDir:
		return extractDirectory(targetPath)
	case tar.TypeReg:
		return extractRegularFile(tr, targetPath, contents)
	default:
		// Skip other file types (symlinks, etc.)
		return nil
	}
}

// validateTarPath validates a tar entry path and returns the target extraction path.
func validateTarPath(name, destDir string) (string, error) {
	cleanPath := filepath.Clean(name)
	if !isValidPath(cleanPath) {
		return "", fmt.Errorf("%w: %s", ErrPathTraversal, name)
	}

	targetPath := filepath.Join(destDir, cleanPath)

	// Ensure target path is within destDir (defense in depth)
	if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %s", ErrPathTraversal, name)
	}

	return targetPath, nil
}

// extractDirectory creates a directory at the target path.
func extractDirectory(targetPath string) error {
	if err := os.MkdirAll(targetPath, dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
	}
	return nil
}

// extractRegularFile extracts a regular file from the tar reader.
func extractRegularFile(tr *tar.Reader, targetPath string, contents *ArchiveContents) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), dirPermissions); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
	}

	// Create and write file
	//nolint:gosec // G304: Target path is validated for path traversal
	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}

	if _, err := io.Copy(outFile, tr); err != nil {
		_ = outFile.Close()
		return fmt.Errorf("failed to write file %s: %w", targetPath, err)
	}

	if err := outFile.Close(); err != nil {
		return fmt.Errorf("failed to close file %s: %w", targetPath, err)
	}

	// Track GitLab export file
	trackExportFile(targetPath, contents)
	return nil
}

// trackExportFile identifies and tracks GitLab export files.
func trackExportFile(targetPath string, contents *ArchiveContents) {
	// Archives may contain:
	// - {projectname}-{projectid}.tar.gz (new format - direct GitLab export)
	// - {projectname}-{projectid}-gitlab.tar.gz (old composite archive format)
	// - project.tar.gz (generic name)
	// Old archives may also contain labels.json and issues.json (ignored for backward compatibility)
	basename := filepath.Base(targetPath)
	isGzipArchive := strings.HasSuffix(basename, ".tar.gz")
	isMetadataFile := strings.HasPrefix(basename, "labels-") || strings.HasPrefix(basename, "issues-")
	if isGzipArchive && !isMetadataFile {
		contents.ProjectExportPath = targetPath
	}
}

// isValidPath checks if a path is valid and does not contain traversal sequences.
func isValidPath(path string) bool {
	// Reject absolute paths
	if filepath.IsAbs(path) {
		return false
	}

	// Reject paths containing ..
	if strings.Contains(path, "..") {
		return false
	}

	// Clean and check again
	cleaned := filepath.Clean(path)
	return !strings.HasPrefix(cleaned, "..")
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
