package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/storage"
)

// MockS3Service implements aws.S3Service for testing
type MockS3Service struct {
	generatePresignedURLFunc func(ctx context.Context, key string, expiration time.Duration) (string, error)
	uploadFileFunc           func(ctx context.Context, key string, filePath string, metadata map[string]string, progressCh chan<- aws.UploadProgress) error
	deleteObjectFunc         func(ctx context.Context, key string) error
	headObjectFunc           func(ctx context.Context, key string) (*s3.HeadObjectOutput, error)
	testConnectionFunc       func(ctx context.Context) error
}

func (m *MockS3Service) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if m.generatePresignedURLFunc != nil {
		return m.generatePresignedURLFunc(ctx, key, expiration)
	}
	return fmt.Sprintf("https://test-bucket.s3.amazonaws.com/%s?expires=%d", key, int64(expiration.Seconds())), nil
}

func (m *MockS3Service) UploadFile(ctx context.Context, key string, filePath string, metadata map[string]string, progressCh chan<- aws.UploadProgress) error {
	if m.uploadFileFunc != nil {
		return m.uploadFileFunc(ctx, key, filePath, metadata, progressCh)
	}
	return nil
}

func (m *MockS3Service) DeleteObject(ctx context.Context, key string) error {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, key)
	}
	return nil
}

func (m *MockS3Service) HeadObject(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	if m.headObjectFunc != nil {
		return m.headObjectFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockS3Service) TestConnection(ctx context.Context) error {
	if m.testConnectionFunc != nil {
		return m.testConnectionFunc(ctx)
	}
	return nil
}

// createShareTestDatabase creates a temporary SQLite database for testing
func createShareTestDatabase(t *testing.T) storage.Database {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})
	
	return db
}

// createTestFileRecord creates a test file record in the database
func createTestFileRecord(t *testing.T, db storage.Database, status storage.FileStatus, expirationDate time.Time) *storage.FileMetadata {
	file := &storage.FileMetadata{
		ID:             uuid.New().String(),
		FileName:       "test.txt",
		FilePath:       "/tmp/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: expirationDate,
		S3Key:          "test-key-" + uuid.New().String(),
		Status:         status,
	}
	
	err := db.SaveFile(file)
	require.NoError(t, err)
	
	return file
}

func TestNewShareManager(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	
	sm := NewShareManager(db, s3Service)
	
	assert.NotNil(t, sm)
	assert.Equal(t, db, sm.db)
	assert.Equal(t, s3Service, sm.s3Service)
}

func TestShareManager_ShareFile_Success(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file that expires in 1 hour
	expirationDate := time.Now().Add(1 * time.Hour)
	file := createTestFileRecord(t, db, storage.StatusActive, expirationDate)
	
	ctx := context.Background()
	recipients := []string{"test@example.com", "user@domain.org"}
	message := "Please review this file"
	
	shareRecord, err := sm.ShareFile(ctx, file.ID, recipients, message)
	
	assert.NoError(t, err)
	assert.NotNil(t, shareRecord)
	assert.NotEmpty(t, shareRecord.ID)
	assert.Equal(t, file.ID, shareRecord.FileID)
	assert.Equal(t, recipients, shareRecord.Recipients)
	assert.Equal(t, message, shareRecord.Message)
	assert.NotEmpty(t, shareRecord.PresignedURL)
	assert.True(t, shareRecord.SharedDate.Before(time.Now().Add(1*time.Second)))
	assert.True(t, shareRecord.URLExpiration.After(time.Now()))
	
	// Verify share record was saved to database
	shares, err := db.GetShareHistory(file.ID)
	assert.NoError(t, err)
	assert.Len(t, shares, 1)
	assert.Equal(t, shareRecord.ID, shares[0].ID)
}

func TestShareManager_ShareFile_EmptyFileID(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	ctx := context.Background()
	recipients := []string{"test@example.com"}
	message := "Test message"
	
	shareRecord, err := sm.ShareFile(ctx, "", recipients, message)
	
	assert.Error(t, err)
	assert.Nil(t, shareRecord)
	assert.Contains(t, err.Error(), "file ID cannot be empty")
}

func TestShareManager_ShareFile_NoRecipients(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	ctx := context.Background()
	recipients := []string{}
	message := "Test message"
	
	shareRecord, err := sm.ShareFile(ctx, "test-file-id", recipients, message)
	
	assert.Error(t, err)
	assert.Nil(t, shareRecord)
	assert.Contains(t, err.Error(), "at least one recipient must be specified")
}

