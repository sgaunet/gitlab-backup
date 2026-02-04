package gitlab

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateCompositeArchive creates a composite tar.gz archive containing the GitLab export,
// labels JSON, and issues JSON files.
func CreateCompositeArchive(gitlabExportPath, labelsPath, issuesPath, outputPath string) error {
	// Create the output file
	//nolint:gosec // G304: File creation with user-provided path is intentional for archive output
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer func() {
		_ = outFile.Close()
	}()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(outFile)
	defer func() {
		_ = gzipWriter.Close()
	}()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		_ = tarWriter.Close()
	}()

	// Add GitLab export archive
	if err := addFileToArchive(tarWriter, gitlabExportPath, filepath.Base(gitlabExportPath)); err != nil {
		return fmt.Errorf("failed to add GitLab export to archive: %w", err)
	}

	// Add labels JSON if it exists
	if labelsPath != "" {
		if _, err := os.Stat(labelsPath); err == nil {
			if err := addFileToArchive(tarWriter, labelsPath, filepath.Base(labelsPath)); err != nil {
				return fmt.Errorf("failed to add labels to archive: %w", err)
			}
		}
	}

	// Add issues JSON if it exists
	if issuesPath != "" {
		if _, err := os.Stat(issuesPath); err == nil {
			if err := addFileToArchive(tarWriter, issuesPath, filepath.Base(issuesPath)); err != nil {
				return fmt.Errorf("failed to add issues to archive: %w", err)
			}
		}
	}

	return nil
}

// addFileToArchive adds a single file to the tar archive with the specified archive name.
func addFileToArchive(tarWriter *tar.Writer, filePath, archiveName string) error {
	// Open the file
	//nolint:gosec // G304: File read with user-provided path is intentional for archive creation
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// Create tar header
	header := &tar.Header{
		Name:    archiveName,
		Size:    fileInfo.Size(),
		Mode:    int64(fileInfo.Mode()),
		ModTime: fileInfo.ModTime(),
	}

	// Write header
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", archiveName, err)
	}

	// Copy file contents
	if _, err := io.Copy(tarWriter, file); err != nil {
		return fmt.Errorf("failed to write file %s to tar: %w", archiveName, err)
	}

	return nil
}
