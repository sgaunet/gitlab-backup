package storage_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateArchive(t *testing.T) {
	t.Run("ValidArchive", func(t *testing.T) {
		// Create a valid tar.gz file
		archivePath := createTestArchive(t, map[string]string{
			"project.tar.gz": "fake project data",
		})

		err := storage.ValidateArchive(archivePath)
		assert.NoError(t, err, "Valid archive should pass validation")
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		err := storage.ValidateArchive("/nonexistent/file.tar.gz")
		assert.Error(t, err, "Validation should fail for non-existent file")
		assert.Contains(t, err.Error(), "archive not found", "Error should indicate file not found")
	})

	t.Run("DirectoryInsteadOfFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := storage.ValidateArchive(tmpDir)
		assert.Error(t, err, "Validation should fail for directory")
		assert.Contains(t, err.Error(), "directory", "Error should indicate it's a directory")
	})

	t.Run("EmptyFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		emptyFile := filepath.Join(tmpDir, "empty.tar.gz")
		require.NoError(t, os.WriteFile(emptyFile, []byte{}, 0644))

		err := storage.ValidateArchive(emptyFile)
		assert.Error(t, err, "Validation should fail for empty file")
		assert.Contains(t, err.Error(), "empty", "Error should indicate empty archive")
	})

	t.Run("InvalidGzipFormat", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidFile := filepath.Join(tmpDir, "invalid.tar.gz")
		require.NoError(t, os.WriteFile(invalidFile, []byte("not gzip data"), 0644))

		err := storage.ValidateArchive(invalidFile)
		assert.Error(t, err, "Validation should fail for invalid gzip")
		assert.Contains(t, err.Error(), "invalid gzip format", "Error should indicate gzip format issue")
	})
}

