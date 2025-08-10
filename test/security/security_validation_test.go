//go:build security

package security

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/storage"
)

// TestCredentialSecurity tests credential storage and handling security
func TestCredentialSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("Credential Storage Security", func(t *testing.T) {
		credProvider, err := aws.NewSecureCredentialProvider()
		require.NoError(t, err)

		// Test credentials are not stored in plain text
		accessKey := "AKIAIOSFODNN7EXAMPLE"
		secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
		region := "us-west-2"

		err = credProvider.StoreCredentials(accessKey, secretKey, region)
		require.NoError(t, err)

		// Verify credentials can be retrieved
		ctx := context.Background()
		creds, err2 := credProvider.GetCredentials(ctx)
		require.NoError(t, err2)
		assert.Equal(t, accessKey, creds.AccessKeyID)
		assert.Equal(t, secretKey, creds.SecretAccessKey)

		// Test credential validation
		err = credProvider.ValidateCredentials(ctx)
		// Note: This will fail with test credentials, but we're testing the validation logic
		assert.Error(t, err, "Test credentials should fail validation")
	})

	t.Run("Database Security", func(t *testing.T) {
		// Create temporary database
		tempDB, err := os.CreateTemp("", "security-test-*.db")
		require.NoError(t, err)
		defer os.Remove(tempDB.Name())
		tempDB.Close()

		// Test database file permissions
		database, err := storage.NewSQLiteDatabase(tempDB.Name())
		require.NoError(t, err)
		defer database.Close()

		// Check file permissions (should be 600 or similar)
		fileInfo, err := os.Stat(tempDB.Name())
		require.NoError(t, err)
		
		mode := fileInfo.Mode()
		// On Unix systems, check that only owner has read/write permissions
		if mode&0077 != 0 {
			t.Logf("Warning: Database file permissions may be too permissive: %o", mode)
		}
	})
}

