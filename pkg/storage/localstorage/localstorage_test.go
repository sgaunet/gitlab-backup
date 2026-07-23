package localstorage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/storage/localstorage"
	"github.com/stretchr/testify/require"
)

func TestSaveFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a temporary source file
	srcFile, err := os.CreateTemp(tempDir, "source")
	if err != nil {
		t.Fatalf("Failed to create temp source file: %v", err)
	}
	defer os.Remove(srcFile.Name())

	// Write some content to the source file
	content := []byte("Hello, World!")
	if _, err := srcFile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp source file: %v", err)
	}
	srcFile.Close()

	// Initialize LocalStorage
	storage := localstorage.NewLocalStorage(tempDir)

	// Define the destination filename
	dstFilename := "dstFile"

	// Call SaveFile
	err = storage.SaveFile(context.Background(), srcFile.Name(), dstFilename)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Verify the file was copied correctly
	dstFilePath := filepath.Join(tempDir, dstFilename)
	dstContent, err := os.ReadFile(dstFilePath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("File content mismatch. Expected %s, got %s", string(content), string(dstContent))
	}
}

func TestSaveFileWithEmptySource(t *testing.T) {
	tempDir := t.TempDir()
	// Initialize LocalStorage
	storage := localstorage.NewLocalStorage(tempDir)

	dstFilename := "dstFile"

	// Call SaveFile
	err := storage.SaveFile(context.Background(), "file-that-does-not-exist", dstFilename)
	require.Error(t, err)
}

func TestSaveFileWithWrontDestinationFolder(t *testing.T) {
	// Initialize LocalStorage
	storage := localstorage.NewLocalStorage("/tmp-does-not-exist")
	dstFilename := "dstFile"

	// Create temp file
	srcFile, err := os.CreateTemp(t.TempDir(), "source")
	require.NoError(t, err)
	defer os.Remove(srcFile.Name())

	// Call SaveFile
	err = storage.SaveFile(context.Background(), srcFile.Name(), dstFilename)
	require.Error(t, err)
}

// TestSaveFile_LargeFile copies a source larger than the copy buffer so the
// multi-iteration copy loop and the EOF-break path are exercised.
func TestSaveFile_LargeFile(t *testing.T) {
	tempDir := t.TempDir()

	// Well over the 32KB copy buffer to force several read/write iterations.
	content := make([]byte, 256*1024)
	for i := range content {
		content[i] = byte(i % 251)
	}

	srcPath := filepath.Join(tempDir, "large-source")
	require.NoError(t, os.WriteFile(srcPath, content, 0o600))

	storage := localstorage.NewLocalStorage(tempDir)
	dstFilename := "large-dst"
	require.NoError(t, storage.SaveFile(context.Background(), srcPath, dstFilename))

	got, err := os.ReadFile(filepath.Join(tempDir, dstFilename))
	require.NoError(t, err)
	require.True(t, bytes.Equal(content, got), "copied content should match source")
}

// TestSaveFile_ContextCancellation verifies that SaveFile respects context cancellation.
func TestSaveFile_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()

	// Create a temporary source file with some content
	srcFile, err := os.CreateTemp(tempDir, "source_large")
	require.NoError(t, err)
	defer os.Remove(srcFile.Name())

	// Write content to the source file
	content := make([]byte, 1024*100) // 100KB
	for i := range content {
		content[i] = byte(i % 256)
	}
	_, err = srcFile.Write(content)
	require.NoError(t, err)
	srcFile.Close()

	// Initialize LocalStorage
	storage := localstorage.NewLocalStorage(tempDir)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Attempt to save file with cancelled context
	dstFilename := "dstFile"
	err = storage.SaveFile(ctx, srcFile.Name(), dstFilename)

	// Should fail due to context cancellation
	require.Error(t, err, "Expected error due to context cancellation")
	require.ErrorIs(t, err, context.Canceled, "Error should be context.Canceled")

	// Verify the destination file was cleaned up
	dstFilePath := filepath.Join(tempDir, dstFilename)
	_, statErr := os.Stat(dstFilePath)
	require.True(t, os.IsNotExist(statErr), "Destination file should be cleaned up after cancellation")
}
