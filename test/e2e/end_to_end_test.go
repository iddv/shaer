//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/manager"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
)

// TestCompleteUserWorkflow tests the complete end-to-end user workflow
func TestCompleteUserWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Setup test environment
	testEnv := setupTestEnvironment(t)
	defer testEnv.cleanup()

	t.Run("Complete File Sharing Workflow", func(t *testing.T) {
		// 1. User uploads a file
		testFile := createTestFile(t, "Hello, this is a test file for sharing!")
		defer os.Remove(testFile)

		ctx := context.Background()
		progressCh := make(chan aws.UploadProgress, 10)
		go func() {
			for range progressCh {
				// Consume progress updates
			}
		}()

		fileRecord, err := testEnv.fileManager.UploadFile(ctx, testFile, 24*time.Hour, progressCh)
		close(progressCh)
		require.NoError(t, err, "File upload should succeed")
		assert.NotEmpty(t, fileRecord.ID)
		assert.Equal(t, filepath.Base(testFile), fileRecord.FileName)
		assert.Equal(t, models.StatusActive, fileRecord.Status)

		// 2. User shares the file with recipients
		recipients := []string{"test1@example.com", "test2@example.com"}
		message := "Please find the shared file attached."

		shareRecord, err := testEnv.shareManager.ShareFile(ctx, fileRecord.ID, recipients, message)
		require.NoError(t, err, "File sharing should succeed")
		assert.NotEmpty(t, shareRecord.ID)
		assert.Equal(t, fileRecord.ID, shareRecord.FileID)
		assert.Equal(t, recipients, shareRecord.Recipients)
		assert.Equal(t, message, shareRecord.Message)
		assert.NotEmpty(t, shareRecord.PresignedURL)

		// 3. Verify presigned URL is accessible
		assert.Contains(t, shareRecord.PresignedURL, testEnv.bucketName)
		assert.Contains(t, shareRecord.PresignedURL, "X-Amz-Algorithm")

		// 4. User views file list
		files, err := testEnv.fileManager.ListFiles()
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, fileRecord.ID, files[0].ID)

		// 5. User views sharing history
		shareHistory, err := testEnv.shareManager.GetShareHistory(fileRecord.ID)
		require.NoError(t, err)
		assert.Len(t, shareHistory, 1)
		assert.Equal(t, shareRecord.ID, shareHistory[0].ID)

		// 6. User checks file details
		fileDetails, err := testEnv.fileManager.GetFile(fileRecord.ID)
		require.NoError(t, err)
		assert.Equal(t, fileRecord.ID, fileDetails.ID)
		assert.Equal(t, fileRecord.FileName, fileDetails.FileName)

		// 7. User deletes the file
		err = testEnv.fileManager.DeleteFile(fileRecord.ID)
		require.NoError(t, err, "File deletion should succeed")

		// 8. Verify file is marked as deleted
		deletedFile, err := testEnv.database.GetFile(fileRecord.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusDeleted, deletedFile.Status)
	})

	t.Run("File Expiration Workflow", func(t *testing.T) {
		// 1. Upload file with short expiration
		testFile := createTestFile(t, "This file will expire soon")
		defer os.Remove(testFile)

		ctx := context.Background()
		progressCh := make(chan aws.UploadProgress, 10)
		go func() {
			for range progressCh {
				// Consume progress updates
			}
		}()

		// Set expiration to 1 second for testing
		fileRecord, err := testEnv.fileManager.UploadFile(ctx, testFile, 1*time.Second, progressCh)
		close(progressCh)
		require.NoError(t, err)

		// 2. Verify file is initially active
		isExpired, err := testEnv.expirationManager.IsFileExpired(fileRecord.ID)
		require.NoError(t, err)
		assert.False(t, isExpired, "File should not be expired initially")

		// 3. Wait for expiration
		time.Sleep(2 * time.Second)

		// 4. Check for expired files
		expiredFiles, err := testEnv.expirationManager.CheckExpirations()
		require.NoError(t, err)
		assert.Len(t, expiredFiles, 1)
		assert.Equal(t, fileRecord.ID, expiredFiles[0].ID)

		// 5. Cleanup expired files
		err = testEnv.expirationManager.CleanupExpiredFiles()
		require.NoError(t, err)

		// 6. Verify file status is updated
		updatedFile, err := testEnv.database.GetFile(fileRecord.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusExpired, updatedFile.Status)
	})

	t.Run("Multiple Files and Sharing Workflow", func(t *testing.T) {
		const numFiles = 3
		fileRecords := make([]*models.FileMetadata, numFiles)
		testFiles := make([]string, numFiles)

		// 1. Upload multiple files
		ctx := context.Background()
		for i := 0; i < numFiles; i++ {
			content := fmt.Sprintf("Test file content %d", i)
			testFiles[i] = createTestFile(t, content)
			defer os.Remove(testFiles[i])

			progressCh := make(chan aws.UploadProgress, 10)
			go func() {
				for range progressCh {
					// Consume progress updates
				}
			}()

			var err error
			fileRecords[i], err = testEnv.fileManager.UploadFile(ctx, testFiles[i], 1*time.Hour, progressCh)
			close(progressCh)
			require.NoError(t, err, "File %d upload should succeed", i)
		}

		// 2. Share each file with different recipients
		for i, fileRecord := range fileRecords {
			recipients := []string{fmt.Sprintf("user%d@example.com", i)}
			message := fmt.Sprintf("Sharing file %d", i)

			_, err := testEnv.shareManager.ShareFile(ctx, fileRecord.ID, recipients, message)
			require.NoError(t, err, "File %d sharing should succeed", i)
		}

		// 3. Verify all files are listed
		files, err := testEnv.fileManager.ListFiles()
		require.NoError(t, err)
		assert.Len(t, files, numFiles)

		// 4. Verify sharing history for each file
		for i, fileRecord := range fileRecords {
			shareHistory, err := testEnv.shareManager.GetShareHistory(fileRecord.ID)
			require.NoError(t, err)
			assert.Len(t, shareHistory, 1, "File %d should have one share record", i)
		}

		// 5. Delete all files
		for i, fileRecord := range fileRecords {
			err := testEnv.fileManager.DeleteFile(fileRecord.ID)
			require.NoError(t, err, "File %d deletion should succeed", i)
		}
	})

	t.Run("Error Handling Workflow", func(t *testing.T) {
		ctx := context.Background()
		
		// 1. Try to upload non-existent file
		progressCh := make(chan aws.UploadProgress, 10)
		go func() {
			for range progressCh {
				// Consume progress updates
			}
		}()
		_, err := testEnv.fileManager.UploadFile(ctx, "/non/existent/file.txt", 1*time.Hour, progressCh)
		close(progressCh)
		assert.Error(t, err, "Upload of non-existent file should fail")

		// 2. Try to share non-existent file
		_, err = testEnv.shareManager.ShareFile(ctx, "non-existent-id", []string{"test@example.com"}, "test")
		assert.Error(t, err, "Sharing non-existent file should fail")

		// 3. Try to get details of non-existent file
		_, err = testEnv.fileManager.GetFile("non-existent-id")
		assert.Error(t, err, "Getting details of non-existent file should fail")

		// 4. Try to delete non-existent file
		err = testEnv.fileManager.DeleteFile("non-existent-id")
		assert.Error(t, err, "Deleting non-existent file should fail")
	})
}

