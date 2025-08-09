package aws

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// UploadProgress represents the progress of a file upload
type UploadProgress struct {
	BytesUploaded int64   `json:"bytes_uploaded"`
	TotalBytes    int64   `json:"total_bytes"`
	Percentage    float64 `json:"percentage"`
}

// S3Service defines the interface for S3 operations
type S3Service interface {
	// UploadFile uploads a file to S3 with optional progress tracking
	UploadFile(ctx context.Context, key string, filePath string, metadata map[string]string, progressCh chan<- UploadProgress) error
	
	// GeneratePresignedURL generates a presigned URL for downloading a file
	GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
	
	// DeleteObject deletes an object from S3
	DeleteObject(ctx context.Context, key string) error
	
	// HeadObject retrieves metadata about an object without downloading it
	HeadObject(ctx context.Context, key string) (*s3.HeadObjectOutput, error)
	
	// TestConnection tests the S3 connection by listing bucket contents
	TestConnection(ctx context.Context) error
}

// S3ServiceImpl implements S3Service using AWS SDK v2
type S3ServiceImpl struct {
	client    *s3.Client
	presigner *s3.PresignClient
	uploader  *manager.Uploader
	bucket    string
	region    string
}

// NewS3Service creates a new S3Service instance
func NewS3Service(credProvider CredentialProvider, bucket string) (*S3ServiceImpl, error) {
	if bucket == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	ctx := context.Background()
	
	// Get credentials from the provider
	creds, err := credProvider.GetCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	// Get region from the provider
	region, err := credProvider.GetRegion()
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS region: %w", err)
	}

	// Create AWS config with credentials and region
	cfg := aws.Config{
		Credentials: credentials.StaticCredentialsProvider{
			Value: creds,
		},
		Region: region,
		// AWS SDK v2 has built-in retry logic with exponential backoff
		// Default retry mode is "standard" with 3 attempts
		RetryMode: aws.RetryModeStandard,
		RetryMaxAttempts: 3,
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)
	
	// Create presigner for generating presigned URLs
	presigner := s3.NewPresignClient(client)
	
	// Create uploader with custom configuration for chunking
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		// Set chunk size to 256MB for files >5MB (as per requirement)
		u.PartSize = 256 * 1024 * 1024 // 256MB
		// Set concurrency to 1 for sequential uploads (simplified approach)
		u.Concurrency = 1
	})

	return &S3ServiceImpl{
		client:    client,
		presigner: presigner,
		uploader:  uploader,
		bucket:    bucket,
		region:    region,
	}, nil
}

// UploadFile uploads a file to S3 with progress tracking and chunking for large files
func (s *S3ServiceImpl) UploadFile(ctx context.Context, key string, filePath string, metadata map[string]string, progressCh chan<- UploadProgress) error {
	if key == "" {
		return fmt.Errorf("S3 object key cannot be empty")
	}
	
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer file.Close()

	// Get file info for size and content type
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for '%s': %w", filePath, err)
	}

	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return fmt.Errorf("file '%s' is empty", filePath)
	}

	// Determine content type based on file extension
	contentType := getContentType(filePath)

	// Create progress reader if progress channel is provided
	var reader io.Reader = file
	if progressCh != nil {
		reader = &progressReader{
			reader:     file,
			totalBytes: fileSize,
			progressCh: progressCh,
		}
	}

	// Prepare metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}
	
	// Add upload timestamp to metadata
	metadata["upload-timestamp"] = time.Now().UTC().Format(time.RFC3339)
	metadata["original-filename"] = filepath.Base(filePath)

	// Create upload input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
		Metadata:    metadata,
		// Enable server-side encryption
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	}

	// Use uploader for chunking (automatically handles multipart uploads for files >5MB)
	_, err = s.uploader.Upload(ctx, input)
	if err != nil {
		return s.handleS3Error("upload file", err)
	}

	// Send final progress update
	if progressCh != nil {
		select {
		case progressCh <- UploadProgress{
			BytesUploaded: fileSize,
			TotalBytes:    fileSize,
			Percentage:    100.0,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// GeneratePresignedURL generates a presigned URL for downloading a file
func (s *S3ServiceImpl) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if key == "" {
		return "", fmt.Errorf("S3 object key cannot be empty")
	}

	if expiration <= 0 {
		return "", fmt.Errorf("expiration duration must be positive")
	}

	// Limit maximum expiration to 7 days for security (as per requirement 8.5)
	maxExpiration := 7 * 24 * time.Hour
	if expiration > maxExpiration {
		expiration = maxExpiration
	}

	// Create presigned request
	request, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", s.handleS3Error("generate presigned URL", err)
	}

	return request.URL, nil
}

// DeleteObject deletes an object from S3
func (s *S3ServiceImpl) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("S3 object key cannot be empty")
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return s.handleS3Error("delete object", err)
	}

	return nil
}

// HeadObject retrieves metadata about an object without downloading it
func (s *S3ServiceImpl) HeadObject(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	if key == "" {
		return nil, fmt.Errorf("S3 object key cannot be empty")
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	output, err := s.client.HeadObject(ctx, input)
	if err != nil {
		return nil, s.handleS3Error("get object metadata", err)
	}

	return output, nil
}

// TestConnection tests the S3 connection by attempting to list bucket contents
func (s *S3ServiceImpl) TestConnection(ctx context.Context) error {
	// Try to list objects in the bucket (limit to 1 to minimize cost)
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		MaxKeys: aws.Int32(1),
	}

	_, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return s.handleS3Error("test connection", err)
	}

	return nil
}

// handleS3Error converts AWS S3 errors to user-friendly error messages
func (s *S3ServiceImpl) handleS3Error(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Handle specific AWS error types by checking error strings
	// Note: AWS SDK v2 error handling is different from v1

	// Handle context errors by checking the error message
	errStr := err.Error()
	if errStr == "context deadline exceeded" {
		return fmt.Errorf("operation timed out while trying to %s. Please check your internet connection and try again", operation)
	}
	if errStr == "context canceled" {
		return fmt.Errorf("operation was canceled while trying to %s", operation)
	}

	// Check for common AWS error patterns in the error message
	if strings.Contains(errStr, "AccessDenied") {
		return fmt.Errorf("access denied to S3 bucket '%s'. Please check your AWS credentials and IAM permissions", s.bucket)
	}
	if strings.Contains(errStr, "NoSuchBucket") {
		return fmt.Errorf("S3 bucket '%s' does not exist or you don't have access to it. Please check your bucket name and permissions", s.bucket)
	}
	if strings.Contains(errStr, "NoSuchKey") {
		return fmt.Errorf("the requested file was not found in S3. It may have been deleted or expired")
	}

	// Generic error with operation context
	return fmt.Errorf("failed to %s: %w", operation, err)
}

// getContentType determines the content type based on file extension
func getContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".zip":
		return "application/zip"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	default:
		return "application/octet-stream"
	}
}

// progressReader wraps an io.Reader to provide upload progress updates
type progressReader struct {
	reader       io.Reader
	totalBytes   int64
	bytesRead    int64
	progressCh   chan<- UploadProgress
}

// Read implements io.Reader and sends progress updates
func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.bytesRead += int64(n)
		percentage := float64(pr.bytesRead) / float64(pr.totalBytes) * 100.0
		
		// Send progress update (non-blocking)
		select {
		case pr.progressCh <- UploadProgress{
			BytesUploaded: pr.bytesRead,
			TotalBytes:    pr.totalBytes,
			Percentage:    percentage,
		}:
		default:
			// Channel is full, skip this update
		}
	}
	return n, err
}