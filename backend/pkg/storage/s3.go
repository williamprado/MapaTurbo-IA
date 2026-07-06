package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
	"mapaturbo-ia/pkg/logger"
)

type S3Client struct {
	client *minio.Client
	bucket string
}

var Client *S3Client

func InitS3(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO: %w", err)
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Log.Info("Created MinIO bucket successfully", zap.String("bucket", bucket))
	} else {
		logger.Log.Info("MinIO bucket already exists", zap.String("bucket", bucket))
	}

	Client = &S3Client{
		client: minioClient,
		bucket: bucket,
	}

	return Client, nil
}

func (s *S3Client) UploadFile(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *S3Client) GetFileURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, key, expires, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}
