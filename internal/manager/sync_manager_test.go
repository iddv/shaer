package manager

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
)

// MockS3ServiceSync for testing sync manager (to avoid conflicts with other tests)
type MockS3ServiceSync struct {
	mock.Mock
}

func (m *MockS3ServiceSync) UploadFile(ctx context.Context, key string, filePath string, metadata map[string]string, progressCh chan<- aws.UploadProgress) error {
	args := m.Called(ctx, key, filePath, metadata, progressCh)
	return args.Error(0)
}

func (m *MockS3ServiceSync) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, key, expiration)
	return args.String(0), args.Error(1)
}

func (m *MockS3ServiceSync) DeleteObject(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockS3ServiceSync) HeadObject(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *MockS3ServiceSync) TestConnection(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}



func TestNewSyncManager(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	// Create mock S3 service
	mockS3 := &MockS3ServiceSync{}

	// Test creating sync manager with S3 service
	syncManager := NewSyncManager(db, mockS3)
	assert.NotNil(t, syncManager)
	assert.False(t, syncManager.IsOfflineMode())

	// Test creating sync manager without S3 service
	syncManagerOffline := NewSyncManagerWithoutS3(db)
	assert.NotNil(t, syncManagerOffline)
	assert.True(t, syncManagerOffline.IsOfflineMode())
}

func TestSyncManager_OfflineMode(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	// Create mock S3 service
	mockS3 := &MockS3ServiceSync{}

	syncManager := NewSyncManager(db, mockS3)

	// Test initial state
	assert.False(t, syncManager.IsOfflineMode())

	// Test setting offline mode
	syncManager.SetOfflineMode(true)
	assert.True(t, syncManager.IsOfflineMode())

	// Test setting online mode
	syncManager.SetOfflineMode(false)
	assert.False(t, syncManager.IsOfflineMode())
}

func TestSyncManager_SyncWithS3_OfflineMode(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	// Create sync manager without S3 service (offline mode)
	syncManager := NewSyncManagerWithoutS3(db)

	ctx := context.Background()
	result, err := syncManager.SyncWithS3(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.OfflineMode)
	assert.Equal(t, 0, result.TotalFiles)
	assert.Equal(t, 0, result.VerifiedFiles)
	assert.Equal(t, 0, result.MissingFiles)
	assert.Equal(t, 0, result.ErrorFiles)
}

func TestSyncManager_SyncWithS3_ConnectionFailure(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	// Create mock S3 service that fails connection test
	mockS3 := &MockS3ServiceSync{}
	mockS3.On("TestConnection", mock.Anything).Return(fmt.Errorf("connection failed"))

	syncManager := NewSyncManager(db, mockS3)

	ctx := context.Background()
	result, err := syncManager.SyncWithS3(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "S3 connection failed")
	assert.NotNil(t, result)
	assert.True(t, result.OfflineMode)
	assert.True(t, syncManager.IsOfflineMode())

	mockS3.AssertExpectations(t)
}

func TestSyncManager_SyncWithS3_Success(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	// Add test files to database
	testFile1 := &storage.FileMetadata{
		ID:             "test-file-1",
		FileName:       "test1.txt",
		FilePath:       "/tmp/test1.txt",
		FileSize:       100,
		UploadDate:     time.Now().Add(-1 * time.Hour),
		ExpirationDate: time.Now().Add(1 * time.Hour),
		S3Key:          "uploads/2024/01/01/test1.txt",
		Status:         storage.StatusActive,
	}

	testFile2 := &storage.FileMetadata{
		ID:             "test-file-2",
		FileName:       "test2.txt",
		FilePath:       "/tmp/test2.txt",
		FileSize:       200,
		UploadDate:     time.Now().Add(-2 * time.Hour),
		ExpirationDate: time.Now().Add(-30 * time.Minute), // Expired
		S3Key:          "uploads/2024/01/01/test2.txt",
		Status:         storage.StatusActive,
	}

	err = db.SaveFile(testFile1)
	assert.NoError(t, err)
	err = db.SaveFile(testFile2)
	assert.NoError(t, err)

	// Create mock S3 service
	mockS3 := &MockS3ServiceSync{}
	mockS3.On("TestConnection", mock.Anything).Return(nil)
	
	// Mock HeadObject calls - file1 exists, file2 exists but expired
	mockS3.On("HeadObject", mock.Anything, "uploads/2024/01/01/test1.txt").Return(&s3.HeadObjectOutput{}, nil)
	mockS3.On("HeadObject", mock.Anything, "uploads/2024/01/01/test2.txt").Return(&s3.HeadObjectOutput{}, nil)

	syncManager := NewSyncManager(db, mockS3)

	ctx := context.Background()
	result, err := syncManager.SyncWithS3(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.OfflineMode)
	assert.Equal(t, 2, result.TotalFiles)
	assert.Equal(t, 2, result.VerifiedFiles)
	assert.Equal(t, 0, result.MissingFiles)
	assert.Equal(t, 0, result.ErrorFiles)
	assert.Contains(t, result.UpdatedFiles, "test-file-2") // File2 should be updated to expired

	// Verify file2 status was updated to expired
	updatedFile2, err := db.GetFile("test-file-2")
	assert.NoError(t, err)
	assert.Equal(t, storage.StatusExpired, updatedFile2.Status)

	mockS3.AssertExpectations(t)
}

