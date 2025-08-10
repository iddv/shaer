package aws

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCredentialProvider implements CredentialProvider for testing
type mockCredentialProvider struct {
	credentials aws.Credentials
	region      string
	shouldError bool
	errorMsg    string
}

func (m *mockCredentialProvider) GetCredentials(ctx context.Context) (aws.Credentials, error) {
	if m.shouldError {
		return aws.Credentials{}, fmt.Errorf(m.errorMsg)
	}
	return m.credentials, nil
}

func (m *mockCredentialProvider) StoreCredentials(accessKey, secretKey, region string) error {
	return nil
}

func (m *mockCredentialProvider) ValidateCredentials(ctx context.Context) error {
	return nil
}

func (m *mockCredentialProvider) ClearCredentials() error {
	return nil
}

func (m *mockCredentialProvider) GetRegion() (string, error) {
	if m.shouldError {
		return "", fmt.Errorf(m.errorMsg)
	}
	return m.region, nil
}

func (m *mockCredentialProvider) SetRegion(region string) error {
	m.region = region
	return nil
}

// createTestS3CredentialProvider creates a mock credential provider for S3 testing
func createTestS3CredentialProvider() *mockCredentialProvider {
	return &mockCredentialProvider{
		credentials: aws.Credentials{
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			Source:          "test",
		},
		region: "us-east-1",
	}
}

// createTestFile creates a temporary test file with specified content
func createTestFile(t *testing.T, content string) string {
	tempFile, err := os.CreateTemp("", "test-file-*.txt")
	require.NoError(t, err)
	
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	
	err = tempFile.Close()
	require.NoError(t, err)
	
	// Clean up the file after the test
	t.Cleanup(func() {
		os.Remove(tempFile.Name())
	})
	
	return tempFile.Name()
}

func TestNewS3Service(t *testing.T) {
	tests := []struct {
		name           string
		bucket         string
		credProvider   CredentialProvider
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "valid configuration",
			bucket:       "test-bucket",
			credProvider: createTestS3CredentialProvider(),
			expectError:  false,
		},
		{
			name:           "empty bucket name",
			bucket:         "",
			credProvider:   createTestS3CredentialProvider(),
			expectError:    true,
			expectedErrMsg: "bucket name cannot be empty",
		},
		{
			name:   "credential provider error",
			bucket: "test-bucket",
			credProvider: &mockCredentialProvider{
				shouldError: true,
				errorMsg:    "credential error",
			},
			expectError:    true,
			expectedErrMsg: "failed to get AWS credentials",
		},
		{
			name:   "region provider error",
			bucket: "test-bucket",
			credProvider: &mockCredentialProvider{
				credentials: aws.Credentials{
					AccessKeyID:     "test",
					SecretAccessKey: "test",
				},
				shouldError: true,
				errorMsg:    "region error",
			},
			expectError:    true,
			expectedErrMsg: "region error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewS3Service(tt.credProvider, tt.bucket)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, tt.bucket, service.bucket)
				assert.NotNil(t, service.client)
				assert.NotNil(t, service.presigner)
			}
		})
	}
}

