package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
)

// createTempDatabase creates a temporary SQLite database for testing
func createTempDatabase(t *testing.T) (*storage.SQLiteDatabase, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	
	return db, dbPath
}

// mockS3Service implements aws.S3Service for testing
type mockS3Service struct {
	shouldError bool
	errorMsg    string
	uploadedFiles map[string]bool
}

func newMockS3Service() *mockS3Service {
	return &mockS3Service{
		uploadedFiles: make(map[string]bool),
	}
}

func (m *mockS3Service) UploadFile(ctx context.Context, key string, filePath string, metadata map[string]string, progressCh chan<- aws.UploadProgress) error {
	if m.shouldError {
		return fmt.Errorf(m.errorMsg)
	}
	
	// Simulate progress updates
	if progressCh != nil {
		select {
		case progressCh <- aws.UploadProgress{BytesUploaded: 50, TotalBytes: 100, Percentage: 50.0}:
		case <-ctx.Done():
			return ctx.Err()
		}
		
		select {
		case progressCh <- aws.UploadProgress{BytesUploaded: 100, TotalBytes: 100, Percentage: 100.0}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	m.uploadedFiles[key] = true
	return nil
}

func (m *mockS3Service) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if m.shouldError {
		return "", fmt.Errorf(m.errorMsg)
	}
	return fmt.Sprintf("https://test-bucket.s3.amazonaws.com/%s?expires=%d", key, expiration), nil
}

func (m *mockS3Service) DeleteObject(ctx context.Context, key string) error {
	if m.shouldError {
		return fmt.Errorf(m.errorMsg)
	}
	delete(m.uploadedFiles, key)
	return nil
}

func (m *mockS3Service) HeadObject(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	if m.shouldError {
		return nil, fmt.Errorf(m.errorMsg)
	}
	return nil, nil
}

func (m *mockS3Service) TestConnection(ctx context.Context) error {
	if m.shouldError {
		return fmt.Errorf(m.errorMsg)
	}
	return nil
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

func TestNewFileManager(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	assert.NotNil(t, fm)
	assert.IsType(t, &FileManagerImpl{}, fm)
}

func TestFileManager_SaveFile(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	tests := []struct {
		name        string
		file        *models.FileMetadata
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid file",
			file: &models.FileMetadata{
				ID:             "test-id-1",
				FileName:       "test.txt",
				FilePath:       "/tmp/test.txt",
				FileSize:       1024,
				UploadDate:     time.Now(),
				ExpirationDate: time.Now().Add(24 * time.Hour),
				S3Key:          "uploads/test.txt",
				Status:         models.StatusActive,
			},
			expectError: false,
		},
		{
			name:        "nil file",
			file:        nil,
			expectError: true,
			errorMsg:    "file metadata cannot be nil",
		},
		{
			name: "empty ID",
			file: &models.FileMetadata{
				FileName: "test.txt",
				S3Key:    "uploads/test.txt",
			},
			expectError: true,
			errorMsg:    "file ID cannot be empty",
		},
		{
			name: "empty filename",
			file: &models.FileMetadata{
				ID:    "test-id",
				S3Key: "uploads/test.txt",
			},
			expectError: true,
			errorMsg:    "file name cannot be empty",
		},
		{
			name: "empty S3 key",
			file: &models.FileMetadata{
				ID:       "test-id",
				FileName: "test.txt",
			},
			expectError: true,
			errorMsg:    "S3 key cannot be empty",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fm.SaveFile(tt.file)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileManager_GetFile(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	// Save a test file first
	testFile := &models.FileMetadata{
		ID:             "test-id-1",
		FileName:       "test.txt",
		FilePath:       "/tmp/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test.txt",
		Status:         models.StatusActive,
	}
	
	err := fm.SaveFile(testFile)
	require.NoError(t, err)
	
	tests := []struct {
		name        string
		fileID      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid file ID",
			fileID:      "test-id-1",
			expectError: false,
		},
		{
			name:        "empty file ID",
			fileID:      "",
			expectError: true,
			errorMsg:    "file ID cannot be empty",
		},
		{
			name:        "non-existent file ID",
			fileID:      "non-existent",
			expectError: true,
			errorMsg:    "file not found",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := fm.GetFile(tt.fileID)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, file)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, file)
				assert.Equal(t, testFile.ID, file.ID)
				assert.Equal(t, testFile.FileName, file.FileName)
				assert.Equal(t, testFile.Status, file.Status)
			}
		})
	}
}

