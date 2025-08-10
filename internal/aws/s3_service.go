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

	"file-sharing-app/pkg/errors"
	"file-sharing-app/pkg/logger"
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
	logger    *logger.Logger
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
		logger:    logger.NewWithComponent("s3_service"),
	}, nil
}

// UploadFile uploads a file to S3 with progress tracking and chunking for large files
func (s *S3ServiceImpl) UploadFile(ctx context.Context, key string, filePath string, metadata map[string]string, progressCh chan<- UploadProgress) error {
	return s.logger.LogOperation("upload_file", func() error {
		if key == "" {
			return errors.NewAppError(errors.ErrInvalidInput, "S3 object key cannot be empty", nil)
		}
		
		if filePath == "" {
			return errors.NewAppError(errors.ErrInvalidInput, "file path cannot be empty", nil)
		}

		s.logger.InfoWithFields("Starting file upload", map[string]interface{}{
			"s3_key":    key,
			"file_path": filepath.Base(filePath), // Only log filename for security
			"bucket":    s.bucket,
		})

		// Open the file
		file, err := os.Open(filePath)
		if err != nil {
			return errors.WrapError(err, errors.ErrFileNotFound, "failed to open file for upload")
		}
		defer file.Close()

		// Get file info for size and content type
		fileInfo, err := file.Stat()
		if err != nil {
			return errors.WrapError(err, errors.ErrInvalidFilePath, "failed to get file information")
		}

		fileSize := fileInfo.Size()
		if fileSize == 0 {
			return errors.NewAppError(errors.ErrFileEmpty, "file is empty", nil)
		}

		s.logger.InfoWithFields("File validated for upload", map[string]interface{}{
			"file_size_bytes": fileSize,
			"content_type":    getContentType(filePath),
		})

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
			s.logger.ErrorWithFields("Upload failed", map[string]interface{}{
				"s3_key": key,
				"bucket": s.bucket,
			})
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
				return errors.ClassifyError(ctx.Err())
			}
		}

		s.logger.InfoWithFields("File upload completed successfully", map[string]interface{}{
			"s3_key":          key,
			"file_size_bytes": fileSize,
		})

		return nil
	})
}

// GeneratePresignedURL generates a presigned URL for downloading a file
func (s *S3ServiceImpl) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	var result string
	err := s.logger.LogOperation("generate_presigned_url", func() error {
		if key == "" {
			return errors.NewAppError(errors.ErrInvalidInput, "S3 object key cannot be empty", nil)
		}

		if expiration <= 0 {
			return errors.NewAppError(errors.ErrInvalidInput, "expiration duration must be positive", nil)
		}

		// Limit maximum expiration to 7 days for security (as per requirement 8.5)
		maxExpiration := 7 * 24 * time.Hour
		originalExpiration := expiration
		if expiration > maxExpiration {
			expiration = maxExpiration
			s.logger.WarnWithFields("Presigned URL expiration capped for security", map[string]interface{}{
				"requested_duration_hours": originalExpiration.Hours(),
				"capped_duration_hours":    expiration.Hours(),
				"s3_key":                   key,
			})
		}

		s.logger.InfoWithFields("Generating presigned URL", map[string]interface{}{
			"s3_key":           key,
			"expiration_hours": expiration.Hours(),
		})

		// Create presigned request
		request, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = expiration
		})

		if err != nil {
			s.logger.ErrorWithFields("Failed to generate presigned URL", map[string]interface{}{
				"s3_key": key,
				"bucket": s.bucket,
			})
			return s.handleS3Error("generate presigned URL", err)
		}

		result = request.URL
		s.logger.InfoWithFields("Presigned URL generated successfully", map[string]interface{}{
			"s3_key":           key,
			"expiration_hours": expiration.Hours(),
		})

		return nil
	})

	return result, err
}

