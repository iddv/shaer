package app

import (
	"path/filepath"
	"testing"
	"time"

	"file-sharing-app/internal/manager"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempDatabase creates a temporary SQLite database for testing
func createTempDatabase(t *testing.T) storage.Database {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err, "Failed to create temporary database")

	return db
}

// MockMainWindow is a minimal mock for testing controller logic
type MockMainWindow struct {
	OnUploadFile           func(filePath string, expiration time.Duration) error
	OnShareFile            func(fileID string, recipients []string, message string) error
	OnDeleteFile           func(fileID string) error
	OnRefreshFiles         func() ([]models.FileMetadata, error)
	OnGeneratePresignedURL func(fileID string, expiration time.Duration) (string, error)
	
	// Track UI updates for testing
	LastStatus      string
	ActionsEnabled  bool
	LastFiles       []models.FileMetadata
}

func (m *MockMainWindow) SetStatus(status string) {
	m.LastStatus = status
}

func (m *MockMainWindow) EnableActions(enabled bool) {
	m.ActionsEnabled = enabled
}

func (m *MockMainWindow) UpdateFiles(files []models.FileMetadata) {
	m.LastFiles = files
}

// Callback setters for interface compliance
func (m *MockMainWindow) SetOnUploadFile(callback func(filePath string, expiration time.Duration) error) {
	m.OnUploadFile = callback
}

func (m *MockMainWindow) SetOnShareFile(callback func(fileID string, recipients []string, message string) error) {
	m.OnShareFile = callback
}

func (m *MockMainWindow) SetOnDeleteFile(callback func(fileID string) error) {
	m.OnDeleteFile = callback
}

func (m *MockMainWindow) SetOnRefreshFiles(callback func() ([]models.FileMetadata, error)) {
	m.OnRefreshFiles = callback
}

func (m *MockMainWindow) SetOnGeneratePresignedURL(callback func(fileID string, expiration time.Duration) (string, error)) {
	m.OnGeneratePresignedURL = callback
}

func TestController_Creation(t *testing.T) {
	// Create test database
	db := createTempDatabase(t)

	// Create managers without S3 service for testing
	fileManager := manager.NewFileManagerWithoutS3(db)
	shareManager := manager.NewShareManager(db, nil)
	expirationManager := manager.NewExpirationManager(db)

	// Create mock UI
	mockWindow := &MockMainWindow{}

	// Create controller
	controller := NewController(fileManager, shareManager, expirationManager, mockWindow)
	assert.NotNil(t, controller)

	// Verify callbacks are set
	assert.NotNil(t, mockWindow.OnUploadFile)
	assert.NotNil(t, mockWindow.OnShareFile)
	assert.NotNil(t, mockWindow.OnDeleteFile)
	assert.NotNil(t, mockWindow.OnRefreshFiles)
	assert.NotNil(t, mockWindow.OnGeneratePresignedURL)

	// Cleanup
	controller.Stop()
}

func TestController_RefreshFiles(t *testing.T) {
	// Create test database
	db := createTempDatabase(t)

	// Create managers
	fileManager := manager.NewFileManagerWithoutS3(db)
	shareManager := manager.NewShareManager(db, nil)
	expirationManager := manager.NewExpirationManager(db)

	// Create mock UI
	mockWindow := &MockMainWindow{}
	
	// Create controller
	controller := NewController(fileManager, shareManager, expirationManager, mockWindow)

	// Add test file to database
	testFile := &models.FileMetadata{
		ID:             "test-file-1",
		FileName:       "test.txt",
		FilePath:       "/tmp/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test.txt",
		Status:         models.StatusActive,
	}
	err := fileManager.SaveFile(testFile)
	require.NoError(t, err)

	// Test refresh files
	files, err := controller.handleRefreshFiles()
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "test-file-1", files[0].ID)
	assert.Equal(t, "test.txt", files[0].FileName)

	// Cleanup
	controller.Stop()
}

func TestController_HandleUploadFile_WithoutS3(t *testing.T) {
	// Create test database
	db := createTempDatabase(t)

	// Create managers without S3 service
	fileManager := manager.NewFileManagerWithoutS3(db)
	shareManager := manager.NewShareManager(db, nil)
	expirationManager := manager.NewExpirationManager(db)

	// Create mock UI
	mockWindow := &MockMainWindow{}
	
	// Create controller
	controller := NewController(fileManager, shareManager, expirationManager, mockWindow)

	// Test upload file (should fail gracefully without S3)
	err := controller.handleUploadFile("/nonexistent/file.txt", 24*time.Hour)
	assert.NoError(t, err) // The error handling is done in the background goroutine

	// Give the goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	controller.Stop()
}

