package localstorage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("/tmp", "localstorage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	storage := NewLocalStorage(tempDir)

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
	tempDir, err := os.MkdirTemp("/tmp", "localstorage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	// Initialize LocalStorage
	storage := NewLocalStorage(tempDir)

	dstFilename := "dstFile"

	// Call SaveFile
	err = storage.SaveFile(context.Background(), "file-that-does-not-exist", dstFilename)
	require.Error(t, err)
}

func TestSaveFileWithWrontDestinationFolder(t *testing.T) {
	// Initialize LocalStorage
	storage := NewLocalStorage("/tmp-does-not-exist")
	dstFilename := "dstFile"

	// Create temp file in /tmp
	srcFile, err := os.CreateTemp("/tmp", "source")
	require.NoError(t, err)
	defer os.Remove(srcFile.Name())

	// Call SaveFile
	err = storage.SaveFile(context.Background(), srcFile.Name(), dstFilename)
	require.Error(t, err)
}