func TestFileManager_ListFiles(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	// Test empty list
	files, err := fm.ListFiles()
	assert.NoError(t, err)
	assert.Empty(t, files)
	
	// Add test files
	testFiles := []*models.FileMetadata{
		{
			ID:             "test-id-1",
			FileName:       "test1.txt",
			FilePath:       "/tmp/test1.txt",
			FileSize:       1024,
			UploadDate:     time.Now(),
			ExpirationDate: time.Now().Add(24 * time.Hour),
			S3Key:          "uploads/test1.txt",
			Status:         models.StatusActive,
		},
		{
			ID:             "test-id-2",
			FileName:       "test2.txt",
			FilePath:       "/tmp/test2.txt",
			FileSize:       2048,
			UploadDate:     time.Now(),
			ExpirationDate: time.Now().Add(48 * time.Hour),
			S3Key:          "uploads/test2.txt",
			Status:         models.StatusUploading,
		},
	}
	
	for _, file := range testFiles {
		err := fm.SaveFile(file)
		require.NoError(t, err)
	}
	
	// Test list with files
	files, err = fm.ListFiles()
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	
	// Verify files are returned (order may vary due to database ordering)
	fileIDs := make(map[string]bool)
	for _, file := range files {
		fileIDs[file.ID] = true
	}
	
	assert.True(t, fileIDs["test-id-1"])
	assert.True(t, fileIDs["test-id-2"])
}

func TestFileManager_UpdateFileStatus(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	// Save a test file first
	testFile := &models.FileMetadata{
		ID:             "test-id-1",
		FileName:       "test.txt",
		FilePath:       "/tmp/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test.txt",
		Status:         models.StatusUploading,
	}
	
	err := fm.SaveFile(testFile)
	require.NoError(t, err)
	
	tests := []struct {
		name        string
		fileID      string
		status      models.FileStatus
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid status update",
			fileID:      "test-id-1",
			status:      models.StatusActive,
			expectError: false,
		},
		{
			name:        "empty file ID",
			fileID:      "",
			status:      models.StatusActive,
			expectError: true,
			errorMsg:    "file ID cannot be empty",
		},
		{
			name:        "non-existent file ID",
			fileID:      "non-existent",
			status:      models.StatusActive,
			expectError: true,
			errorMsg:    "file not found",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fm.UpdateFileStatus(tt.fileID, tt.status)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				
				// Verify the status was updated
				file, err := fm.GetFile(tt.fileID)
				assert.NoError(t, err)
				assert.Equal(t, tt.status, file.Status)
			}
		})
	}
}

func TestFileManager_DeleteFile(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	// Save a test file first
	testFile := &models.FileMetadata{
		ID:             "test-id-1",
		FileName:       "test.txt",
		FilePath:       "/tmp/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test.txt",
		Status:         models.StatusActive,
	}
	
	err := fm.SaveFile(testFile)
	require.NoError(t, err)
	
	tests := []struct {
		name        string
		fileID      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid file deletion",
			fileID:      "test-id-1",
			expectError: false,
		},
		{
			name:        "empty file ID",
			fileID:      "",
			expectError: true,
			errorMsg:    "file ID cannot be empty",
		},
		{
			name:        "non-existent file ID",
			fileID:      "non-existent",
			expectError: true,
			errorMsg:    "file not found",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fm.DeleteFile(tt.fileID)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				
				// Verify the file was deleted
				_, err := fm.GetFile(tt.fileID)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "file not found")
			}
		})
	}
}

