//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"file-sharing-app/internal/aws"
	"file-sharing-app/test/config"
)

// TestAWSIntegration tests the complete AWS integration workflow
func TestAWSIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	testConfig, err := config.LoadTestConfig()
	require.NoError(t, err, "Failed to load test configuration")
	
	err = testConfig.Validate()
	require.NoError(t, err, "Test configuration validation failed")

	t.Logf("Running integration tests with bucket: %s in region: %s", testConfig.S3Bucket, testConfig.AWSRegion)

	// Create credential provider
	credProvider, err := aws.NewSecureCredentialProvider()
	require.NoError(t, err)

	// Store test credentials temporarily
	err = credProvider.StoreCredentials(testConfig.AWSAccessKeyID, testConfig.AWSSecretAccessKey, testConfig.AWSRegion)
	require.NoError(t, err)

	// Create S3 service
	s3Service, err := aws.NewS3Service(credProvider, testConfig.S3Bucket)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("File Upload and Download Workflow", func(t *testing.T) {
		// Create test file
		testContent := "This is integration test content for AWS S3"
		testFile := createTestFile(t, testContent)
		defer os.Remove(testFile)

		testKey := testConfig.GetTestKeyPrefix() + "/" + generateTestKey()

		// Test file upload
		progressCh := make(chan aws.UploadProgress, 10)
		go func() {
			for progress := range progressCh {
				t.Logf("Upload progress: %.2f%% (%d/%d bytes)", 
					progress.Percentage, progress.BytesUploaded, progress.TotalBytes)
			}
		}()

		metadata := map[string]string{
			"expiration": "1day",
			"test":       "integration",
		}

		err := s3Service.UploadFile(ctx, testKey, testFile, metadata, progressCh)
		close(progressCh)
		require.NoError(t, err, "File upload should succeed")

		// Test head object (verify file exists)
		headOutput, err := s3Service.HeadObject(ctx, testKey)
		require.NoError(t, err, "Head object should succeed")
		assert.NotNil(t, headOutput)
		assert.Equal(t, "1day", headOutput.Metadata["expiration"])

		// Test presigned URL generation
		presignedURL, err := s3Service.GeneratePresignedURL(ctx, testKey, 1*time.Hour)
		require.NoError(t, err, "Presigned URL generation should succeed")
		assert.NotEmpty(t, presignedURL)
		assert.Contains(t, presignedURL, testConfig.S3Bucket)
		assert.Contains(t, presignedURL, testKey)

		// Test presigned URL access (basic validation)
		assert.Contains(t, presignedURL, "X-Amz-Algorithm=AWS4-HMAC-SHA256")
		assert.Contains(t, presignedURL, "X-Amz-Expires=3600") // 1 hour

		// Test file deletion
		err = s3Service.DeleteObject(ctx, testKey)
		require.NoError(t, err, "File deletion should succeed")

		// Verify file is deleted
		_, err = s3Service.HeadObject(ctx, testKey)
		assert.Error(t, err, "Head object should fail after deletion")
	})

	t.Run("Large File Upload with Chunking", func(t *testing.T) {
		// Create a larger test file (10MB)
		largeContent := make([]byte, 10*1024*1024) // 10MB
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		largeTestFile := createTestFileWithContent(t, largeContent)
		defer os.Remove(largeTestFile)

		testKey := testConfig.GetTestKeyPrefix() + "/large-" + generateTestKey()

		// Test large file upload
		progressCh := make(chan aws.UploadProgress, 100)
		progressReceived := false
		go func() {
			for progress := range progressCh {
				progressReceived = true
				t.Logf("Large file upload progress: %.2f%% (%d/%d bytes)", 
					progress.Percentage, progress.BytesUploaded, progress.TotalBytes)
			}
		}()

		err := s3Service.UploadFile(ctx, testKey, largeTestFile, nil, progressCh)
		close(progressCh)
		require.NoError(t, err, "Large file upload should succeed")
		assert.True(t, progressReceived, "Progress updates should be received")

		// Verify large file exists
		headOutput, err := s3Service.HeadObject(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, int64(10*1024*1024), *headOutput.ContentLength)

		// Cleanup
		err = s3Service.DeleteObject(ctx, testKey)
		require.NoError(t, err)
	})

	t.Run("Multiple File Operations", func(t *testing.T) {
		const numFiles = 5
		testKeys := make([]string, numFiles)
		testFiles := make([]string, numFiles)

		// Upload multiple files
		for i := 0; i < numFiles; i++ {
			content := fmt.Sprintf("Test file content %d", i)
			testFiles[i] = createTestFile(t, content)
			testKeys[i] = fmt.Sprintf("%s/multi-%d-%s", testConfig.GetTestKeyPrefix(), i, generateTestKey())

			progressCh := make(chan aws.UploadProgress, 10)
			go func() {
				for range progressCh {
					// Consume progress updates
				}
			}()

			err := s3Service.UploadFile(ctx, testKeys[i], testFiles[i], nil, progressCh)
			close(progressCh)
			require.NoError(t, err, "File %d upload should succeed", i)
		}

		// Verify all files exist
		for i, key := range testKeys {
			_, err := s3Service.HeadObject(ctx, key)
			assert.NoError(t, err, "File %d should exist", i)
		}

		// Generate presigned URLs for all files
		for i, key := range testKeys {
			url, err := s3Service.GeneratePresignedURL(ctx, key, 30*time.Minute)
			assert.NoError(t, err, "Presigned URL generation for file %d should succeed", i)
			assert.NotEmpty(t, url)
		}

		// Cleanup all files
		for i, key := range testKeys {
			err := s3Service.DeleteObject(ctx, key)
			assert.NoError(t, err, "File %d deletion should succeed", i)
			os.Remove(testFiles[i])
		}
	})

	t.Run("Error Handling", func(t *testing.T) {
		// Test upload to non-existent file
		err := s3Service.UploadFile(ctx, "test-key", "/non/existent/file", nil, nil)
		assert.Error(t, err, "Upload of non-existent file should fail")

		// Test head object for non-existent key
		_, err = s3Service.HeadObject(ctx, "non-existent-key")
		assert.Error(t, err, "Head object for non-existent key should fail")

		// Test delete non-existent object (should not error)
		err = s3Service.DeleteObject(ctx, "non-existent-key")
		assert.NoError(t, err, "Delete non-existent object should not error")

		// Test presigned URL for non-existent key (should still generate URL)
		url, err := s3Service.GeneratePresignedURL(ctx, "non-existent-key", 1*time.Hour)
		assert.NoError(t, err, "Presigned URL generation should succeed even for non-existent key")
		assert.NotEmpty(t, url)
	})
}