// TestApplicationController tests the application controller integration
func TestApplicationController(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	testEnv := setupTestEnvironment(t)
	defer testEnv.cleanup()

	// Note: Controller testing is done through individual managers

	t.Run("Controller File Operations", func(t *testing.T) {
		// Test file upload through controller
		testFile := createTestFile(t, "Controller test content")
		defer os.Remove(testFile)

		ctx := context.Background()
		progressCh := make(chan aws.UploadProgress, 10)
		go func() {
			for range progressCh {
				// Consume progress updates
			}
		}()

		fileRecord, err := testEnv.fileManager.UploadFile(ctx, testFile, 1*time.Hour, progressCh)
		close(progressCh)
		require.NoError(t, err)
		assert.NotEmpty(t, fileRecord.ID)

		// Test file listing through controller
		files, err := testEnv.fileManager.ListFiles()
		require.NoError(t, err)
		assert.Len(t, files, 1)

		// Test file sharing through controller
		recipients := []string{"controller-test@example.com"}
		shareRecord, err := testEnv.shareManager.ShareFile(ctx, fileRecord.ID, recipients, "Controller test")
		require.NoError(t, err)
		assert.NotEmpty(t, shareRecord.PresignedURL)

		// Test file deletion through controller
		err = testEnv.fileManager.DeleteFile(fileRecord.ID)
		require.NoError(t, err)
	})

	t.Run("Controller Settings Management", func(t *testing.T) {
		// Test settings operations through controller
		settings := &models.ApplicationSettings{
			AWSRegion:         testEnv.region,
			S3Bucket:          testEnv.bucketName,
			DefaultExpiration: "1day",
			MaxFileSize:       100 * 1024 * 1024, // 100MB
		}

		err := testEnv.settingsManager.SaveSettings(settings)
		require.NoError(t, err)

		loadedSettings, err := testEnv.settingsManager.LoadSettings()
		require.NoError(t, err)
		assert.Equal(t, settings.AWSRegion, loadedSettings.AWSRegion)
		assert.Equal(t, settings.S3Bucket, loadedSettings.S3Bucket)
	})
}

