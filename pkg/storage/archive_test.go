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

	t.Run("ReturnsArchivePath", func(t *testing.T) {
		// Create test archive
		archivePath := createTestArchive(t, map[string]string{
			"VERSION": "0.1",
			"tree/":   "",
		})

		destDir := t.TempDir()

		// ExtractArchive should return the archive path as-is (no extraction)
		contents, err := storage.ExtractArchive(ctx, archivePath, destDir)

		// Assertions
		require.NoError(t, err, "Should always succeed")
		require.NotNil(t, contents, "Contents should not be nil")
		assert.Equal(t, archivePath, contents.ProjectExportPath, "Should return archive path as-is")
		assert.Equal(t, destDir, contents.ExtractionDir, "Extraction dir should match")
	})
}

func TestExtractArchive_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	destDir := t.TempDir()

	testCases := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError string
	}{
		{
			name: "non-existent file",
			setupFile: func(t *testing.T) string {
				return "/nonexistent/archive.tar.gz"
			},
			expectError: "archive not found",
		},
		{
			name: "directory instead of file",
			setupFile: func(t *testing.T) string {
				dir := t.TempDir()
				return dir
			},
			expectError: "is a directory",
		},
		{
			name: "empty file",
			setupFile: func(t *testing.T) string {
				emptyFile := filepath.Join(t.TempDir(), "empty.tar.gz")
				err := os.WriteFile(emptyFile, []byte{}, 0644)
				require.NoError(t, err)
				return emptyFile
			},
			expectError: "archive is empty",
		},
		{
			name: "invalid gzip format",
			setupFile: func(t *testing.T) string {
				invalidFile := filepath.Join(t.TempDir(), "invalid.tar.gz")
				err := os.WriteFile(invalidFile, []byte("not a gzip file"), 0644)
				require.NoError(t, err)
				return invalidFile
			},
			expectError: "invalid gzip format",
		},
		{
			name: "corrupted tar header",
			setupFile: func(t *testing.T) string {
				// Create a valid gzip file with invalid tar content
				corruptedFile := filepath.Join(t.TempDir(), "corrupted.tar.gz")
				f, err := os.Create(corruptedFile)
				require.NoError(t, err)
				defer f.Close()

				gzw := gzip.NewWriter(f)
				_, err = gzw.Write([]byte("invalid tar content"))
				require.NoError(t, err)
				err = gzw.Close()
				require.NoError(t, err)

				return corruptedFile
			},
			expectError: "invalid tar format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			archivePath := tc.setupFile(t)

			_, err := storage.ExtractArchive(ctx, archivePath, destDir)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
			assert.Contains(t, err.Error(), "invalid archive format")
		})
	}
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
