package s3storage

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"io"
	"os"

	// "github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	// s3Client *minio.Client
	s3Client *s3.Client
	endpoint string
	region   string
	bucket   string
	path     string
}

// InitClient initializes the s3 client
func (s *S3Storage) InitClient() (err error) {
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		staticResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               s.endpoint, // or where ever you ran minio
				SigningRegion:     s.region,
				HostnameImmutable: true,
			}, nil
		})

		cfg := aws.Config{
			Region:           s.region,
			Credentials:      credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
			EndpointResolver: staticResolver,
		}
		s.s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s.endpoint)
		})
		return
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
		return err
	}
	s.s3Client = s3.NewFromConfig(cfg)
	return nil
}

// NewS3Storage creates a new S3Storage
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

// CreateBucket creates the bucket
func (s *S3Storage) CreateBucket(ctx context.Context) error {
	// return s.s3Client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region})
	_, err := s.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	return err
}

// SaveFile saves the file in s3
func (s *S3Storage) SaveFile(ctx context.Context, archiveFilePath string, dstFilename string) error {
	f, err := os.Open(archiveFilePath)
	if err != nil {
		return err
	}
	// calculate md5 of f
	hash := md5.New()
	_, err = io.Copy(hash, f)
	if err != nil {
		return err
	}
	f.Close()
	fsrc, err := os.Open(archiveFilePath)
	if err != nil {
		return err
	}
	defer fsrc.Close()

	md5b64 := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	_, err = s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(s.path + "/" + dstFilename),
		Body:       fsrc,
		ContentMD5: aws.String(md5b64),
	})
	return err
}