func TestController_HandleDeleteFile(t *testing.T) {
	// Create test database
	db := createTempDatabase(t)

	// Create managers
	fileManager := manager.NewFileManagerWithoutS3(db)
	shareManager := manager.NewShareManager(db, nil)
	expirationManager := manager.NewExpirationManager(db)

	// Create mock UI
	mockWindow := &MockMainWindow{}
	
	// Create controller
	controller := NewController(fileManager, shareManager, expirationManager, mockWindow)

	// Add test file to database
	testFile := &models.FileMetadata{
		ID:             "test-file-1",
		FileName:       "test.txt",
		FilePath:       "/tmp/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test.txt",
		Status:         models.StatusActive,
	}
	err := fileManager.SaveFile(testFile)
	require.NoError(t, err)

	// Test delete file
	err = controller.handleDeleteFile("test-file-1")
	assert.NoError(t, err)

	// Give the goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	// Verify file is deleted
	_, err = fileManager.GetFile("test-file-1")
	assert.Error(t, err) // Should not be found

	// Cleanup
	controller.Stop()
}

func TestController_BackgroundExpirationChecker(t *testing.T) {
	// Create test database
	db := createTempDatabase(t)

	// Create managers
	fileManager := manager.NewFileManagerWithoutS3(db)
	shareManager := manager.NewShareManager(db, nil)
	expirationManager := manager.NewExpirationManager(db)

	// Create mock UI
	mockWindow := &MockMainWindow{}
	
	// Create controller
	controller := NewController(fileManager, shareManager, expirationManager, mockWindow)

	// Add expired test file to database
	expiredFile := &models.FileMetadata{
		ID:             "expired-file-1",
		FileName:       "expired.txt",
		FilePath:       "/tmp/expired.txt",
		FileSize:       1024,
		UploadDate:     time.Now().Add(-2 * time.Hour),
		ExpirationDate: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		S3Key:          "uploads/expired.txt",
		Status:         models.StatusActive,
	}
	err := fileManager.SaveFile(expiredFile)
	require.NoError(t, err)

	// Start controller (this starts the background expiration checker)
	err = controller.Start()
	assert.NoError(t, err)

	// Wait a bit for the background process to run
	time.Sleep(200 * time.Millisecond)

	// Manually trigger expiration check
	controller.checkAndCleanupExpiredFiles()

	// Verify file status was updated to expired
	updatedFile, err := fileManager.GetFile("expired-file-1")
	assert.NoError(t, err)
	assert.Equal(t, models.StatusExpired, updatedFile.Status)

	// Cleanup
	controller.Stop()
}

func TestController_GeneratePresignedURL_WithoutS3(t *testing.T) {
	// Create test database
	db := createTempDatabase(t)

	// Create managers without S3 service
	fileManager := manager.NewFileManagerWithoutS3(db)
	shareManager := manager.NewShareManager(db, nil)
	expirationManager := manager.NewExpirationManager(db)

	// Create mock UI
	mockWindow := &MockMainWindow{}
	
	// Create controller
	controller := NewController(fileManager, shareManager, expirationManager, mockWindow)

	// Test generate presigned URL (should fail gracefully without S3)
	url, err := controller.GeneratePresignedURL("nonexistent-file", 24*time.Hour)
	assert.Error(t, err)
	assert.Empty(t, url)

	// Cleanup
	controller.Stop()
}

func TestController_UIIntegration(t *testing.T) {
	// Create test database
	db := createTempDatabase(t)

	// Create managers
	fileManager := manager.NewFileManagerWithoutS3(db)
	shareManager := manager.NewShareManager(db, nil)
	expirationManager := manager.NewExpirationManager(db)

	// Create mock UI
	mockWindow := &MockMainWindow{}
	
	// Create controller
	controller := NewController(fileManager, shareManager, expirationManager, mockWindow)

	// Test that controller properly updates UI state
	err := controller.Start()
	assert.NoError(t, err)

	// Verify UI was updated
	assert.True(t, mockWindow.ActionsEnabled)
	assert.Equal(t, "Ready", mockWindow.LastStatus)

	// Cleanup
	controller.Stop()
}