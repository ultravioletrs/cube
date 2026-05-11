package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3Store struct {
	client *minio.Client
	bucket string
}

func newS3Store(cfg Config) (*s3Store, error) {
	endpoint := strings.TrimSpace(cfg.S3Endpoint)
	bucket := strings.TrimSpace(cfg.S3Bucket)
	accessKey := strings.TrimSpace(cfg.S3AccessKeyID)
	secretKey := strings.TrimSpace(cfg.S3SecretAccessKey)

	if endpoint == "" {
		return nil, fmt.Errorf("s3 endpoint is required")
	}
	if bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("s3 access and secret keys are required")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       cfg.S3UseSSL,
		Region:       cfg.S3Region,
		BucketLookup: bucketLookup(cfg.S3PathStyle),
	})
	if err != nil {
		return nil, err
	}

	store := &s3Store{client: client, bucket: bucket}
	if cfg.S3EnsureBucket {
		if err := store.ensureBucket(context.Background()); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *s3Store) Put(ctx context.Context, key, contentType string, size int64, body io.Reader) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *s3Store) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *s3Store) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func bucketLookup(forcePathStyle bool) minio.BucketLookupType {
	if forcePathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}