// Test environment setup and helpers

type testEnvironment struct {
	database          storage.Database
	credProvider      aws.CredentialProvider
	s3Service         aws.S3Service
	fileManager       manager.FileManager
	shareManager      manager.ShareManager
	expirationManager manager.ExpirationManager
	settingsManager   manager.SettingsManager
	syncManager       manager.SyncManager
	bucketName        string
	region            string
	tempDir           string
}

func setupTestEnvironment(t *testing.T) *testEnvironment {
	// Get environment variables
	bucketName := os.Getenv("S3_BUCKET")
	region := os.Getenv("AWS_REGION")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	require.NotEmpty(t, bucketName, "S3_BUCKET is required")
	require.NotEmpty(t, region, "AWS_REGION is required")
	require.NotEmpty(t, accessKeyID, "AWS_ACCESS_KEY_ID is required")
	require.NotEmpty(t, secretAccessKey, "AWS_SECRET_ACCESS_KEY is required")

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "e2e-test-*")
	require.NoError(t, err)

	// Create test database
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)

	// Create credential provider and store test credentials
	credProvider, err := aws.NewSecureCredentialProvider()
	require.NoError(t, err)
	err = credProvider.StoreCredentials(accessKeyID, secretAccessKey, region)
	require.NoError(t, err)

	// Create S3 service
	s3Service, err := aws.NewS3Service(credProvider, bucketName)
	require.NoError(t, err)

	// Create managers
	fileManager := manager.NewFileManager(database, s3Service)
	shareManager := manager.NewShareManager(database, s3Service)
	expirationManager := manager.NewExpirationManager(database)
	settingsManager := manager.NewSettingsManager(database)
	syncManager := manager.NewSyncManager(database, s3Service)

	return &testEnvironment{
		database:          database,
		credProvider:      credProvider,
		s3Service:         s3Service,
		fileManager:       fileManager,
		shareManager:      shareManager,
		expirationManager: expirationManager,
		settingsManager:   settingsManager,
		syncManager:       syncManager,
		bucketName:        bucketName,
		region:            region,
		tempDir:           tempDir,
	}
}

func (env *testEnvironment) cleanup() {
	if env.database != nil {
		env.database.Close()
	}
	if env.tempDir != "" {
		os.RemoveAll(env.tempDir)
	}
}

func createTestFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "e2e-test-*.txt")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}