func TestSyncManager_SyncWithS3_MissingFiles(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	// Add test file to database
	testFile := &storage.FileMetadata{
		ID:             "test-file-missing",
		FileName:       "missing.txt",
		FilePath:       "/tmp/missing.txt",
		FileSize:       100,
		UploadDate:     time.Now().Add(-1 * time.Hour),
		ExpirationDate: time.Now().Add(1 * time.Hour),
		S3Key:          "uploads/2024/01/01/missing.txt",
		Status:         storage.StatusActive,
	}

	err = db.SaveFile(testFile)
	assert.NoError(t, err)

	// Create mock S3 service
	mockS3 := &MockS3ServiceSync{}
	mockS3.On("TestConnection", mock.Anything).Return(nil)
	
	// Mock HeadObject call - file not found
	mockS3.On("HeadObject", mock.Anything, "uploads/2024/01/01/missing.txt").Return(nil, fmt.Errorf("NoSuchKey: The specified key does not exist"))

	syncManager := NewSyncManager(db, mockS3)

	ctx := context.Background()
	result, err := syncManager.SyncWithS3(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.OfflineMode)
	assert.Equal(t, 1, result.TotalFiles)
	assert.Equal(t, 0, result.VerifiedFiles)
	assert.Equal(t, 1, result.MissingFiles)
	assert.Equal(t, 0, result.ErrorFiles)
	assert.Contains(t, result.MissingFileIDs, "test-file-missing")
	assert.Contains(t, result.UpdatedFiles, "test-file-missing")

	// Verify file status was updated to deleted
	updatedFile, err := db.GetFile("test-file-missing")
	assert.NoError(t, err)
	assert.Equal(t, storage.StatusDeleted, updatedFile.Status)

	mockS3.AssertExpectations(t)
}

func TestSyncManager_VerifyFileExists_OfflineMode(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	// Add test file to database
	testFile := &storage.FileMetadata{
		ID:             "test-file-offline",
		FileName:       "offline.txt",
		FilePath:       "/tmp/offline.txt",
		FileSize:       100,
		UploadDate:     time.Now().Add(-1 * time.Hour),
		ExpirationDate: time.Now().Add(1 * time.Hour),
		S3Key:          "uploads/2024/01/01/offline.txt",
		Status:         storage.StatusActive,
	}

	err = db.SaveFile(testFile)
	assert.NoError(t, err)

	// Create sync manager in offline mode
	syncManager := NewSyncManagerWithoutS3(db)

	ctx := context.Background()
	result, err := syncManager.VerifyFileExists(ctx, "test-file-offline")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-file-offline", result.FileID)
	assert.True(t, result.Exists) // Should assume active files exist in offline mode
	assert.Equal(t, models.StatusActive, result.OldStatus)
	assert.Equal(t, models.StatusActive, result.NewStatus)
	assert.Nil(t, result.Error)
}

func TestSyncManager_VerifyFileExists_Online(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	// Add test file to database
	testFile := &storage.FileMetadata{
		ID:             "test-file-online",
		FileName:       "online.txt",
		FilePath:       "/tmp/online.txt",
		FileSize:       100,
		UploadDate:     time.Now().Add(-1 * time.Hour),
		ExpirationDate: time.Now().Add(1 * time.Hour),
		S3Key:          "uploads/2024/01/01/online.txt",
		Status:         storage.StatusActive,
	}

	err = db.SaveFile(testFile)
	assert.NoError(t, err)

	// Create mock S3 service
	mockS3 := &MockS3ServiceSync{}
	mockS3.On("HeadObject", mock.Anything, "uploads/2024/01/01/online.txt").Return(&s3.HeadObjectOutput{}, nil)

	syncManager := NewSyncManager(db, mockS3)

	ctx := context.Background()
	result, err := syncManager.VerifyFileExists(ctx, "test-file-online")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-file-online", result.FileID)
	assert.True(t, result.Exists)
	assert.Equal(t, models.StatusActive, result.OldStatus)
	assert.Equal(t, models.StatusActive, result.NewStatus)
	assert.Nil(t, result.Error)

	mockS3.AssertExpectations(t)
}

func TestSyncManager_GetLastSyncTime(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := storage.NewSQLiteDatabase(dbPath)
	assert.NoError(t, err)
	defer db.Close()

	syncManager := NewSyncManagerWithoutS3(db)

	// Test when no sync time is saved
	_, err = syncManager.GetLastSyncTime()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Save a sync time
	testTime := time.Now().Truncate(time.Second)
	err = db.SaveConfig("last_sync_time", testTime.Format(time.RFC3339))
	assert.NoError(t, err)

	// Test retrieving sync time
	retrievedTime, err := syncManager.GetLastSyncTime()
	assert.NoError(t, err)
	assert.Equal(t, testTime, retrievedTime)
}

func TestSyncManager_ErrorCategorization(t *testing.T) {
	tests := []struct {
		name     string
		error    string
		expected string
	}{
		{"Access Denied", "AccessDenied: Access denied", "access_denied"},
		{"Not Found", "NoSuchKey: The specified key does not exist", "not_found"},
		{"Network Timeout", "connection timeout", "network"},
		{"Unknown Error", "some random error", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(fmt.Errorf(tt.error))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSyncManager_IsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected bool
	}{
		{"NoSuchKey Error", fmt.Errorf("NoSuchKey: The specified key does not exist"), true},
		{"404 Error", fmt.Errorf("404 not found"), true},
		{"Not Found Error", fmt.Errorf("file not found"), true},
		{"Access Denied", fmt.Errorf("AccessDenied: Access denied"), false},
		{"Network Error", fmt.Errorf("connection timeout"), false},
		{"Nil Error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.error)
			assert.Equal(t, tt.expected, result)
		})
	}
}