// TestCredentialValidation tests credential validation workflow
func TestCredentialValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	testConfig, err := config.LoadTestConfig()
	require.NoError(t, err, "Failed to load test configuration")

	credProvider, err := aws.NewSecureCredentialProvider()
	require.NoError(t, err)

	// Test storing and retrieving credentials
	err = credProvider.StoreCredentials(testConfig.AWSAccessKeyID, testConfig.AWSSecretAccessKey, testConfig.AWSRegion)
	require.NoError(t, err)

	// Test credential validation
	ctx := context.Background()
	err = credProvider.ValidateCredentials(ctx)
	assert.NoError(t, err, "Credential validation should succeed with valid credentials")

	// Test getting credentials
	creds, err := credProvider.GetCredentials(ctx)
	assert.NoError(t, err)
	assert.Equal(t, testConfig.AWSAccessKeyID, creds.AccessKeyID)
	assert.Equal(t, testConfig.AWSSecretAccessKey, creds.SecretAccessKey)
}

// Helper functions

func createTestFile(t *testing.T, content string) string {
	return createTestFileWithContent(t, []byte(content))
}

func createTestFileWithContent(t *testing.T, content []byte) string {
	tmpFile, err := os.CreateTemp("", "integration-test-*.txt")
	require.NoError(t, err)

	_, err = tmpFile.Write(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func generateTestKey() string {
	return fmt.Sprintf("test-%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}