package s3storage_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/storage/s3storage"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// rustfsImage is the S3-compatible server used for integration tests.
const rustfsImage = "rustfs/rustfs:1.0.1"

// s3Endpoint is the URL of the shared RustFS instance started in TestMain.
var s3Endpoint string

// TestMain starts a single RustFS container shared by all S3 tests and benchmarks.
func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        rustfsImage,
		ExposedPorts: []string{"9000/tcp"},
		WaitingFor:   wait.ForHTTP("/health").WithPort("9000/tcp"),
		Env: map[string]string{
			"RUSTFS_ACCESS_KEY": "rustfsadmin",
			"RUSTFS_SECRET_KEY": "rustfsadmin",
		},
	}
	rustfs, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start RustFS container: %v\n", err)
		os.Exit(1)
	}

	endpoint, err := rustfs.Endpoint(ctx, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get RustFS endpoint: %v\n", err)
		_ = rustfs.Terminate(ctx)
		os.Exit(1)
	}
	s3Endpoint = "http://" + endpoint

	os.Setenv("AWS_ACCESS_KEY_ID", "rustfsadmin")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "rustfsadmin")

	code := m.Run()

	if err := rustfs.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to terminate RustFS container: %v\n", err)
	}
	os.Exit(code)
}

func TestS3Storage_SaveFile(t *testing.T) {
	ctx := context.Background()

	s3, err := s3storage.NewS3Storage(ctx, "us-east-1", s3Endpoint, "tests", "tests")
	if err != nil {
		t.Errorf("error: %v", err)
	}

	err = s3.CreateBucket(ctx)
	if err != nil {
		t.Errorf("error: %v", err)
	}

	err = s3.SaveFile(ctx, "../../../README.md", "README.md")
	if err != nil {
		t.Errorf("error: %v", err)
	}

}

// func TestS3StorageAWS_SaveFile(t *testing.T) {
// 	ctx := context.Background()
// 	os.Setenv("AWS_ACCESS_KEY_ID", "...")
// 	os.Setenv("AWS_SECRET_ACCESS_KEY", "...")

// 	s3, err := s3storage.NewS3Storage("eu-west-3", "https://s3.eu-west-3.amazonaws.com", "mybucket", "prefix")
// 	if err != nil {
// 		t.Errorf("error: %v", err)
// 	}
// 	err = s3.SaveFile(ctx, "/home/sylvain/GITHUB/PUBLIC/gitlab-backup/README.md", "README.md")
// 	if err != nil {
// 		t.Errorf("error: %v", err)
// 	}
// }

// TestNewS3Storage_ContextAccepted verifies that NewS3Storage accepts and propagates context.
// Note: This test verifies the context is passed through without causing errors.
// Actual cancellation behavior depends on AWS SDK network calls which may not occur
// during initialization if credentials are cached or loaded from local config.
func TestNewS3Storage_ContextAccepted(t *testing.T) {
	// Save current AWS env vars
	oldAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	oldSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	defer func() {
		if oldAccessKey != "" {
			os.Setenv("AWS_ACCESS_KEY_ID", oldAccessKey)
		} else {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
		}
		if oldSecretKey != "" {
			os.Setenv("AWS_SECRET_ACCESS_KEY", oldSecretKey)
		} else {
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		}
	}()

	// Set static credentials to avoid network calls during test
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*1000000000) // 5 seconds
	defer cancel()

	// Create S3Storage with context - should succeed with static credentials
	s3, err := s3storage.NewS3Storage(ctx, "us-east-1", "https://s3.amazonaws.com", "test-bucket", "test-path")

	// Should succeed - we're just verifying context is accepted
	if err != nil {
		t.Errorf("Expected NewS3Storage to succeed with valid context, got error: %v", err)
	}

	if s3 == nil {
		t.Error("Expected non-nil S3Storage instance")
	}
}

// TestS3Storage_SaveFile_SingleOpen verifies the seek-based approach works correctly.
// This test validates the fix for issue #301 - optimize double file open in S3 SaveFile.
// The implementation now opens the file once, calculates MD5, seeks back to start,
// and uploads, rather than opening the file twice.
func TestS3Storage_SaveFile_SingleOpen(t *testing.T) {
	ctx := context.Background()

	// Create S3Storage
	s3, err := s3storage.NewS3Storage(ctx, "us-east-1", s3Endpoint, "tests-singleopen", "tests-singleopen")
	if err != nil {
		t.Fatalf("Failed to create S3Storage: %v", err)
	}

	// Create bucket
	err = s3.CreateBucket(ctx)
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Create a temporary test file with known content
	tmpFile, err := os.CreateTemp("", "s3test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := []byte("This is test content for verifying single file open optimization (issue #301)")
	_, err = tmpFile.Write(testContent)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Upload using the optimized SaveFile method
	err = s3.SaveFile(ctx, tmpFile.Name(), "test-single-open.txt")
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Verify the file was uploaded successfully by downloading it back
	downloadPath := tmpFile.Name() + ".download"
	defer os.Remove(downloadPath)

	err = s3.GetFile(ctx, "test-single-open.txt", downloadPath)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	// Verify content matches
	downloadedContent, err := os.ReadFile(downloadPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloadedContent) != string(testContent) {
		t.Errorf("Content mismatch.\nExpected: %s\nGot: %s", testContent, downloadedContent)
	}
}

// BenchmarkSaveFile measures upload performance for the optimized single-open implementation.
// This benchmark helps verify that the fix for issue #301 provides performance benefits
// by eliminating the double file open overhead.
func BenchmarkSaveFile(b *testing.B) {
	ctx := context.Background()

	// Create S3Storage
	s3, err := s3storage.NewS3Storage(ctx, "us-east-1", s3Endpoint, "bench", "bench")
	if err != nil {
		b.Fatalf("Failed to create S3Storage: %v", err)
	}

	// Create bucket
	err = s3.CreateBucket(ctx)
	if err != nil {
		b.Fatalf("Failed to create bucket: %v", err)
	}

	// Create a test file of realistic size (10MB)
	tmpFile, err := os.CreateTemp("", "s3bench-*.dat")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write 10MB of data
	data := make([]byte, 10*1024*1024) // 10MB
	for i := range data {
		data[i] = byte(i % 256)
	}
	_, err = tmpFile.Write(data)
	if err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("bench-%d.dat", i)
		err = s3.SaveFile(ctx, tmpFile.Name(), filename)
		if err != nil {
			b.Fatalf("SaveFile failed: %v", err)
		}
	}
}
