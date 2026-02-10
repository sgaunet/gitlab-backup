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

func TestS3Storage_SaveFile(t *testing.T) {
	ctx := context.Background()
	// req := testcontainers.ContainerRequest{
	//     Image:        "redis:latest",
	//     ExposedPorts: []string{"6379/tcp"},
	//     WaitingFor:   wait.ForLog("Ready to accept connections"),
	// }
	// redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
	//     ContainerRequest: req,
	//     Started:          true,
	// })
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:RELEASE.2023-04-13T03-08-07Z.fips",
		ExposedPorts: []string{"9000/tcp"},
		// ExposedPorts: []string{"9000/tcp", "8080/tcp"},
		WaitingFor: wait.ForLog("Console: http://0.0.0.0:8080"),
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadminn",
			"MINIO_ROOT_PASSWORD": "minioadminn",
			"MINIO_BUCKET":        "tests",
		},
		Cmd: []string{"server", "/export", "--console-address", "0.0.0.0:8080"},
	}
	minio, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := minio.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	endpoint, err := minio.Endpoint(ctx, "")
	if err != nil {
		t.Error(err)
	}

	os.Setenv("AWS_ACCESS_KEY_ID", "minioadminn")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "minioadminn")

	s3, err := s3storage.NewS3Storage(ctx, "us-east-1", fmt.Sprintf("http://%s", endpoint), "tests", "tests")
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