func TestFileManager_CreateFileRecord(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	expirationDate := time.Now().Add(24 * time.Hour)
	
	tests := []struct {
		name           string
		fileName       string
		filePath       string
		fileSize       int64
		s3Key          string
		expirationDate time.Time
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "valid file record",
			fileName:       "test.txt",
			filePath:       "/tmp/test.txt",
			fileSize:       1024,
			s3Key:          "uploads/test.txt",
			expirationDate: expirationDate,
			expectError:    false,
		},
		{
			name:           "empty file name",
			fileName:       "",
			filePath:       "/tmp/test.txt",
			fileSize:       1024,
			s3Key:          "uploads/test.txt",
			expirationDate: expirationDate,
			expectError:    true,
			errorMsg:       "file name cannot be empty",
		},
		{
			name:           "empty file path",
			fileName:       "test.txt",
			filePath:       "",
			fileSize:       1024,
			s3Key:          "uploads/test.txt",
			expirationDate: expirationDate,
			expectError:    true,
			errorMsg:       "file path cannot be empty",
		},
		{
			name:           "invalid file size",
			fileName:       "test.txt",
			filePath:       "/tmp/test.txt",
			fileSize:       0,
			s3Key:          "uploads/test.txt",
			expirationDate: expirationDate,
			expectError:    true,
			errorMsg:       "file size must be greater than 0",
		},
		{
			name:           "empty S3 key",
			fileName:       "test.txt",
			filePath:       "/tmp/test.txt",
			fileSize:       1024,
			s3Key:          "",
			expirationDate: expirationDate,
			expectError:    true,
			errorMsg:       "S3 key cannot be empty",
		},
		{
			name:           "zero expiration date",
			fileName:       "test.txt",
			filePath:       "/tmp/test.txt",
			fileSize:       1024,
			s3Key:          "uploads/test.txt",
			expirationDate: time.Time{},
			expectError:    true,
			errorMsg:       "expiration date cannot be zero",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := fm.CreateFileRecord(tt.fileName, tt.filePath, tt.fileSize, tt.s3Key, tt.expirationDate)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, file)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, file)
				assert.NotEmpty(t, file.ID)
				assert.Equal(t, tt.fileName, file.FileName)
				assert.Equal(t, tt.filePath, file.FilePath)
				assert.Equal(t, tt.fileSize, file.FileSize)
				assert.Equal(t, tt.s3Key, file.S3Key)
				assert.Equal(t, tt.expirationDate, file.ExpirationDate)
				assert.Equal(t, models.StatusUploading, file.Status)
				assert.False(t, file.UploadDate.IsZero())
				
				// Verify the file was saved to the database
				savedFile, err := fm.GetFile(file.ID)
				assert.NoError(t, err)
				assert.Equal(t, file.ID, savedFile.ID)
			}
		})
	}
}

func TestFileManager_GetFilesByStatus(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	// Add test files with different statuses
	testFiles := []*models.FileMetadata{
		{
			ID:             "active-1",
			FileName:       "active1.txt",
			FilePath:       "/tmp/active1.txt",
			FileSize:       1024,
			UploadDate:     time.Now(),
			ExpirationDate: time.Now().Add(24 * time.Hour),
			S3Key:          "uploads/active1.txt",
			Status:         models.StatusActive,
		},
		{
			ID:             "active-2",
			FileName:       "active2.txt",
			FilePath:       "/tmp/active2.txt",
			FileSize:       2048,
			UploadDate:     time.Now(),
			ExpirationDate: time.Now().Add(24 * time.Hour),
			S3Key:          "uploads/active2.txt",
			Status:         models.StatusActive,
		},
		{
			ID:             "uploading-1",
			FileName:       "uploading1.txt",
			FilePath:       "/tmp/uploading1.txt",
			FileSize:       1024,
			UploadDate:     time.Now(),
			ExpirationDate: time.Now().Add(24 * time.Hour),
			S3Key:          "uploads/uploading1.txt",
			Status:         models.StatusUploading,
		},
		{
			ID:             "expired-1",
			FileName:       "expired1.txt",
			FilePath:       "/tmp/expired1.txt",
			FileSize:       1024,
			UploadDate:     time.Now(),
			ExpirationDate: time.Now().Add(24 * time.Hour),
			S3Key:          "uploads/expired1.txt",
			Status:         models.StatusExpired,
		},
	}
	
	for _, file := range testFiles {
		err := fm.SaveFile(file)
		require.NoError(t, err)
	}
	
	tests := []struct {
		name           string
		status         models.FileStatus
		expectedCount  int
		expectedIDs    []string
	}{
		{
			name:          "active files",
			status:        models.StatusActive,
			expectedCount: 2,
			expectedIDs:   []string{"active-1", "active-2"},
		},
		{
			name:          "uploading files",
			status:        models.StatusUploading,
			expectedCount: 1,
			expectedIDs:   []string{"uploading-1"},
		},
		{
			name:          "expired files",
			status:        models.StatusExpired,
			expectedCount: 1,
			expectedIDs:   []string{"expired-1"},
		},
		{
			name:          "deleted files",
			status:        models.StatusDeleted,
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := fm.GetFilesByStatus(tt.status)
			assert.NoError(t, err)
			assert.Len(t, files, tt.expectedCount)
			
			if tt.expectedCount > 0 {
				fileIDs := make([]string, len(files))
				for i, file := range files {
					fileIDs[i] = file.ID
					assert.Equal(t, tt.status, file.Status)
				}
				
				for _, expectedID := range tt.expectedIDs {
					assert.Contains(t, fileIDs, expectedID)
				}
			}
		})
	}
}

