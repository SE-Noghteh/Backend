// storage-init creates the configured private S3 bucket for local development.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"zedlancer/backend/bootstrap"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	settings := bootstrap.NewEnv().S3
	if settings.Bucket == "" || settings.Endpoint == "" {
		return fmt.Errorf("set S3_BUCKET and S3_ENDPOINT for local storage first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(settings.Region))
	if err != nil {
		return err
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(settings.Endpoint)
		options.UsePathStyle = settings.UsePathStyle
	})
	input := &s3.CreateBucketInput{Bucket: aws.String(settings.Bucket)}
	if settings.Region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{LocationConstraint: types.BucketLocationConstraint(settings.Region)}
	}
	_, err = client.CreateBucket(ctx, input)
	var owned *types.BucketAlreadyOwnedByYou
	if err != nil && !errors.As(err, &owned) {
		return fmt.Errorf("create private bucket: %w", err)
	}
	if _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: input.Bucket}); err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	log.Printf("Private bucket %q is ready at %s", settings.Bucket, settings.Endpoint)
	return nil
}
