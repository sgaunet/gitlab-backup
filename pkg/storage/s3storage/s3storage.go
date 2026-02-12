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
// The context is used for AWS SDK configuration loading and may respect timeout/cancellation.
func NewS3Storage(ctx context.Context, region string, endpoint string, bucket string, path string) (*S3Storage, error) {
	var err error

	s := &S3Storage{
		endpoint: endpoint,
		region:   region,
		bucket:   bucket,
		path:     path,
	}
	err = s.initClient(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
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
func (s *S3Storage) SaveFile(ctx context.Context, archiveFilePath string, dstFilename string) (err error) {
	// Open file once
	f, openErr := os.Open(archiveFilePath) //nolint:gosec // G304: File access is intentional for backup functionality
	if openErr != nil {
		return fmt.Errorf("failed to open archive file %s: %w", archiveFilePath, openErr)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close archive file %s: %w", archiveFilePath, closeErr)
		}
	}()

	// First pass: calculate MD5
	hash := md5.New() //nolint:gosec // G401: MD5 required for S3 Content-MD5 header
	_, err = io.Copy(hash, f)
	if err != nil {
		return fmt.Errorf("failed to calculate MD5 hash: %w", err)
	}

	// Seek back to beginning for upload
	_, err = f.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to file start: %w", err)
	}

	// Second pass: upload with ContentMD5
	md5b64 := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	fullKey := s.path + "/" + dstFilename
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(fullKey),
		Body:       f,
		ContentMD5: aws.String(md5b64),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file %s to S3 bucket %s (key: %s): %w",
			dstFilename, s.bucket, fullKey, err)
	}

	return nil
}

// GetFile downloads a file from S3 and saves it to the specified local path.
func (s *S3Storage) GetFile(ctx context.Context, key string, localPath string) (err error) {
	// Create local file
	//nolint:gosec // G304: File creation is intentional for restore functionality
	outFile, createErr := os.Create(localPath)
	if createErr != nil {
		return fmt.Errorf("failed to create local file %s: %w", localPath, createErr)
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close local file %s: %w", localPath, closeErr)
		}
	}()

	// Construct full S3 key
	fullKey := s.path + "/" + key
	if s.path == "" {
		fullKey = key
	}

	// Download from S3
	result, getErr := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})
	if getErr != nil {
		return fmt.Errorf("failed to download file %s from S3 bucket %s (key: %s): %w",
			key, s.bucket, fullKey, getErr)
	}
	defer func() {
		if closeErr := result.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close S3 response body for key %s: %w", fullKey, closeErr)
		}
	}()

	// Copy to local file
	_, err = io.Copy(outFile, result.Body)
	if err != nil {
		return fmt.Errorf("failed to write downloaded file to %s: %w", localPath, err)
	}

	return nil
}

// initClient initializes the s3 client with context support.
func (s *S3Storage) initClient(ctx context.Context) error {
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

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(s.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	s.s3Client = s3.NewFromConfig(cfg)
	return nil
}