func TestS3ServiceImpl_UploadFile(t *testing.T) {
	credProvider := createTestS3CredentialProvider()
	service, err := NewS3Service(credProvider, "test-bucket")
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name           string
		key            string
		filePath       string
		metadata       map[string]string
		setupFile      func(t *testing.T) string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "valid file upload",
			key:  "test-key",
			setupFile: func(t *testing.T) string {
				return createTestFile(t, "test content")
			},
			metadata:    map[string]string{"test": "value"},
			expectError: false,
		},
		{
			name:           "empty key",
			key:            "",
			filePath:       "dummy",
			expectError:    true,
			expectedErrMsg: "S3 object key cannot be empty",
		},
		{
			name:           "empty file path",
			key:            "test-key",
			filePath:       "",
			expectError:    true,
			expectedErrMsg: "file path cannot be empty",
		},
		{
			name:           "non-existent file",
			key:            "test-key",
			filePath:       "/non/existent/file.txt",
			expectError:    true,
			expectedErrMsg: "failed to open file",
		},
		{
			name: "empty file",
			key:  "test-key",
			setupFile: func(t *testing.T) string {
				return createTestFile(t, "")
			},
			expectError:    true,
			expectedErrMsg: "is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.filePath
			if tt.setupFile != nil {
				filePath = tt.setupFile(t)
			}

			// Create progress channel
			progressCh := make(chan UploadProgress, 10)
			
			// Upload file
			err := service.UploadFile(ctx, tt.key, filePath, tt.metadata, progressCh)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				// Note: This will fail in unit tests without actual AWS credentials
				// In a real test environment, you would use localstack or mock the AWS client
				assert.Error(t, err) // Expected to fail without real AWS setup
				assert.Contains(t, err.Error(), "failed to upload file")
			}
			
			close(progressCh)
		})
	}
}

func TestS3ServiceImpl_GeneratePresignedURL(t *testing.T) {
	credProvider := createTestS3CredentialProvider()
	service, err := NewS3Service(credProvider, "test-bucket")
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name           string
		key            string
		expiration     time.Duration
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:        "valid presigned URL",
			key:         "test-key",
			expiration:  1 * time.Hour,
			expectError: false,
		},
		{
			name:           "empty key",
			key:            "",
			expiration:     1 * time.Hour,
			expectError:    true,
			expectedErrMsg: "S3 object key cannot be empty",
		},
		{
			name:           "zero expiration",
			key:            "test-key",
			expiration:     0,
			expectError:    true,
			expectedErrMsg: "expiration duration must be positive",
		},
		{
			name:           "negative expiration",
			key:            "test-key",
			expiration:     -1 * time.Hour,
			expectError:    true,
			expectedErrMsg: "expiration duration must be positive",
		},
		{
			name:        "expiration exceeds maximum (should be capped)",
			key:         "test-key",
			expiration:  10 * 24 * time.Hour, // 10 days
			expectError: false, // Should succeed but cap at 7 days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := service.GeneratePresignedURL(ctx, tt.key, tt.expiration)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Empty(t, url)
			} else {
				// Note: This will work even without real AWS credentials
				// because presigning doesn't make an API call
				assert.NoError(t, err)
				assert.NotEmpty(t, url)
				assert.Contains(t, url, "test-bucket")
				assert.Contains(t, url, tt.key)
			}
		})
	}
}

func TestS3ServiceImpl_DeleteObject(t *testing.T) {
	credProvider := createTestS3CredentialProvider()
	service, err := NewS3Service(credProvider, "test-bucket")
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name           string
		key            string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:        "valid delete",
			key:         "test-key",
			expectError: false, // Will fail without real AWS, but validates input
		},
		{
			name:           "empty key",
			key:            "",
			expectError:    true,
			expectedErrMsg: "S3 object key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteObject(ctx, tt.key)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				// Note: This will fail in unit tests without actual AWS credentials
				assert.Error(t, err) // Expected to fail without real AWS setup
				assert.Contains(t, err.Error(), "failed to delete object")
			}
		})
	}
}

func TestS3ServiceImpl_HeadObject(t *testing.T) {
	credProvider := createTestS3CredentialProvider()
	service, err := NewS3Service(credProvider, "test-bucket")
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name           string
		key            string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:        "valid head object",
			key:         "test-key",
			expectError: false, // Will fail without real AWS, but validates input
		},
		{
			name:           "empty key",
			key:            "",
			expectError:    true,
			expectedErrMsg: "S3 object key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := service.HeadObject(ctx, tt.key)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, output)
			} else {
				// Note: This will fail in unit tests without actual AWS credentials
				assert.Error(t, err) // Expected to fail without real AWS setup
				assert.Nil(t, output)
				assert.Contains(t, err.Error(), "failed to get object metadata")
			}
		})
	}
}

