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

	s3, err := s3storage.NewS3Storage("us-east-1", fmt.Sprintf("http://%s", endpoint), "tests", "tests")
	if err != nil {
		t.Errorf(err.Error())
	}

	err = s3.CreateBucket(ctx)
	if err != nil {
		t.Errorf(err.Error())
	}

	err = s3.SaveFile(ctx, "/home/sylvain/GITHUB/PUBLIC/gitlab-backup/README.md", "README.md")
	if err != nil {
		t.Errorf(err.Error())
	}

}

// func TestS3StorageAWS_SaveFile(t *testing.T) {
// 	ctx := context.Background()
// 	os.Setenv("AWS_ACCESS_KEY_ID", "...")
// 	os.Setenv("AWS_SECRET_ACCESS_KEY", "...")

// 	s3, err := s3storage.NewS3Storage("eu-west-3", "https://s3.eu-west-3.amazonaws.com", "mybucket", "prefix")
// 	if err != nil {
// 		t.Errorf(err.Error())
// 	}
// 	err = s3.SaveFile(ctx, "/home/sylvain/GITHUB/PUBLIC/gitlab-backup/README.md", "README.md")
// 	if err != nil {
// 		t.Errorf(err.Error())
// 	}
// }