func TestShareManager_ShareFile_InvalidEmail(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	ctx := context.Background()
	recipients := []string{"invalid-email"}
	message := "Test message"
	
	shareRecord, err := sm.ShareFile(ctx, "test-file-id", recipients, message)
	
	assert.Error(t, err)
	assert.Nil(t, shareRecord)
	assert.Contains(t, err.Error(), "invalid recipient email")
}

func TestShareManager_ShareFile_FileNotFound(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	ctx := context.Background()
	recipients := []string{"test@example.com"}
	message := "Test message"
	
	shareRecord, err := sm.ShareFile(ctx, "non-existent-file", recipients, message)
	
	assert.Error(t, err)
	assert.Nil(t, shareRecord)
	assert.Contains(t, err.Error(), "failed to get file metadata")
}

func TestShareManager_ShareFile_InactiveFile(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file with expired status
	expirationDate := time.Now().Add(1 * time.Hour)
	file := createTestFileRecord(t, db, storage.StatusExpired, expirationDate)
	
	ctx := context.Background()
	recipients := []string{"test@example.com"}
	message := "Test message"
	
	shareRecord, err := sm.ShareFile(ctx, file.ID, recipients, message)
	
	assert.Error(t, err)
	assert.Nil(t, shareRecord)
	assert.Contains(t, err.Error(), "cannot share file with status: expired")
}

func TestShareManager_ShareFile_ExpiredFile(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file that expired 1 hour ago
	expirationDate := time.Now().Add(-1 * time.Hour)
	file := createTestFileRecord(t, db, storage.StatusActive, expirationDate)
	
	ctx := context.Background()
	recipients := []string{"test@example.com"}
	message := "Test message"
	
	shareRecord, err := sm.ShareFile(ctx, file.ID, recipients, message)
	
	assert.Error(t, err)
	assert.Nil(t, shareRecord)
	assert.Contains(t, err.Error(), "cannot share expired file")
}

func TestShareManager_ShareFile_S3Error(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{
		generatePresignedURLFunc: func(ctx context.Context, key string, expiration time.Duration) (string, error) {
			return "", fmt.Errorf("S3 service error")
		},
	}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file that expires in 1 hour
	expirationDate := time.Now().Add(1 * time.Hour)
	file := createTestFileRecord(t, db, storage.StatusActive, expirationDate)
	
	ctx := context.Background()
	recipients := []string{"test@example.com"}
	message := "Test message"
	
	shareRecord, err := sm.ShareFile(ctx, file.ID, recipients, message)
	
	assert.Error(t, err)
	assert.Nil(t, shareRecord)
	assert.Contains(t, err.Error(), "failed to generate presigned URL")
}

func TestShareManager_GetShareHistory_Success(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file
	expirationDate := time.Now().Add(1 * time.Hour)
	file := createTestFileRecord(t, db, storage.StatusActive, expirationDate)
	
	// Create multiple share records
	ctx := context.Background()
	
	share1, err := sm.ShareFile(ctx, file.ID, []string{"user1@example.com"}, "First share")
	require.NoError(t, err)
	
	share2, err := sm.ShareFile(ctx, file.ID, []string{"user2@example.com"}, "Second share")
	require.NoError(t, err)
	
	// Get share history
	shares, err := sm.GetShareHistory(file.ID)
	
	assert.NoError(t, err)
	assert.Len(t, shares, 2)
	
	// Shares should be ordered by shared_date DESC
	assert.Equal(t, share2.ID, shares[0].ID) // Most recent first
	assert.Equal(t, share1.ID, shares[1].ID)
}

func TestShareManager_GetShareHistory_EmptyFileID(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	shares, err := sm.GetShareHistory("")
	
	assert.Error(t, err)
	assert.Nil(t, shares)
	assert.Contains(t, err.Error(), "file ID cannot be empty")
}

func TestShareManager_GetShareHistory_NoShares(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	shares, err := sm.GetShareHistory("non-existent-file")
	
	assert.NoError(t, err)
	assert.Empty(t, shares)
}

func TestShareManager_RevokeShare(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	err := sm.RevokeShare("test-share-id")
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "share revocation not implemented")
}

func TestShareManager_RevokeShare_EmptyShareID(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	err := sm.RevokeShare("")
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "share ID cannot be empty")
}