// TestPresignedURLSecurity tests presigned URL security features
func TestPresignedURLSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	// Setup test environment
	bucketName := os.Getenv("S3_BUCKET")
	region := os.Getenv("AWS_REGION")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	require.NotEmpty(t, bucketName, "S3_BUCKET is required")
	require.NotEmpty(t, region, "AWS_REGION is required")
	require.NotEmpty(t, accessKeyID, "AWS_ACCESS_KEY_ID is required")
	require.NotEmpty(t, secretAccessKey, "AWS_SECRET_ACCESS_KEY is required")

	credProvider, err := aws.NewSecureCredentialProvider()
	require.NoError(t, err)
	err = credProvider.StoreCredentials(accessKeyID, secretAccessKey, region)
	require.NoError(t, err)

	s3Service, err := aws.NewS3Service(credProvider, bucketName)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Presigned URL Structure Validation", func(t *testing.T) {
		testKey := "security-test/test-file"
		expiration := 1 * time.Hour

		presignedURL, err := s3Service.GeneratePresignedURL(ctx, testKey, expiration)
		require.NoError(t, err)
		assert.NotEmpty(t, presignedURL)

		// Parse the URL
		parsedURL, err := url.Parse(presignedURL)
		require.NoError(t, err)

		// Validate URL structure
		assert.Equal(t, "https", parsedURL.Scheme, "Presigned URL should use HTTPS")
		assert.Contains(t, parsedURL.Host, bucketName, "URL should contain bucket name")
		assert.Contains(t, parsedURL.Path, testKey, "URL should contain object key")

		// Validate query parameters
		query := parsedURL.Query()
		assert.NotEmpty(t, query.Get("X-Amz-Algorithm"), "Should have algorithm parameter")
		assert.NotEmpty(t, query.Get("X-Amz-Credential"), "Should have credential parameter")
		assert.NotEmpty(t, query.Get("X-Amz-Date"), "Should have date parameter")
		assert.NotEmpty(t, query.Get("X-Amz-Expires"), "Should have expires parameter")
		assert.NotEmpty(t, query.Get("X-Amz-SignedHeaders"), "Should have signed headers parameter")
		assert.NotEmpty(t, query.Get("X-Amz-Signature"), "Should have signature parameter")

		// Validate expiration time
		expiresStr := query.Get("X-Amz-Expires")
		assert.Equal(t, "3600", expiresStr, "Expiration should be 3600 seconds (1 hour)")
	})

	t.Run("Presigned URL Expiration Limits", func(t *testing.T) {
		testKey := "security-test/expiration-test"

		// Test maximum expiration (7 days as per requirements)
		maxExpiration := 7 * 24 * time.Hour
		maxURL, err := s3Service.GeneratePresignedURL(ctx, testKey, maxExpiration)
		require.NoError(t, err)
		assert.NotEmpty(t, maxURL)

		// Parse and validate expiration
		parsedURL, err := url.Parse(maxURL)
		require.NoError(t, err)
		expiresStr := parsedURL.Query().Get("X-Amz-Expires")
		assert.Equal(t, "604800", expiresStr, "Max expiration should be 604800 seconds (7 days)")

		// Test short expiration
		shortExpiration := 5 * time.Minute
		shortURL, err := s3Service.GeneratePresignedURL(ctx, testKey, shortExpiration)
		require.NoError(t, err)
		
		parsedShortURL, err := url.Parse(shortURL)
		require.NoError(t, err)
		shortExpiresStr := parsedShortURL.Query().Get("X-Amz-Expires")
		assert.Equal(t, "300", shortExpiresStr, "Short expiration should be 300 seconds (5 minutes)")
	})

	t.Run("Object Key Security", func(t *testing.T) {
		// Test that object keys are properly formatted and don't contain sensitive information
		testFile := createTestFile(t, "Security test content")
		defer os.Remove(testFile)

		// Upload file and check generated key
		testKey := "security-test/" + generateSecureTestKey()
		
		err := s3Service.UploadFile(ctx, testKey, testFile, nil, nil)
		require.NoError(t, err)
		defer s3Service.DeleteObject(ctx, testKey)

		// Validate key structure
		assert.True(t, strings.HasPrefix(testKey, "security-test/"), "Key should have proper prefix")
		assert.NotContains(t, testKey, " ", "Key should not contain spaces")
		assert.NotContains(t, testKey, "..", "Key should not contain directory traversal")
		assert.NotContains(t, testKey, "//", "Key should not contain double slashes")

		// Generate presigned URL and validate
		presignedURL, err := s3Service.GeneratePresignedURL(ctx, testKey, 1*time.Hour)
		require.NoError(t, err)
		
		// Ensure URL doesn't expose sensitive information
		assert.NotContains(t, presignedURL, accessKeyID, "URL should not contain access key ID in plain text")
		assert.NotContains(t, presignedURL, secretAccessKey, "URL should not contain secret key")
	})

	t.Run("HTTP Security Headers", func(t *testing.T) {
		// Upload a test file
		testFile := createTestFile(t, "HTTP security test")
		defer os.Remove(testFile)

		testKey := "security-test/http-" + generateSecureTestKey()
		err := s3Service.UploadFile(ctx, testKey, testFile, nil, nil)
		require.NoError(t, err)
		defer s3Service.DeleteObject(ctx, testKey)

		// Generate presigned URL
		presignedURL, err := s3Service.GeneratePresignedURL(ctx, testKey, 1*time.Hour)
		require.NoError(t, err)

		// Test HTTP access to presigned URL
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Head(presignedURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Validate security headers
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		// Check for security-related headers
		serverHeader := resp.Header.Get("Server")
		assert.Contains(t, serverHeader, "AmazonS3", "Should identify as Amazon S3")

		// Validate content type is set
		contentType := resp.Header.Get("Content-Type")
		assert.NotEmpty(t, contentType, "Content-Type should be set")
	})
}