// DeleteObject deletes an object from S3
func (s *S3ServiceImpl) DeleteObject(ctx context.Context, key string) error {
	return s.logger.LogOperation("delete_object", func() error {
		if key == "" {
			return errors.NewAppError(errors.ErrInvalidInput, "S3 object key cannot be empty", nil)
		}

		s.logger.InfoWithFields("Deleting S3 object", map[string]interface{}{
			"s3_key": key,
			"bucket": s.bucket,
		})

		input := &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}

		_, err := s.client.DeleteObject(ctx, input)
		if err != nil {
			s.logger.ErrorWithFields("Failed to delete S3 object", map[string]interface{}{
				"s3_key": key,
				"bucket": s.bucket,
			})
			return s.handleS3Error("delete object", err)
		}

		s.logger.InfoWithFields("S3 object deleted successfully", map[string]interface{}{
			"s3_key": key,
		})

		return nil
	})
}

// HeadObject retrieves metadata about an object without downloading it
func (s *S3ServiceImpl) HeadObject(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	var result *s3.HeadObjectOutput
	err := s.logger.LogOperation("head_object", func() error {
		if key == "" {
			return errors.NewAppError(errors.ErrInvalidInput, "S3 object key cannot be empty", nil)
		}

		s.logger.DebugWithFields("Getting S3 object metadata", map[string]interface{}{
			"s3_key": key,
			"bucket": s.bucket,
		})

		input := &s3.HeadObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}

		output, err := s.client.HeadObject(ctx, input)
		if err != nil {
			s.logger.ErrorWithFields("Failed to get S3 object metadata", map[string]interface{}{
				"s3_key": key,
				"bucket": s.bucket,
			})
			return s.handleS3Error("get object metadata", err)
		}

		result = output
		s.logger.DebugWithFields("S3 object metadata retrieved successfully", map[string]interface{}{
			"s3_key":     key,
			"size_bytes": output.ContentLength,
		})

		return nil
	})

	return result, err
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

// handleS3Error converts AWS S3 errors to user-friendly AppErrors
func (s *S3ServiceImpl) handleS3Error(operation string, err error) error {
	if err == nil {
		return nil
	}

	s.logger.ErrorWithFields("AWS S3 operation failed", map[string]interface{}{
		"operation": operation,
		"bucket":    s.bucket,
	})

	// First try to classify the error using our error classification system
	appErr := errors.ClassifyError(err)
	
	// If it's already classified appropriately, return it
	if appErr.Code != errors.ErrUnknownError {
		return appErr
	}

	// Handle specific AWS error patterns that might not be caught by classification
	errStr := strings.ToLower(err.Error())
	
	if strings.Contains(errStr, "nosuchbucket") {
		return errors.NewAppErrorWithContext(
			errors.ErrS3BucketNotFound,
			fmt.Sprintf("S3 bucket '%s' does not exist or is not accessible", s.bucket),
			err,
			map[string]interface{}{
				"bucket":    s.bucket,
				"operation": operation,
			},
		)
	}
	
	if strings.Contains(errStr, "nosuchkey") {
		return errors.NewAppErrorWithContext(
			errors.ErrS3ObjectNotFound,
			"The requested file was not found in S3",
			err,
			map[string]interface{}{
				"bucket":    s.bucket,
				"operation": operation,
			},
		)
	}
	
	if strings.Contains(errStr, "accessdenied") {
		return errors.NewAppErrorWithContext(
			errors.ErrS3AccessDenied,
			fmt.Sprintf("Access denied to S3 bucket '%s'", s.bucket),
			err,
			map[string]interface{}{
				"bucket":    s.bucket,
				"operation": operation,
			},
		)
	}

	// For any other AWS service errors, wrap with context
	return errors.NewAppErrorWithContext(
		errors.ErrAWSServiceError,
		fmt.Sprintf("AWS S3 operation failed: %s", operation),
		err,
		map[string]interface{}{
			"bucket":    s.bucket,
			"operation": operation,
		},
	)
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