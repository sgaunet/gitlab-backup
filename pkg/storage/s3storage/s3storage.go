// Package s3storage provides AWS S3 storage implementation.
package s3storage

import (
	"context"
	"crypto/md5" //nolint:gosec // G501: MD5 required for S3 Content-MD5 header
	"encoding/base64"
	"fmt"
	"io"
	"os"

	// "github.com/minio/minio-go/v7/pkg/credentials".

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage implements storage interface for AWS S3.
type S3Storage struct {
	// s3Client *minio.Client
	s3Client *s3.Client
	endpoint string
	region   string
	bucket   string
	path     string
}

// NewS3Storage creates a new S3Storage.
func NewS3Storage(region string, endpoint string, bucket string, path string) (*S3Storage, error) {
	var err error

	s := &S3Storage{
		endpoint: endpoint,
		region:   region,
		bucket:   bucket,
		path:     path,
	}
	err = s.InitClient()
	if err != nil {
		return nil, err
	}
	return s, nil
}

// InitClient initializes the s3 client.
func (s *S3Storage) InitClient() error {
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		//nolint:staticcheck // SA1019: Using deprecated AWS endpoint resolver for compatibility
		staticResolver := aws.EndpointResolverFunc(func(_, _ string) (aws.Endpoint, error) {
			return aws.Endpoint{ //nolint:staticcheck // SA1019: aws.Endpoint is deprecated but still needed for custom endpoints
				PartitionID:       "aws",
				URL:               s.endpoint, // or where ever you ran minio
				SigningRegion:     s.region,
				HostnameImmutable: true,
			}, nil
		})

		cfg := aws.Config{
			Region:           s.region,
			Credentials: credentials.NewStaticCredentialsProvider(
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"),
				"",
			),
			EndpointResolver: staticResolver,
		}
		s.s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s.endpoint)
		})
		return nil
	}

	// if s.cfg.SsoAwsProfile != "" {
	// 	fmt.Println("Try to use SSO profile")
	// 	cfg, err = config.LoadDefaultConfig(
	// 		context.TODO(),
	// 		config.WithSharedConfigProfile(s.cfg.SsoAwsProfile),
	// 	)
	// 	return
	// }

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	s.s3Client = s3.NewFromConfig(cfg)
	return nil
}


// CreateBucket creates the bucket.
func (s *S3Storage) CreateBucket(ctx context.Context) error {
	// return s.s3Client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region})
	_, err := s.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create S3 bucket %s: %w", s.bucket, err)
	}
	return nil
}

// SaveFile saves the file in s3.
func (s *S3Storage) SaveFile(ctx context.Context, archiveFilePath string, dstFilename string) error {
	f, err := os.Open(archiveFilePath) //nolint:gosec // G304: File access is intentional for backup functionality
	if err != nil {
		return fmt.Errorf("failed to open archive file %s: %w", archiveFilePath, err)
	}
	// calculate md5 of f
	hash := md5.New() //nolint:gosec // G401: MD5 required for S3 Content-MD5 header
	_, err = io.Copy(hash, f)
	if err != nil {
		return fmt.Errorf("failed to calculate MD5 hash: %w", err)
	}
	_ = f.Close()
	fsrc, err := os.Open(archiveFilePath) //nolint:gosec // G304: File access is intentional for backup functionality
	if err != nil {
		return fmt.Errorf("failed to reopen archive file %s: %w", archiveFilePath, err)
	}
	defer func() { _ = fsrc.Close() }()

	md5b64 := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(s.path + "/" + dstFilename),
		Body:       fsrc,
		ContentMD5: aws.String(md5b64),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file %s to S3: %w", dstFilename, err)
	}
	return nil
}
