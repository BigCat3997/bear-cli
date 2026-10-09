package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func CreateS3(ctx context.Context, bucketName string) {
	if err := CreateS3Bucket(ctx, bucketName); err != nil {
		panic(err)
	}
	fmt.Println("Bucket created:", bucketName)
}

// CreateS3Bucket creates a bucket and returns errors to callers.
func CreateS3Bucket(ctx context.Context, bucketName string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if bucketName == "" {
		return fmt.Errorf("bucket name is required")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucketName,
	})

	if err != nil {
		return fmt.Errorf("create S3 bucket: %w", err)
	}
	return nil
}

func awsString(value string) *string {
	return &value
}
