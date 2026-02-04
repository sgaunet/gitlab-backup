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

// ArchiveContents represents the extracted contents of a composite archive.
type ArchiveContents struct {
	// ProjectExportPath is the path to GitLab native export (*-gitlab.tar.gz or project.tar.gz).
	ProjectExportPath string
	// LabelsJSONPath is the path to labels JSON (labels-*.json or labels.json) (optional).
	LabelsJSONPath string
	// IssuesJSONPath is the path to issues JSON (issues-*.json or issues.json) (optional).
	IssuesJSONPath string
	// ExtractionDir is the temporary directory for extracted files.
	ExtractionDir string
}

// HasLabels returns true if labels.json is present in the archive.
func (ac *ArchiveContents) HasLabels() bool {
	return ac.LabelsJSONPath != ""
}

// HasIssues returns true if issues.json is present in the archive.
func (ac *ArchiveContents) HasIssues() bool {
	return ac.IssuesJSONPath != ""
}

// ExtractArchive extracts a tar.gz archive to a destination directory.
// It implements path traversal protection to prevent malicious archives
// from writing outside the destination directory.
//
// Returns ArchiveContents with paths to extracted files.
// Validates that GitLab export file (*-gitlab.tar.gz or project.tar.gz) is present in the archive.
func ExtractArchive(ctx context.Context, archivePath string, destDir string) (*ArchiveContents, error) {
	// Open archive file
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Initialize archive contents
	contents := &ArchiveContents{
		ExtractionDir: destDir,
	}

	// Extract files
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Validate and clean the path to prevent traversal attacks
		cleanPath := filepath.Clean(header.Name)
		if !isValidPath(cleanPath) {
			return nil, fmt.Errorf("invalid path in archive (path traversal detected): %s", header.Name)
		}

		// Build full destination path
		targetPath := filepath.Join(destDir, cleanPath)

		// Ensure target path is within destDir (defense in depth)
		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("path traversal detected: %s", header.Name)
		}

		// Extract based on file type
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			// Copy file contents
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return nil, fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()

			// Track extracted files
			// gitlab-backup creates files with patterns:
			// - {projectname}-{projectid}-gitlab.tar.gz
			// - labels-{projectid}.json
			// - issues-{projectid}.json
			basename := filepath.Base(cleanPath)
			if strings.HasSuffix(basename, "-gitlab.tar.gz") || basename == "project.tar.gz" {
				contents.ProjectExportPath = targetPath
			} else if strings.HasPrefix(basename, "labels-") && strings.HasSuffix(basename, ".json") || basename == "labels.json" {
				contents.LabelsJSONPath = targetPath
			} else if strings.HasPrefix(basename, "issues-") && strings.HasSuffix(basename, ".json") || basename == "issues.json" {
				contents.IssuesJSONPath = targetPath
			}

		default:
			// Skip other file types (symlinks, etc.)
			continue
		}
	}

	// Validate required files
	if contents.ProjectExportPath == "" {
		return nil, errors.New("archive does not contain required GitLab export file (*-gitlab.tar.gz or project.tar.gz)")
	}

	return contents, nil
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
	if strings.HasPrefix(cleaned, "..") {
		return false
	}

	return true
}

// ValidateArchive validates that a file is a valid tar.gz archive.
// It checks the file exists, is readable, and has valid gzip/tar format.
// It does NOT extract the archive.
func ValidateArchive(archivePath string) error {
	// Check file exists
	info, err := os.Stat(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("archive not found: %s", archivePath)
		}
		return fmt.Errorf("failed to stat archive: %w", err)
	}

	// Check it's a regular file
	if info.IsDir() {
		return fmt.Errorf("archive path is a directory: %s", archivePath)
	}

	// Check file size is reasonable (not empty)
	if info.Size() == 0 {
		return fmt.Errorf("archive is empty: %s", archivePath)
	}

	// Open file
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Validate gzip format by creating reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("invalid gzip format: %w", err)
	}
	defer gzr.Close()

	// Validate tar format by reading first header
	tr := tar.NewReader(gzr)
	_, err = tr.Next()
	if err != nil && err != io.EOF {
		return fmt.Errorf("invalid tar format: %w", err)
	}

	return nil
}
