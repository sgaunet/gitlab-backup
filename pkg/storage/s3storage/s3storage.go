package s3storage

import (
	"context"
	"io"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sgaunet/dsn/v2/pkg/dsn"
)

type S3Storage struct {
	s3Client *minio.Client
	endpoint string
	region   string
	bucket   string
	path     string
}

func NewS3Storage(region string, endpoint string, bucket string, path string) (*S3Storage, error) {
	var err error
	var secure bool

	d, err := dsn.New(endpoint)
	if err != nil {
		return nil, err
	}
	endpointWithoutScheme := d.GetHost() + ":" + d.GetPort("443")
	if d.GetScheme() == "https" {
		secure = true
	}

	s := &S3Storage{
		endpoint: endpointWithoutScheme,
		region:   region,
		bucket:   bucket,
		path:     path,
	}

	s.s3Client, err = minio.New(s.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		Secure: secure,
		Region: s.region,
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *S3Storage) SaveFile(ctx context.Context, src io.Reader, dstFilename string, fileSize int64) error {
	_, err := s.s3Client.PutObject(context.TODO(), s.bucket, s.path+"/"+dstFilename, src, fileSize, minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	return err
}
