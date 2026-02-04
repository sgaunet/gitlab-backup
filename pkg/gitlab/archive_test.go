package gitlab

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCompositeArchive_Success(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	gitlabExportPath := filepath.Join(tmpDir, "project-123-gitlab.tar.gz")
	labelsPath := filepath.Join(tmpDir, "labels-123.json")
	issuesPath := filepath.Join(tmpDir, "issues-123.json")
	outputPath := filepath.Join(tmpDir, "project-123.tar.gz")

	// Write test content to files
	err = os.WriteFile(gitlabExportPath, []byte("gitlab export content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(labelsPath, []byte(`[{"id": 1, "name": "bug"}]`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(issuesPath, []byte(`[{"id": 1, "title": "Issue 1"}]`), 0644)
	require.NoError(t, err)

	// Create composite archive
	err = CreateCompositeArchive(gitlabExportPath, labelsPath, issuesPath, outputPath)
	require.NoError(t, err)

	// Verify archive was created
	assert.FileExists(t, outputPath)

	// Verify archive contents
	fileContents := extractArchiveContents(t, outputPath)

	assert.Contains(t, fileContents, "project-123-gitlab.tar.gz")
	assert.Contains(t, fileContents, "labels-123.json")
	assert.Contains(t, fileContents, "issues-123.json")

	assert.Equal(t, "gitlab export content", fileContents["project-123-gitlab.tar.gz"])
	assert.Equal(t, `[{"id": 1, "name": "bug"}]`, fileContents["labels-123.json"])
	assert.Equal(t, `[{"id": 1, "title": "Issue 1"}]`, fileContents["issues-123.json"])
}

func TestCreateCompositeArchive_OnlyGitLabExport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitlabExportPath := filepath.Join(tmpDir, "project-123-gitlab.tar.gz")
	outputPath := filepath.Join(tmpDir, "project-123.tar.gz")

	err = os.WriteFile(gitlabExportPath, []byte("gitlab export content"), 0644)
	require.NoError(t, err)

	// Create composite archive with empty labels and issues paths
	err = CreateCompositeArchive(gitlabExportPath, "", "", outputPath)
	require.NoError(t, err)

	assert.FileExists(t, outputPath)

	fileContents := extractArchiveContents(t, outputPath)
	assert.Contains(t, fileContents, "project-123-gitlab.tar.gz")
	assert.NotContains(t, fileContents, "labels-123.json")
	assert.NotContains(t, fileContents, "issues-123.json")
}

func TestCreateCompositeArchive_WithLabelsOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitlabExportPath := filepath.Join(tmpDir, "project-123-gitlab.tar.gz")
	labelsPath := filepath.Join(tmpDir, "labels-123.json")
	outputPath := filepath.Join(tmpDir, "project-123.tar.gz")

	err = os.WriteFile(gitlabExportPath, []byte("gitlab export content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(labelsPath, []byte(`[{"id": 1, "name": "bug"}]`), 0644)
	require.NoError(t, err)

	err = CreateCompositeArchive(gitlabExportPath, labelsPath, "", outputPath)
	require.NoError(t, err)

	fileContents := extractArchiveContents(t, outputPath)
	assert.Contains(t, fileContents, "project-123-gitlab.tar.gz")
	assert.Contains(t, fileContents, "labels-123.json")
	assert.NotContains(t, fileContents, "issues-123.json")
}

func TestCreateCompositeArchive_WithIssuesOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitlabExportPath := filepath.Join(tmpDir, "project-123-gitlab.tar.gz")
	issuesPath := filepath.Join(tmpDir, "issues-123.json")
	outputPath := filepath.Join(tmpDir, "project-123.tar.gz")

	err = os.WriteFile(gitlabExportPath, []byte("gitlab export content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(issuesPath, []byte(`[{"id": 1, "title": "Issue 1"}]`), 0644)
	require.NoError(t, err)

	err = CreateCompositeArchive(gitlabExportPath, "", issuesPath, outputPath)
	require.NoError(t, err)

	fileContents := extractArchiveContents(t, outputPath)
	assert.Contains(t, fileContents, "project-123-gitlab.tar.gz")
	assert.NotContains(t, fileContents, "labels-123.json")
	assert.Contains(t, fileContents, "issues-123.json")
}

func TestCreateCompositeArchive_InvalidGitLabExportPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "project-123.tar.gz")

	err = CreateCompositeArchive("/nonexistent/path.tar.gz", "", "", outputPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add GitLab export to archive")
}

func TestCreateCompositeArchive_InvalidOutputPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitlabExportPath := filepath.Join(tmpDir, "project-123-gitlab.tar.gz")
	err = os.WriteFile(gitlabExportPath, []byte("content"), 0644)
	require.NoError(t, err)

	err = CreateCompositeArchive(gitlabExportPath, "", "", "/invalid/path/output.tar.gz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output file")
}

func TestAddFileToArchive_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFilePath := filepath.Join(tmpDir, "test.txt")
	testContent := "test content"
	err = os.WriteFile(testFilePath, []byte(testContent), 0644)
	require.NoError(t, err)

	// Create archive file
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	outFile, err := os.Create(archivePath)
	require.NoError(t, err)
	defer outFile.Close()

	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Add file to archive
	err = addFileToArchive(tarWriter, testFilePath, "test.txt")
	require.NoError(t, err)

	// Close writers to flush
	tarWriter.Close()
	gzipWriter.Close()
	outFile.Close()

	// Verify archive contents
	fileContents := extractArchiveContents(t, archivePath)
	assert.Contains(t, fileContents, "test.txt")
	assert.Equal(t, testContent, fileContents["test.txt"])
}

func TestAddFileToArchive_NonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	outFile, err := os.Create(archivePath)
	require.NoError(t, err)
	defer outFile.Close()

	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = addFileToArchive(tarWriter, "/nonexistent/file.txt", "file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

// Helper function to extract and return archive contents
func extractArchiveContents(t *testing.T, archivePath string) map[string]string {
	t.Helper()

	file, err := os.Open(archivePath)
	require.NoError(t, err)
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	contents := make(map[string]string)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		data, err := io.ReadAll(tarReader)
		require.NoError(t, err)

		contents[header.Name] = string(data)
	}

	return contents
}