func TestFileManager_GetExpiredFiles(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)
	
	// Add test files with different expiration dates and statuses
	testFiles := []*models.FileMetadata{
		{
			ID:             "expired-active",
			FileName:       "expired-active.txt",
			FilePath:       "/tmp/expired-active.txt",
			FileSize:       1024,
			UploadDate:     now,
			ExpirationDate: pastTime, // Expired
			S3Key:          "uploads/expired-active.txt",
			Status:         models.StatusActive, // Should be included
		},
		{
			ID:             "expired-uploading",
			FileName:       "expired-uploading.txt",
			FilePath:       "/tmp/expired-uploading.txt",
			FileSize:       1024,
			UploadDate:     now,
			ExpirationDate: pastTime, // Expired
			S3Key:          "uploads/expired-uploading.txt",
			Status:         models.StatusUploading, // Should be included
		},
		{
			ID:             "expired-already-expired",
			FileName:       "expired-already-expired.txt",
			FilePath:       "/tmp/expired-already-expired.txt",
			FileSize:       1024,
			UploadDate:     now,
			ExpirationDate: pastTime, // Expired
			S3Key:          "uploads/expired-already-expired.txt",
			Status:         models.StatusExpired, // Should NOT be included
		},
		{
			ID:             "expired-deleted",
			FileName:       "expired-deleted.txt",
			FilePath:       "/tmp/expired-deleted.txt",
			FileSize:       1024,
			UploadDate:     now,
			ExpirationDate: pastTime, // Expired
			S3Key:          "uploads/expired-deleted.txt",
			Status:         models.StatusDeleted, // Should NOT be included
		},
		{
			ID:             "not-expired",
			FileName:       "not-expired.txt",
			FilePath:       "/tmp/not-expired.txt",
			FileSize:       1024,
			UploadDate:     now,
			ExpirationDate: futureTime, // Not expired
			S3Key:          "uploads/not-expired.txt",
			Status:         models.StatusActive, // Should NOT be included
		},
	}
	
	for _, file := range testFiles {
		err := fm.SaveFile(file)
		require.NoError(t, err)
	}
	
	// Test getting expired files
	expiredFiles, err := fm.GetExpiredFiles()
	assert.NoError(t, err)
	assert.Len(t, expiredFiles, 2)
	
	// Verify the correct files are returned
	expectedIDs := []string{"expired-active", "expired-uploading"}
	actualIDs := make([]string, len(expiredFiles))
	for i, file := range expiredFiles {
		actualIDs[i] = file.ID
		assert.True(t, file.ExpirationDate.Before(now))
		assert.NotEqual(t, models.StatusExpired, file.Status)
		assert.NotEqual(t, models.StatusDeleted, file.Status)
	}
	
	for _, expectedID := range expectedIDs {
		assert.Contains(t, actualIDs, expectedID)
	}
}