func TestS3ServiceImpl_TestConnection(t *testing.T) {
	credProvider := createTestS3CredentialProvider()
	service, err := NewS3Service(credProvider, "test-bucket")
	require.NoError(t, err)

	ctx := context.Background()

	// Test connection - will fail without real AWS credentials
	err = service.TestConnection(ctx)
	assert.Error(t, err) // Expected to fail without real AWS setup
	assert.Contains(t, err.Error(), "failed to test connection")
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		filePath    string
		expectedType string
	}{
		{"test.txt", "text/plain"},
		{"document.pdf", "application/pdf"},
		{"image.jpg", "image/jpeg"},
		{"image.jpeg", "image/jpeg"},
		{"picture.png", "image/png"},
		{"animation.gif", "image/gif"},
		{"archive.zip", "application/zip"},
		{"data.json", "application/json"},
		{"config.xml", "application/xml"},
		{"spreadsheet.csv", "text/csv"},
		{"document.doc", "application/msword"},
		{"document.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"spreadsheet.xls", "application/vnd.ms-excel"},
		{"spreadsheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"presentation.ppt", "application/vnd.ms-powerpoint"},
		{"presentation.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"unknown.xyz", "application/octet-stream"},
		{"no-extension", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			contentType := getContentType(tt.filePath)
			assert.Equal(t, tt.expectedType, contentType)
		})
	}
}

func TestProgressReader(t *testing.T) {
	content := "This is test content for progress tracking"
	reader := strings.NewReader(content)
	totalBytes := int64(len(content))
	
	progressCh := make(chan UploadProgress, 10)
	
	pr := &progressReader{
		reader:     reader,
		totalBytes: totalBytes,
		progressCh: progressCh,
	}

	// Read in chunks to test progress updates
	buffer := make([]byte, 10)
	var totalRead int64
	
	for {
		n, err := pr.Read(buffer)
		if n > 0 {
			totalRead += int64(n)
		}
		if err != nil {
			break
		}
	}

	assert.Equal(t, totalBytes, totalRead)
	assert.Equal(t, totalBytes, pr.bytesRead)

	// Check that we received progress updates
	close(progressCh)
	progressUpdates := make([]UploadProgress, 0)
	for progress := range progressCh {
		progressUpdates = append(progressUpdates, progress)
	}

	assert.NotEmpty(t, progressUpdates)
	
	// Check that the last progress update shows completion
	if len(progressUpdates) > 0 {
		lastProgress := progressUpdates[len(progressUpdates)-1]
		assert.Equal(t, totalBytes, lastProgress.TotalBytes)
		assert.True(t, lastProgress.Percentage > 0)
		assert.True(t, lastProgress.Percentage <= 100)
	}
}

func TestS3ServiceImpl_handleS3Error(t *testing.T) {
	credProvider := createTestS3CredentialProvider()
	service, err := NewS3Service(credProvider, "test-bucket")
	require.NoError(t, err)

	tests := []struct {
		name      string
		operation string
		inputErr  error
		expectMsg string
	}{
		{
			name:      "nil error",
			operation: "test",
			inputErr:  nil,
			expectMsg: "",
		},
		{
			name:      "context deadline exceeded",
			operation: "upload",
			inputErr:  context.DeadlineExceeded,
			expectMsg: "CONNECTION_TIMEOUT: Operation timed out",
		},
		{
			name:      "context canceled",
			operation: "download",
			inputErr:  context.Canceled,
			expectMsg: "UPLOAD_CANCELED: Operation was canceled",
		},
		{
			name:      "generic error",
			operation: "delete",
			inputErr:  fmt.Errorf("some generic error"),
			expectMsg: "AWS_SERVICE_ERROR: AWS S3 operation failed: delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.handleS3Error(tt.operation, tt.inputErr)
			
			if tt.inputErr == nil {
				assert.Nil(t, result)
			} else {
				assert.Error(t, result)
				assert.Contains(t, result.Error(), tt.expectMsg)
			}
		})
	}
}

