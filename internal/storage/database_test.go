package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempDatabase creates a temporary SQLite database for testing
func createTempDatabase(t *testing.T) (*SQLiteDatabase, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewSQLiteDatabase(dbPath)
	require.NoError(t, err, "Failed to create temporary database")

	return db, dbPath
}

func TestNewSQLiteDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Verify database file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "Database file should exist")

	// Verify file permissions (600 - read/write for owner only)
	info, err := os.Stat(dbPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "Database file should have 600 permissions")
}

func TestSQLiteDatabase_SaveFile(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	file := &FileMetadata{
		ID:             "test-id-1",
		FileName:       "test.txt",
		FilePath:       "/tmp/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test-id-1/test.txt",
		Status:         StatusActive,
	}

	err := db.SaveFile(file)
	assert.NoError(t, err)

	// Verify the file was saved with timestamps
	assert.False(t, file.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, file.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

func TestSQLiteDatabase_GetFile(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save a file first
	originalFile := &FileMetadata{
		ID:             "test-id-2",
		FileName:       "test2.txt",
		FilePath:       "/tmp/test2.txt",
		FileSize:       2048,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(48 * time.Hour),
		S3Key:          "uploads/test-id-2/test2.txt",
		Status:         StatusUploading,
	}

	err := db.SaveFile(originalFile)
	require.NoError(t, err)

	// Retrieve the file
	retrievedFile, err := db.GetFile("test-id-2")
	require.NoError(t, err)

	assert.Equal(t, originalFile.ID, retrievedFile.ID)
	assert.Equal(t, originalFile.FileName, retrievedFile.FileName)
	assert.Equal(t, originalFile.FilePath, retrievedFile.FilePath)
	assert.Equal(t, originalFile.FileSize, retrievedFile.FileSize)
	assert.Equal(t, originalFile.S3Key, retrievedFile.S3Key)
	assert.Equal(t, originalFile.Status, retrievedFile.Status)
	assert.WithinDuration(t, originalFile.UploadDate, retrievedFile.UploadDate, time.Second)
	assert.WithinDuration(t, originalFile.ExpirationDate, retrievedFile.ExpirationDate, time.Second)
}

func TestSQLiteDatabase_GetFile_NotFound(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	_, err := db.GetFile("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestSQLiteDatabase_ListFiles(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save multiple files
	files := []*FileMetadata{
		{
			ID:             "test-id-3",
			FileName:       "test3.txt",
			FilePath:       "/tmp/test3.txt",
			FileSize:       1024,
			UploadDate:     time.Now().Add(-2 * time.Hour),
			ExpirationDate: time.Now().Add(22 * time.Hour),
			S3Key:          "uploads/test-id-3/test3.txt",
			Status:         StatusActive,
		},
		{
			ID:             "test-id-4",
			FileName:       "test4.txt",
			FilePath:       "/tmp/test4.txt",
			FileSize:       2048,
			UploadDate:     time.Now().Add(-1 * time.Hour),
			ExpirationDate: time.Now().Add(23 * time.Hour),
			S3Key:          "uploads/test-id-4/test4.txt",
			Status:         StatusExpired,
		},
	}

	for _, file := range files {
		err := db.SaveFile(file)
		require.NoError(t, err)
	}

	// List all files
	retrievedFiles, err := db.ListFiles()
	require.NoError(t, err)
	assert.Len(t, retrievedFiles, 2)

	// Files should be ordered by upload_date DESC (most recent first)
	assert.Equal(t, "test-id-4", retrievedFiles[0].ID)
	assert.Equal(t, "test-id-3", retrievedFiles[1].ID)
}

func TestSQLiteDatabase_UpdateFileStatus(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save a file first
	file := &FileMetadata{
		ID:             "test-id-5",
		FileName:       "test5.txt",
		FilePath:       "/tmp/test5.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test-id-5/test5.txt",
		Status:         StatusUploading,
	}

	err := db.SaveFile(file)
	require.NoError(t, err)

	// Update status
	err = db.UpdateFileStatus("test-id-5", StatusActive)
	assert.NoError(t, err)

	// Verify status was updated
	retrievedFile, err := db.GetFile("test-id-5")
	require.NoError(t, err)
	assert.Equal(t, StatusActive, retrievedFile.Status)
}

func TestSQLiteDatabase_UpdateFileStatus_NotFound(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	err := db.UpdateFileStatus("non-existent-id", StatusActive)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestSQLiteDatabase_UpdateFileExpiration(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save a file first
	originalExpiration := time.Now().Add(24 * time.Hour)
	file := &FileMetadata{
		ID:             "test-id-expiration",
		FileName:       "test-expiration.txt",
		FilePath:       "/tmp/test-expiration.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: originalExpiration,
		S3Key:          "uploads/test-expiration.txt",
		Status:         StatusActive,
	}

	err := db.SaveFile(file)
	require.NoError(t, err)

	// Update expiration date
	newExpiration := time.Now().Add(48 * time.Hour)
	err = db.UpdateFileExpiration("test-id-expiration", newExpiration)
	assert.NoError(t, err)

	// Verify the expiration was updated
	retrievedFile, err := db.GetFile("test-id-expiration")
	require.NoError(t, err)
	
	// Compare with some tolerance for time precision
	timeDiff := retrievedFile.ExpirationDate.Sub(newExpiration)
	assert.True(t, timeDiff < time.Second && timeDiff > -time.Second, 
		"Expected expiration date to be close to %v, got %v", newExpiration, retrievedFile.ExpirationDate)
}

func TestSQLiteDatabase_UpdateFileExpiration_NotFound(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	newExpiration := time.Now().Add(48 * time.Hour)
	err := db.UpdateFileExpiration("non-existent-id", newExpiration)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestSQLiteDatabase_DeleteFile(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save a file first
	file := &FileMetadata{
		ID:             "test-id-6",
		FileName:       "test6.txt",
		FilePath:       "/tmp/test6.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test-id-6/test6.txt",
		Status:         StatusActive,
	}

	err := db.SaveFile(file)
	require.NoError(t, err)

	// Delete the file
	err = db.DeleteFile("test-id-6")
	assert.NoError(t, err)

	// Verify file is deleted
	_, err = db.GetFile("test-id-6")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestSQLiteDatabase_DeleteFile_NotFound(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	err := db.DeleteFile("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestSQLiteDatabase_SaveShare(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save a file first (required for foreign key constraint)
	file := &FileMetadata{
		ID:             "test-file-id",
		FileName:       "test.txt",
		FilePath:       "/tmp/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test-file-id/test.txt",
		Status:         StatusActive,
	}

	err := db.SaveFile(file)
	require.NoError(t, err)

	// Save a share record
	share := &ShareRecord{
		ID:            "share-id-1",
		FileID:        "test-file-id",
		Recipients:    []string{"user1@example.com", "user2@example.com"},
		Message:       "Please find the attached file",
		SharedDate:    time.Now(),
		PresignedURL:  "https://s3.amazonaws.com/bucket/key?signature=xyz",
		URLExpiration: time.Now().Add(7 * 24 * time.Hour),
	}

	err = db.SaveShare(share)
	assert.NoError(t, err)

	// Verify the share was saved with timestamp
	assert.False(t, share.CreatedAt.IsZero(), "CreatedAt should be set")
}

func TestSQLiteDatabase_GetShareHistory(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save a file first
	file := &FileMetadata{
		ID:             "test-file-id-2",
		FileName:       "test2.txt",
		FilePath:       "/tmp/test2.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test-file-id-2/test2.txt",
		Status:         StatusActive,
	}

	err := db.SaveFile(file)
	require.NoError(t, err)

	// Save multiple share records
	shares := []*ShareRecord{
		{
			ID:            "share-id-2",
			FileID:        "test-file-id-2",
			Recipients:    []string{"user1@example.com"},
			Message:       "First share",
			SharedDate:    time.Now().Add(-2 * time.Hour),
			PresignedURL:  "https://s3.amazonaws.com/bucket/key1?signature=abc",
			URLExpiration: time.Now().Add(5 * 24 * time.Hour),
		},
		{
			ID:            "share-id-3",
			FileID:        "test-file-id-2",
			Recipients:    []string{"user2@example.com", "user3@example.com"},
			Message:       "Second share",
			SharedDate:    time.Now().Add(-1 * time.Hour),
			PresignedURL:  "https://s3.amazonaws.com/bucket/key2?signature=def",
			URLExpiration: time.Now().Add(6 * 24 * time.Hour),
		},
	}

	for _, share := range shares {
		err := db.SaveShare(share)
		require.NoError(t, err)
	}

	// Get share history
	retrievedShares, err := db.GetShareHistory("test-file-id-2")
	require.NoError(t, err)
	assert.Len(t, retrievedShares, 2)

	// Shares should be ordered by shared_date DESC (most recent first)
	assert.Equal(t, "share-id-3", retrievedShares[0].ID)
	assert.Equal(t, "share-id-2", retrievedShares[1].ID)

	// Verify recipients are properly unmarshaled
	assert.Equal(t, []string{"user2@example.com", "user3@example.com"}, retrievedShares[0].Recipients)
	assert.Equal(t, []string{"user1@example.com"}, retrievedShares[1].Recipients)
}

func TestSQLiteDatabase_GetShareHistory_NoShares(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	shares, err := db.GetShareHistory("non-existent-file-id")
	assert.NoError(t, err)
	assert.Empty(t, shares)
}

func TestSQLiteDatabase_SaveConfig(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	err := db.SaveConfig("aws_region", "us-west-2")
	assert.NoError(t, err)

	err = db.SaveConfig("s3_bucket", "my-test-bucket")
	assert.NoError(t, err)
}

func TestSQLiteDatabase_GetConfig(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save config first
	err := db.SaveConfig("default_expiration", "24h")
	require.NoError(t, err)

	// Retrieve config
	value, err := db.GetConfig("default_expiration")
	assert.NoError(t, err)
	assert.Equal(t, "24h", value)
}

func TestSQLiteDatabase_GetConfig_NotFound(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	_, err := db.GetConfig("non-existent-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config key not found")
}

func TestSQLiteDatabase_SaveConfig_Replace(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save initial config
	err := db.SaveConfig("max_file_size", "100MB")
	require.NoError(t, err)

	// Replace with new value
	err = db.SaveConfig("max_file_size", "200MB")
	require.NoError(t, err)

	// Verify new value
	value, err := db.GetConfig("max_file_size")
	assert.NoError(t, err)
	assert.Equal(t, "200MB", value)
}

func TestSQLiteDatabase_ForeignKeyConstraint(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Try to save a share without a corresponding file (should fail due to foreign key constraint)
	share := &ShareRecord{
		ID:            "share-id-orphan",
		FileID:        "non-existent-file-id",
		Recipients:    []string{"user@example.com"},
		Message:       "This should fail",
		SharedDate:    time.Now(),
		PresignedURL:  "https://s3.amazonaws.com/bucket/key?signature=xyz",
		URLExpiration: time.Now().Add(24 * time.Hour),
	}

	err := db.SaveShare(share)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FOREIGN KEY constraint failed")
}

func TestSQLiteDatabase_CascadeDelete(t *testing.T) {
	db, _ := createTempDatabase(t)
	defer db.Close()

	// Save a file
	file := &FileMetadata{
		ID:             "test-cascade-file",
		FileName:       "cascade-test.txt",
		FilePath:       "/tmp/cascade-test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "uploads/test-cascade-file/cascade-test.txt",
		Status:         StatusActive,
	}

	err := db.SaveFile(file)
	require.NoError(t, err)

	// Save a share for the file
	share := &ShareRecord{
		ID:            "share-cascade-test",
		FileID:        "test-cascade-file",
		Recipients:    []string{"user@example.com"},
		Message:       "Test cascade delete",
		SharedDate:    time.Now(),
		PresignedURL:  "https://s3.amazonaws.com/bucket/key?signature=xyz",
		URLExpiration: time.Now().Add(24 * time.Hour),
	}

	err = db.SaveShare(share)
	require.NoError(t, err)

	// Verify share exists
	shares, err := db.GetShareHistory("test-cascade-file")
	require.NoError(t, err)
	assert.Len(t, shares, 1)

	// Delete the file
	err = db.DeleteFile("test-cascade-file")
	require.NoError(t, err)

	// Verify shares are also deleted (cascade delete)
	shares, err = db.GetShareHistory("test-cascade-file")
	require.NoError(t, err)
	assert.Empty(t, shares)
}

func TestSQLiteDatabase_Close(t *testing.T) {
	db, _ := createTempDatabase(t)

	err := db.Close()
	assert.NoError(t, err)

	// Calling Close again should not error
	err = db.Close()
	assert.NoError(t, err)
}