func TestFileManager_Integration(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	fm := NewFileManagerWithoutS3(db)
	
	// Test complete workflow
	expirationDate := time.Now().Add(24 * time.Hour)
	
	// 1. Create a file record
	file, err := fm.CreateFileRecord("integration-test.txt", "/tmp/integration-test.txt", 2048, "uploads/integration-test.txt", expirationDate)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.Equal(t, models.StatusUploading, file.Status)
	
	// 2. Update status to active
	err = fm.UpdateFileStatus(file.ID, models.StatusActive)
	assert.NoError(t, err)
	
	// 3. Verify status update
	updatedFile, err := fm.GetFile(file.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.StatusActive, updatedFile.Status)
	
	// 4. List files and verify it's included
	allFiles, err := fm.ListFiles()
	assert.NoError(t, err)
	assert.Len(t, allFiles, 1)
	assert.Equal(t, file.ID, allFiles[0].ID)
	
	// 5. Get files by status
	activeFiles, err := fm.GetFilesByStatus(models.StatusActive)
	assert.NoError(t, err)
	assert.Len(t, activeFiles, 1)
	assert.Equal(t, file.ID, activeFiles[0].ID)
	
	// 6. Delete the file
	err = fm.DeleteFile(file.ID)
	assert.NoError(t, err)
	
	// 7. Verify deletion
	_, err = fm.GetFile(file.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
	
	// 8. Verify empty list
	allFiles, err = fm.ListFiles()
	assert.NoError(t, err)
	assert.Empty(t, allFiles)
}

func TestFileManager_UploadFile(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	mockS3 := newMockS3Service()
	fm := NewFileManager(db, mockS3)
	
	ctx := context.Background()
	expiration := 24 * time.Hour
	
	tests := []struct {
		name           string
		setupFile      func(t *testing.T) string
		filePath       string
		expiration     time.Duration
		setupMock      func(*mockS3Service)
		expectError    bool
		expectedErrMsg string
		expectedStatus models.FileStatus
	}{
		{
			name: "successful upload",
			setupFile: func(t *testing.T) string {
				return createTestFile(t, "test file content")
			},
			expiration:     expiration,
			expectError:    false,
			expectedStatus: models.StatusActive,
		},
		{
			name:           "empty file path",
			filePath:       "",
			expiration:     expiration,
			expectError:    true,
			expectedErrMsg: "file path cannot be empty",
		},
		{
			name:           "non-existent file",
			filePath:       "/non/existent/file.txt",
			expiration:     expiration,
			expectError:    true,
			expectedErrMsg: "failed to get file info",
		},
		{
			name: "empty file",
			setupFile: func(t *testing.T) string {
				return createTestFile(t, "")
			},
			expiration:     expiration,
			expectError:    true,
			expectedErrMsg: "file is empty",
		},
		{
			name: "S3 upload failure",
			setupFile: func(t *testing.T) string {
				return createTestFile(t, "test content")
			},
			expiration: expiration,
			setupMock: func(m *mockS3Service) {
				m.shouldError = true
				m.errorMsg = "S3 upload failed"
			},
			expectError:    true,
			expectedErrMsg: "failed to upload file to S3",
			expectedStatus: models.StatusError,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockS3.shouldError = false
			mockS3.errorMsg = ""
			mockS3.uploadedFiles = make(map[string]bool)
			
			// Setup mock if needed
			if tt.setupMock != nil {
				tt.setupMock(mockS3)
			}
			
			// Setup file if needed
			filePath := tt.filePath
			if tt.setupFile != nil {
				filePath = tt.setupFile(t)
			}
			
			// Create progress channel
			progressCh := make(chan aws.UploadProgress, 10)
			
			// Upload file
			fileRecord, err := fm.UploadFile(ctx, filePath, tt.expiration, progressCh)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				
				if tt.expectedStatus != "" {
					// Check if file record was created with error status
					if fileRecord == nil {
						// File record might not be created for validation errors
						return
					}
					assert.Equal(t, tt.expectedStatus, fileRecord.Status)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, fileRecord)
				assert.Equal(t, tt.expectedStatus, fileRecord.Status)
				assert.NotEmpty(t, fileRecord.ID)
				assert.NotEmpty(t, fileRecord.S3Key)
				assert.Contains(t, fileRecord.S3Key, "uploads/")
				assert.False(t, fileRecord.UploadDate.IsZero())
				assert.False(t, fileRecord.ExpirationDate.IsZero())
				
				// Verify file was uploaded to S3
				assert.True(t, mockS3.uploadedFiles[fileRecord.S3Key])
				
				// Check progress updates
				close(progressCh)
				progressUpdates := make([]aws.UploadProgress, 0)
				for progress := range progressCh {
					progressUpdates = append(progressUpdates, progress)
				}
				assert.NotEmpty(t, progressUpdates)
			}
		})
	}
}