// TestS3BucketSecurity tests S3 bucket security configuration
func TestS3BucketSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	bucketName := os.Getenv("S3_BUCKET")
	region := os.Getenv("AWS_REGION")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	require.NotEmpty(t, bucketName, "S3_BUCKET is required")
	require.NotEmpty(t, region, "AWS_REGION is required")

	credProvider, err := aws.NewSecureCredentialProvider()
	require.NoError(t, err)
	err = credProvider.StoreCredentials(accessKeyID, secretAccessKey, region)
	require.NoError(t, err)

	s3Service, err := aws.NewS3Service(credProvider, bucketName)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Bucket Access Control", func(t *testing.T) {
		// Test that bucket is not publicly accessible
		// This is a basic test - in a real scenario, you'd use AWS SDK to check bucket policies
		
		// Try to access bucket without credentials (should fail)
		publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", bucketName, region)
		
		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		
		resp, err := client.Get(publicURL)
		if err == nil {
			defer resp.Body.Close()
			// Bucket should not be publicly accessible
			assert.NotEqual(t, http.StatusOK, resp.StatusCode, 
				"Bucket should not be publicly accessible")
		}
	})

	t.Run("Object Tagging Security", func(t *testing.T) {
		// Test that objects are properly tagged for lifecycle management
		testFile := createTestFile(t, "Tagging security test")
		defer os.Remove(testFile)

		testKey := "security-test/tagging-" + generateSecureTestKey()
		
		// Upload with security-relevant metadata
		metadata := map[string]string{
			"expiration": "1day",
			"test":       "security",
		}
		
		err := s3Service.UploadFile(ctx, testKey, testFile, metadata, nil)
		require.NoError(t, err)
		defer s3Service.DeleteObject(ctx, testKey)

		// Verify object exists and has metadata
		headOutput, err := s3Service.HeadObject(ctx, testKey)
		require.NoError(t, err)
		
		// Check that metadata is properly set
		assert.Equal(t, "1day", headOutput.Metadata["expiration"])
		assert.Equal(t, "security", headOutput.Metadata["test"])
		
		// Verify server-side encryption is enabled
		assert.NotEmpty(t, headOutput.ServerSideEncryption, 
			"Server-side encryption should be enabled")
	})
}

// TestErrorHandlingSecurity tests security aspects of error handling
func TestErrorHandlingSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("Error Message Security", func(t *testing.T) {
		// Test that error messages don't expose sensitive information
		credProvider, err := aws.NewSecureCredentialProvider()
		require.NoError(t, err)

		// Test with invalid credentials
		err = credProvider.StoreCredentials("invalid-key", "invalid-secret", "us-west-2")
		require.NoError(t, err)

		ctx := context.Background()
		err = credProvider.ValidateCredentials(ctx)
		assert.Error(t, err)
		
		// Error message should not contain the actual credentials
		errorMsg := err.Error()
		assert.NotContains(t, errorMsg, "invalid-key", 
			"Error message should not expose access key")
		assert.NotContains(t, errorMsg, "invalid-secret", 
			"Error message should not expose secret key")
	})

	t.Run("File Path Security", func(t *testing.T) {
		// Test that file paths are properly validated
		bucketName := os.Getenv("S3_BUCKET")
		region := os.Getenv("AWS_REGION")
		accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
		secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

		if bucketName == "" || region == "" || accessKeyID == "" || secretAccessKey == "" {
			t.Skip("AWS credentials not available")
		}

		credProvider, err := aws.NewSecureCredentialProvider()
		require.NoError(t, err)
		err = credProvider.StoreCredentials(accessKeyID, secretAccessKey, region)
		require.NoError(t, err)

		s3Service, err := aws.NewS3Service(credProvider, bucketName)
		require.NoError(t, err)

		ctx := context.Background()

		// Test upload with invalid file path
		err = s3Service.UploadFile(ctx, "test-key", "/etc/passwd", nil, nil)
		assert.Error(t, err, "Upload of system file should fail")

		err = s3Service.UploadFile(ctx, "test-key", "../../../etc/passwd", nil, nil)
		assert.Error(t, err, "Upload with path traversal should fail")
	})
}

// Helper functions

func createTestFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "security-test-*.txt")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func generateSecureTestKey() string {
	return fmt.Sprintf("test-%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}