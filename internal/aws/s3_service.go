package aws

import (
	"context"
	"time"
)

// S3Service interface defines AWS S3 operations
type S3Service interface {
	UploadFile(ctx context.Context, key string, filePath string, metadata map[string]string) error
	GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
	DeleteObject(ctx context.Context, key string) error
	HeadObject(ctx context.Context, key string) error
}

// S3ServiceImpl implements S3Service using AWS SDK v2
type S3ServiceImpl struct {
	// Will be implemented in task 4
}

// NewS3Service creates a new S3 service instance
func NewS3Service(region, bucket string) (*S3ServiceImpl, error) {
	// Will be implemented in task 4
	return &S3ServiceImpl{}, nil
}