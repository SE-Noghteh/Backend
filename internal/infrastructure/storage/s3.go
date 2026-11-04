package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"zedlancer/backend/bootstrap"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/valueobjects"
)

type s3API interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type S3Storage struct {
	client    s3API
	presigner *s3.PresignClient
	bucket    string
}

func NewS3Storage() (*S3Storage, error) {
	settings := bootstrap.Run().Env.S3
	// Missing storage configuration must not disable existing job seeker APIs.
	if settings.Bucket == "" {
		return &S3Storage{}, nil
	}
	if settings.Endpoint != "" {
		parsed, err := url.Parse(settings.Endpoint)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return nil, fmt.Errorf("S3_ENDPOINT must be an HTTP or HTTPS URL")
		}
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(settings.Region), config.WithRetryMaxAttempts(2), config.WithHTTPClient(&httpClient))
	if err != nil {
		return nil, fmt.Errorf("load S3 configuration: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.UsePathStyle = settings.UsePathStyle
		if settings.Endpoint != "" {
			options.BaseEndpoint = aws.String(settings.Endpoint)
		}
	})
	return &S3Storage{client: client, presigner: s3.NewPresignClient(client), bucket: settings.Bucket}, nil
}

func (s *S3Storage) Put(ctx context.Context, key string, body io.ReadSeeker, contentType string, size int64) (*valueobjects.StoredObject, error) {
	if s.client == nil {
		return nil, domainerrors.ErrStorageUnavailable
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: body, ContentType: aws.String(contentType), ContentLength: aws.Int64(size)})
	if err != nil {
		return nil, fmt.Errorf("upload S3 object: %w", err)
	}
	return &valueobjects.StoredObject{Key: key, ContentType: contentType, Size: size}, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if s.client == nil {
		return domainerrors.ErrStorageUnavailable
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}

func (s *S3Storage) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.presigner == nil {
		return "", domainerrors.ErrStorageUnavailable
	}
	if ttl <= 0 || ttl > 15*time.Minute {
		return "", fmt.Errorf("invalid download URL lifetime")
	}
	result, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), ResponseCacheControl: aws.String("no-store"), ResponseContentDisposition: aws.String("attachment")}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}