func TestExtractArchive(t *testing.T) {
	ctx := context.Background()

	t.Run("ValidArchive", func(t *testing.T) {
		// Create test archive
		archivePath := createTestArchive(t, map[string]string{
			"project.tar.gz": "project export data",
		})

		destDir := t.TempDir()

		// Extract archive
		contents, err := storage.ExtractArchive(ctx, archivePath, destDir)

		// Assertions
		require.NoError(t, err, "Extraction should succeed")
		require.NotNil(t, contents, "Contents should not be nil")
		assert.NotEmpty(t, contents.ProjectExportPath, "Project export path should be set")
		assert.Equal(t, destDir, contents.ExtractionDir, "Extraction dir should match")

		// Verify files exist
		assert.FileExists(t, contents.ProjectExportPath, "Project export file should exist")

		// Verify file contents
		projectData, err := os.ReadFile(contents.ProjectExportPath)
		require.NoError(t, err)
		assert.Equal(t, "project export data", string(projectData))
	})

	t.Run("BackwardCompatibilityOldArchive", func(t *testing.T) {
		// Archive with old format (includes labels.json and issues.json)
		// These files should be silently ignored for backward compatibility
		archivePath := createTestArchive(t, map[string]string{
			"myproject-123-gitlab.tar.gz": "gitlab export",
			"labels.json":                 `[{"name":"bug","color":"#FF0000"}]`,
			"issues.json":                 `[{"id":1,"title":"Test issue"}]`,
		})

		destDir := t.TempDir()
		contents, err := storage.ExtractArchive(ctx, archivePath, destDir)

		// Should succeed and only track the GitLab export file
		require.NoError(t, err, "Extraction should succeed with old archive format")
		assert.NotEmpty(t, contents.ProjectExportPath, "Project export path should be set")
		assert.Contains(t, contents.ProjectExportPath, "gitlab.tar.gz", "Should identify -gitlab.tar.gz file")

		// Verify labels.json and issues.json are extracted but not tracked
		labelsPath := filepath.Join(destDir, "labels.json")
		issuesPath := filepath.Join(destDir, "issues.json")
		assert.FileExists(t, labelsPath, "Labels file should be extracted")
		assert.FileExists(t, issuesPath, "Issues file should be extracted")
	})

	t.Run("MissingProjectTarGz", func(t *testing.T) {
		// Archive without project.tar.gz (invalid archive)
		archivePath := createTestArchive(t, map[string]string{
			"README.txt": "not a valid backup",
		})

		destDir := t.TempDir()
		contents, err := storage.ExtractArchive(ctx, archivePath, destDir)

		require.Error(t, err, "Extraction should fail without project.tar.gz")
		assert.Nil(t, contents, "Contents should be nil on error")
		assert.Contains(t, err.Error(), "GitLab export", "Error should mention missing GitLab export file")
	})

	t.Run("PathTraversalAbsolutePath", func(t *testing.T) {
		// Create archive with absolute path (security test)
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "malicious.tar.gz")

		file, err := os.Create(archivePath)
		require.NoError(t, err)
		defer file.Close()

		gzw := gzip.NewWriter(file)
		defer gzw.Close()

		tw := tar.NewWriter(gzw)
		defer tw.Close()

		// Try to write to absolute path
		maliciousPath := "/etc/passwd"
		header := &tar.Header{
			Name: maliciousPath,
			Mode: 0644,
			Size: int64(len("malicious content")),
		}
		require.NoError(t, tw.WriteHeader(header))
		_, err = tw.Write([]byte("malicious content"))
		require.NoError(t, err)

		tw.Close()
		gzw.Close()
		file.Close()

		// Attempt extraction
		destDir := t.TempDir()
		contents, err := storage.ExtractArchive(ctx, archivePath, destDir)

		// Should fail with path traversal error
		require.Error(t, err, "Extraction should fail for absolute paths")
		assert.Nil(t, contents, "Contents should be nil on security error")
		assert.Contains(t, err.Error(), "path traversal", "Error should indicate path traversal")
	})

	t.Run("PathTraversalDotDot", func(t *testing.T) {
		// Create archive with .. in path (security test)
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "malicious2.tar.gz")

		file, err := os.Create(archivePath)
		require.NoError(t, err)
		defer file.Close()

		gzw := gzip.NewWriter(file)
		defer gzw.Close()

		tw := tar.NewWriter(gzw)
		defer tw.Close()

		// Try to write outside destDir using ..
		maliciousPath := "../../../etc/passwd"
		header := &tar.Header{
			Name: maliciousPath,
			Mode: 0644,
			Size: int64(len("malicious content")),
		}
		require.NoError(t, tw.WriteHeader(header))
		_, err = tw.Write([]byte("malicious content"))
		require.NoError(t, err)

		tw.Close()
		gzw.Close()
		file.Close()

		// Attempt extraction
		destDir := t.TempDir()
		contents, err := storage.ExtractArchive(ctx, archivePath, destDir)

		// Should fail with path traversal error
		require.Error(t, err, "Extraction should fail for paths with ..")
		assert.Nil(t, contents, "Contents should be nil on security error")
		assert.Contains(t, err.Error(), "path traversal", "Error should indicate path traversal")
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		// Create large archive
		archivePath := createTestArchive(t, map[string]string{
			"project.tar.gz": "data",
		})

		// Create cancelled context
		ctx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		destDir := t.TempDir()
		contents, err := storage.ExtractArchive(ctx, archivePath, destDir)

		// Should fail with context error
		require.Error(t, err, "Extraction should fail with cancelled context")
		assert.Nil(t, contents, "Contents should be nil on context cancellation")
	})
}

// createTestArchive creates a test tar.gz archive with the given files.
// Returns the path to the created archive.
func createTestArchive(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test-archive.tar.gz")

	file, err := os.Create(archivePath)
	require.NoError(t, err, "Failed to create archive file")
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(header), "Failed to write tar header for "+name)
		_, err = tw.Write([]byte(content))
		require.NoError(t, err, "Failed to write tar content for "+name)
	}

	tw.Close()
	gzw.Close()
	file.Close()

	return archivePath
}