func TestShareManager_GeneratePresignedURL_Success(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file that expires in 2 hours
	expirationDate := time.Now().Add(2 * time.Hour)
	file := createTestFileRecord(t, db, storage.StatusActive, expirationDate)
	
	ctx := context.Background()
	expiration := 1 * time.Hour
	
	url, err := sm.GeneratePresignedURL(ctx, file.ID, expiration)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, file.S3Key)
}

func TestShareManager_GeneratePresignedURL_EmptyFileID(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	ctx := context.Background()
	expiration := 1 * time.Hour
	
	url, err := sm.GeneratePresignedURL(ctx, "", expiration)
	
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "file ID cannot be empty")
}

func TestShareManager_GeneratePresignedURL_InvalidExpiration(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	ctx := context.Background()
	expiration := -1 * time.Hour
	
	url, err := sm.GeneratePresignedURL(ctx, "test-file-id", expiration)
	
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "expiration duration must be positive")
}

func TestShareManager_GeneratePresignedURL_FileNotFound(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	ctx := context.Background()
	expiration := 1 * time.Hour
	
	url, err := sm.GeneratePresignedURL(ctx, "non-existent-file", expiration)
	
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "failed to get file metadata")
}

func TestShareManager_GeneratePresignedURL_InactiveFile(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file with expired status
	expirationDate := time.Now().Add(1 * time.Hour)
	file := createTestFileRecord(t, db, storage.StatusExpired, expirationDate)
	
	ctx := context.Background()
	expiration := 1 * time.Hour
	
	url, err := sm.GeneratePresignedURL(ctx, file.ID, expiration)
	
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "cannot generate URL for file with status: expired")
}

func TestShareManager_GeneratePresignedURL_ExpiredFile(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file that expired 1 hour ago
	expirationDate := time.Now().Add(-1 * time.Hour)
	file := createTestFileRecord(t, db, storage.StatusActive, expirationDate)
	
	ctx := context.Background()
	expiration := 1 * time.Hour
	
	url, err := sm.GeneratePresignedURL(ctx, file.ID, expiration)
	
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "cannot generate URL for expired file")
}

func TestShareManager_GeneratePresignedURL_ExpirationLimitedByFile(t *testing.T) {
	db := createShareTestDatabase(t)
	s3Service := &MockS3Service{
		generatePresignedURLFunc: func(ctx context.Context, key string, expiration time.Duration) (string, error) {
			// Verify that expiration was limited by file expiration
			assert.True(t, expiration <= 30*time.Minute, "Expiration should be limited by file expiration")
			return "https://test-url.com", nil
		},
	}
	sm := NewShareManager(db, s3Service)
	
	// Create a test file that expires in 30 minutes
	expirationDate := time.Now().Add(30 * time.Minute)
	file := createTestFileRecord(t, db, storage.StatusActive, expirationDate)
	
	ctx := context.Background()
	expiration := 2 * time.Hour // Request 2 hours but should be limited to 30 minutes
	
	url, err := sm.GeneratePresignedURL(ctx, file.ID, expiration)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, url)
}

// Test helper functions

func TestValidateEmail_Valid(t *testing.T) {
	validEmails := []string{
		"test@example.com",
		"user.name@domain.org",
		"user+tag@example.co.uk",
		"123@456.789",
	}
	
	for _, email := range validEmails {
		t.Run(email, func(t *testing.T) {
			err := validateEmail(email)
			assert.NoError(t, err)
		})
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	invalidEmails := []struct {
		email string
		error string
	}{
		{"", "email cannot be empty"},
		{"no-at-symbol", "email must contain @ symbol"},
		{"@domain.com", "email must have local part before @"},
		{"user@", "email must have domain part after @"},
		{"user@@domain.com", "email contains multiple @ symbols"},
		{"user@domain", "email domain must contain at least one dot"},
	}
	
	for _, test := range invalidEmails {
		t.Run(test.email, func(t *testing.T) {
			err := validateEmail(test.email)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.error)
		})
	}
}

func TestCalculateURLExpiration(t *testing.T) {
	now := time.Now()
	
	// Test case 1: File expires after default expiration (24 hours)
	fileExpiration := now.Add(48 * time.Hour)
	urlExpiration := calculateURLExpiration(fileExpiration)
	expectedExpiration := now.Add(24 * time.Hour)
	
	// Allow for small time differences due to execution time
	assert.True(t, urlExpiration.Sub(expectedExpiration) < 1*time.Second)
	
	// Test case 2: File expires before default expiration
	fileExpiration = now.Add(2 * time.Hour)
	urlExpiration = calculateURLExpiration(fileExpiration)
	
	assert.Equal(t, fileExpiration, urlExpiration)
}