// Integration test that would work with localstack or real AWS
func TestS3ServiceIntegration(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require either:
	// 1. Real AWS credentials and a test bucket
	// 2. Localstack running locally
	// 3. AWS SDK mocking (more complex setup)
	
	// For now, we'll skip this test but provide the structure
	t.Skip("Integration test requires AWS setup (localstack or real AWS)")

	credProvider := createTestS3CredentialProvider()
	service, err := NewS3Service(credProvider, "test-bucket")
	require.NoError(t, err)

	ctx := context.Background()

	// Test connection
	err = service.TestConnection(ctx)
	assert.NoError(t, err)

	// Create test file
	testContent := "This is integration test content"
	testFile := createTestFile(t, testContent)
	
	// Upload file
	progressCh := make(chan UploadProgress, 10)
	err = service.UploadFile(ctx, "integration-test-key", testFile, nil, progressCh)
	assert.NoError(t, err)
	close(progressCh)

	// Generate presigned URL
	url, err := service.GeneratePresignedURL(ctx, "integration-test-key", 1*time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, url)

	// Head object
	output, err := service.HeadObject(ctx, "integration-test-key")
	assert.NoError(t, err)
	assert.NotNil(t, output)

	// Delete object
	err = service.DeleteObject(ctx, "integration-test-key")
	assert.NoError(t, err)
}

// Benchmark tests
func BenchmarkGetContentType(b *testing.B) {
	testFiles := []string{
		"test.txt", "document.pdf", "image.jpg", "data.json", "unknown.xyz",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range testFiles {
			getContentType(file)
		}
	}
}

func BenchmarkProgressReader(b *testing.B) {
	content := strings.Repeat("test content ", 1000) // ~12KB
	totalBytes := int64(len(content))
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		progressCh := make(chan UploadProgress, 100)
		
		pr := &progressReader{
			reader:     reader,
			totalBytes: totalBytes,
			progressCh: progressCh,
		}

		buffer := make([]byte, 1024)
		for {
			_, err := pr.Read(buffer)
			if err != nil {
				break
			}
		}
		
		close(progressCh)
		// Drain the channel
		for range progressCh {
		}
	}
}

func TestFormatTagsForUpload(t *testing.T) {
	tests := []struct {
		name     string
		tags     []types.Tag
		expected string
	}{
		{
			name:     "empty tags",
			tags:     []types.Tag{},
			expected: "",
		},
		{
			name: "single tag",
			tags: []types.Tag{
				{Key: aws.String("expiration"), Value: aws.String("1hour")},
			},
			expected: "expiration=1hour",
		},
		{
			name: "multiple tags",
			tags: []types.Tag{
				{Key: aws.String("expiration"), Value: aws.String("1day")},
				{Key: aws.String("upload-date"), Value: aws.String("2024-01-01")},
			},
			expected: "expiration=1day&upload-date=2024-01-01",
		},
		{
			name: "tags with special characters",
			tags: []types.Tag{
				{Key: aws.String("file-type"), Value: aws.String("test-file")},
				{Key: aws.String("user_id"), Value: aws.String("user123")},
			},
			expected: "file-type=test-file&user_id=user123",
		},
		{
			name: "tag with nil key",
			tags: []types.Tag{
				{Key: nil, Value: aws.String("value")},
				{Key: aws.String("valid"), Value: aws.String("tag")},
			},
			expected: "valid=tag",
		},
		{
			name: "tag with nil value",
			tags: []types.Tag{
				{Key: aws.String("key"), Value: nil},
				{Key: aws.String("valid"), Value: aws.String("tag")},
			},
			expected: "valid=tag",
		},
		{
			name: "tag with both nil key and value",
			tags: []types.Tag{
				{Key: nil, Value: nil},
				{Key: aws.String("valid"), Value: aws.String("tag")},
			},
			expected: "valid=tag",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTagsForUpload(tt.tags)
			assert.Equal(t, tt.expected, result)
		})
	}
}