func TestFileManager_UploadFile_LargeFile(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	mockS3 := newMockS3Service()
	fm := NewFileManager(db, mockS3)
	
	ctx := context.Background()
	
	// Create a file larger than 100MB (should be rejected)
	tempFile, err := os.CreateTemp("", "large-file-*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	
	// Write more than 100MB of data
	largeContent := make([]byte, 101*1024*1024) // 101MB
	for i := range largeContent {
		largeContent[i] = 'A'
	}
	
	_, err = tempFile.Write(largeContent)
	require.NoError(t, err)
	
	err = tempFile.Close()
	require.NoError(t, err)
	
	// Try to upload the large file
	progressCh := make(chan aws.UploadProgress, 10)
	fileRecord, err := fm.UploadFile(ctx, tempFile.Name(), 24*time.Hour, progressCh)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	assert.Nil(t, fileRecord)
	
	close(progressCh)
}

func TestFileManager_UploadFile_WithoutS3Service(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	// Create FileManager without S3 service
	fm := NewFileManagerWithoutS3(db)
	
	ctx := context.Background()
	testFile := createTestFile(t, "test content")
	
	progressCh := make(chan aws.UploadProgress, 10)
	fileRecord, err := fm.UploadFile(ctx, testFile, 24*time.Hour, progressCh)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "S3 service not configured")
	assert.Nil(t, fileRecord)
	
	close(progressCh)
}

func TestGenerateS3Key(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
	}{
		{
			name:     "file with extension",
			fileName: "test.txt",
		},
		{
			name:     "file without extension",
			fileName: "testfile",
		},
		{
			name:     "file with multiple dots",
			fileName: "test.backup.txt",
		},
		{
			name:     "file with special characters",
			fileName: "test-file_v1.2.pdf",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3Key := generateS3Key(tt.fileName)
			
			// Verify key format: uploads/YYYY/MM/DD/UUID.ext
			assert.Contains(t, s3Key, "uploads/")
			assert.Contains(t, s3Key, time.Now().UTC().Format("2006/01/02"))
			
			// Verify UUID is present (36 characters including hyphens)
			parts := strings.Split(s3Key, "/")
			assert.Len(t, parts, 5) // uploads, year, month, day, uuid.ext
			
			// Verify extension is preserved
			ext := filepath.Ext(tt.fileName)
			if ext != "" {
				assert.Contains(t, s3Key, ext)
			}
			
			// Verify uniqueness by generating multiple keys
			s3Key2 := generateS3Key(tt.fileName)
			assert.NotEqual(t, s3Key, s3Key2)
		})
	}
}

func TestFileManager_UploadFile_Integration(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()
	
	mockS3 := newMockS3Service()
	fm := NewFileManager(db, mockS3)
	
	ctx := context.Background()
	
	// Create test file
	testContent := "This is integration test content for file upload"
	testFile := createTestFile(t, testContent)
	
	// Upload file
	progressCh := make(chan aws.UploadProgress, 10)
	expiration := 24 * time.Hour
	
	fileRecord, err := fm.UploadFile(ctx, testFile, expiration, progressCh)
	assert.NoError(t, err)
	assert.NotNil(t, fileRecord)
	
	// Verify file record
	assert.Equal(t, models.StatusActive, fileRecord.Status)
	assert.Equal(t, filepath.Base(testFile), fileRecord.FileName)
	assert.Equal(t, testFile, fileRecord.FilePath)
	assert.Equal(t, int64(len(testContent)), fileRecord.FileSize)
	assert.NotEmpty(t, fileRecord.S3Key)
	
	// Verify file was uploaded to S3
	assert.True(t, mockS3.uploadedFiles[fileRecord.S3Key])
	
	// Verify file can be retrieved from database
	retrievedFile, err := fm.GetFile(fileRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, fileRecord.ID, retrievedFile.ID)
	assert.Equal(t, models.StatusActive, retrievedFile.Status)
	
	// Verify file appears in list
	allFiles, err := fm.ListFiles()
	assert.NoError(t, err)
	assert.Len(t, allFiles, 1)
	assert.Equal(t, fileRecord.ID, allFiles[0].ID)
	
	// Verify progress updates were sent
	close(progressCh)
	progressUpdates := make([]aws.UploadProgress, 0)
	for progress := range progressCh {
		progressUpdates = append(progressUpdates, progress)
	}
	assert.NotEmpty(t, progressUpdates)
	
	// Verify final progress shows completion
	if len(progressUpdates) > 0 {
		finalProgress := progressUpdates[len(progressUpdates)-1]
		assert.Equal(t, float64(100), finalProgress.Percentage)
	}
}