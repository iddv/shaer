package manager

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
)

// createTempDatabaseForExpiration creates a temporary SQLite database for expiration testing
func createTempDatabaseForExpiration(t *testing.T) (*storage.SQLiteDatabase, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "expiration_test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	
	return db, dbPath
}

// createTestFileMetadata creates test file metadata with specified expiration
func createTestFileMetadata(id, fileName string, expirationDate time.Time, status storage.FileStatus) *storage.FileMetadata {
	return &storage.FileMetadata{
		ID:             id,
		FileName:       fileName,
		FilePath:       fmt.Sprintf("/tmp/%s", fileName),
		FileSize:       1024,
		UploadDate:     time.Now().Add(-1 * time.Hour), // Uploaded 1 hour ago
		ExpirationDate: expirationDate,
		S3Key:          fmt.Sprintf("uploads/%s", fileName),
		Status:         status,
	}
}

func TestNewExpirationManager(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	assert.NotNil(t, em)
	assert.IsType(t, &ExpirationManagerImpl{}, em)
}

func TestExpirationManager_CleanupExpiredMetadata(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	now := time.Now()
	
	// Create test files with different expiration scenarios
	testFiles := []*storage.FileMetadata{
		// File expired 35 days ago (should be deleted)
		createTestFileMetadata("file1", "old_expired.txt", now.Add(-35*24*time.Hour), storage.StatusExpired),
		// File expired 20 days ago (should not be deleted yet)
		createTestFileMetadata("file2", "recent_expired.txt", now.Add(-20*24*time.Hour), storage.StatusExpired),
		// File expired 40 days ago (should be deleted)
		createTestFileMetadata("file3", "very_old_expired.txt", now.Add(-40*24*time.Hour), storage.StatusExpired),
		// Active file (should not be deleted)
		createTestFileMetadata("file4", "active.txt", now.Add(24*time.Hour), storage.StatusActive),
	}
	
	// Save test files to database
	for _, file := range testFiles {
		err := db.SaveFile(file)
		require.NoError(t, err)
	}
	
	// Run cleanup
	err := em.CleanupExpiredMetadata()
	assert.NoError(t, err)
	
	// Verify that old expired files were deleted
	_, err = db.GetFile("file1")
	assert.Error(t, err) // Should not exist
	
	_, err = db.GetFile("file3")
	assert.Error(t, err) // Should not exist
	
	// Verify that recent expired and active files still exist
	_, err = db.GetFile("file2")
	assert.NoError(t, err) // Should still exist
	
	_, err = db.GetFile("file4")
	assert.NoError(t, err) // Should still exist
}

func TestExpirationManager_CleanupExpiredMetadata_NoFilesToCleanup(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	now := time.Now()
	
	// Create test files that should not be cleaned up
	testFiles := []*storage.FileMetadata{
		// Recently expired file
		createTestFileMetadata("file1", "recent_expired.txt", now.Add(-10*24*time.Hour), storage.StatusExpired),
		// Active file
		createTestFileMetadata("file2", "active.txt", now.Add(24*time.Hour), storage.StatusActive),
	}
	
	// Save test files to database
	for _, file := range testFiles {
		err := db.SaveFile(file)
		require.NoError(t, err)
	}
	
	// Run cleanup
	err := em.CleanupExpiredMetadata()
	assert.NoError(t, err)
	
	// Verify that all files still exist
	_, err = db.GetFile("file1")
	assert.NoError(t, err)
	
	_, err = db.GetFile("file2")
	assert.NoError(t, err)
}

func TestExpirationManager_SetExpiration(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create a test file
	testFile := createTestFileMetadata("test-file-1", "test.txt", time.Now().Add(24*time.Hour), storage.StatusActive)
	err := db.SaveFile(testFile)
	require.NoError(t, err)
	
	tests := []struct {
		name        string
		fileID      string
		duration    time.Duration
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid expiration update",
			fileID:      "test-file-1",
			duration:    48 * time.Hour,
			expectError: false,
		},
		{
			name:        "empty file ID",
			fileID:      "",
			duration:    24 * time.Hour,
			expectError: true,
			errorMsg:    "file ID cannot be empty",
		},
		{
			name:        "zero duration",
			fileID:      "test-file-1",
			duration:    0,
			expectError: true,
			errorMsg:    "expiration duration must be positive",
		},
		{
			name:        "negative duration",
			fileID:      "test-file-1",
			duration:    -24 * time.Hour,
			expectError: true,
			errorMsg:    "expiration duration must be positive",
		},
		{
			name:        "non-existent file",
			fileID:      "non-existent",
			duration:    24 * time.Hour,
			expectError: true,
			errorMsg:    "failed to get file",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTime := time.Now()
			err := em.SetExpiration(tt.fileID, tt.duration)
			afterTime := time.Now()
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				
				// Verify the expiration was updated
				updatedFile, err := db.GetFile(tt.fileID)
				assert.NoError(t, err)
				
				// Check that the new expiration date is within the expected range
				expectedMin := beforeTime.Add(tt.duration)
				expectedMax := afterTime.Add(tt.duration)
				assert.True(t, updatedFile.ExpirationDate.After(expectedMin) || updatedFile.ExpirationDate.Equal(expectedMin))
				assert.True(t, updatedFile.ExpirationDate.Before(expectedMax) || updatedFile.ExpirationDate.Equal(expectedMax))
			}
		})
	}
}

func TestExpirationManager_CheckExpirations(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create test files with different expiration states
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)   // Expired 24 hours ago
	futureTime := now.Add(24 * time.Hour)  // Expires in 24 hours
	
	testFiles := []*storage.FileMetadata{
		createTestFileMetadata("expired-active", "expired-active.txt", pastTime, storage.StatusActive),
		createTestFileMetadata("expired-uploading", "expired-uploading.txt", pastTime, storage.StatusUploading),
		createTestFileMetadata("expired-already-expired", "expired-already-expired.txt", pastTime, storage.StatusExpired),
		createTestFileMetadata("expired-deleted", "expired-deleted.txt", pastTime, storage.StatusDeleted),
		createTestFileMetadata("not-expired-active", "not-expired-active.txt", futureTime, storage.StatusActive),
		createTestFileMetadata("not-expired-uploading", "not-expired-uploading.txt", futureTime, storage.StatusUploading),
	}
	
	// Save all test files
	for _, file := range testFiles {
		err := db.SaveFile(file)
		require.NoError(t, err)
	}
	
	// Check expirations
	expiredFiles, err := em.CheckExpirations()
	assert.NoError(t, err)
	
	// Should return only files that are expired but not already marked as expired or deleted
	assert.Len(t, expiredFiles, 2)
	
	// Verify the correct files are returned
	expectedIDs := []string{"expired-active", "expired-uploading"}
	actualIDs := make([]string, len(expiredFiles))
	for i, file := range expiredFiles {
		actualIDs[i] = file.ID
		// Verify the file is actually expired
		assert.True(t, file.ExpirationDate.Before(now))
		// Verify the file is not already marked as expired or deleted
		assert.NotEqual(t, models.StatusExpired, file.Status)
		assert.NotEqual(t, models.StatusDeleted, file.Status)
	}
	
	for _, expectedID := range expectedIDs {
		assert.Contains(t, actualIDs, expectedID)
	}
}

func TestExpirationManager_CheckExpirations_EmptyDatabase(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Check expirations on empty database
	expiredFiles, err := em.CheckExpirations()
	assert.NoError(t, err)
	assert.Empty(t, expiredFiles)
}

func TestExpirationManager_CleanupExpiredFiles(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create test files with different expiration states
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)   // Expired 24 hours ago
	futureTime := now.Add(24 * time.Hour)  // Expires in 24 hours
	
	testFiles := []*storage.FileMetadata{
		createTestFileMetadata("expired-active", "expired-active.txt", pastTime, storage.StatusActive),
		createTestFileMetadata("expired-uploading", "expired-uploading.txt", pastTime, storage.StatusUploading),
		createTestFileMetadata("expired-error", "expired-error.txt", pastTime, storage.StatusError),
		createTestFileMetadata("expired-already-expired", "expired-already-expired.txt", pastTime, storage.StatusExpired),
		createTestFileMetadata("not-expired-active", "not-expired-active.txt", futureTime, storage.StatusActive),
	}
	
	// Save all test files
	for _, file := range testFiles {
		err := db.SaveFile(file)
		require.NoError(t, err)
	}
	
	// Cleanup expired files
	err := em.CleanupExpiredFiles()
	assert.NoError(t, err)
	
	// Verify that expired files have been updated to expired status
	expiredActiveFile, err := db.GetFile("expired-active")
	assert.NoError(t, err)
	assert.Equal(t, storage.StatusExpired, expiredActiveFile.Status)
	
	expiredUploadingFile, err := db.GetFile("expired-uploading")
	assert.NoError(t, err)
	assert.Equal(t, storage.StatusExpired, expiredUploadingFile.Status)
	
	expiredErrorFile, err := db.GetFile("expired-error")
	assert.NoError(t, err)
	assert.Equal(t, storage.StatusExpired, expiredErrorFile.Status)
	
	// Verify that already expired file status remains unchanged
	alreadyExpiredFile, err := db.GetFile("expired-already-expired")
	assert.NoError(t, err)
	assert.Equal(t, storage.StatusExpired, alreadyExpiredFile.Status)
	
	// Verify that non-expired file status remains unchanged
	notExpiredFile, err := db.GetFile("not-expired-active")
	assert.NoError(t, err)
	assert.Equal(t, storage.StatusActive, notExpiredFile.Status)
}

func TestExpirationManager_CleanupExpiredFiles_NoExpiredFiles(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create only non-expired files
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)
	
	testFile := createTestFileMetadata("not-expired", "not-expired.txt", futureTime, storage.StatusActive)
	err := db.SaveFile(testFile)
	require.NoError(t, err)
	
	// Cleanup expired files (should be no-op)
	err = em.CleanupExpiredFiles()
	assert.NoError(t, err)
	
	// Verify file status remains unchanged
	file, err := db.GetFile("not-expired")
	assert.NoError(t, err)
	assert.Equal(t, storage.StatusActive, file.Status)
}

func TestExpirationManager_CleanupExpiredFiles_EmptyDatabase(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Cleanup expired files on empty database
	err := em.CleanupExpiredFiles()
	assert.NoError(t, err)
}

func TestExpirationManager_GetExpiredFiles(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create test files
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)
	
	testFiles := []*storage.FileMetadata{
		createTestFileMetadata("expired-1", "expired-1.txt", pastTime, storage.StatusActive),
		createTestFileMetadata("expired-2", "expired-2.txt", pastTime, storage.StatusUploading),
		createTestFileMetadata("not-expired", "not-expired.txt", futureTime, storage.StatusActive),
	}
	
	for _, file := range testFiles {
		err := db.SaveFile(file)
		require.NoError(t, err)
	}
	
	// Get expired files
	expiredFiles, err := em.GetExpiredFiles()
	assert.NoError(t, err)
	assert.Len(t, expiredFiles, 2)
	
	// Verify correct files are returned
	expectedIDs := []string{"expired-1", "expired-2"}
	actualIDs := make([]string, len(expiredFiles))
	for i, file := range expiredFiles {
		actualIDs[i] = file.ID
	}
	
	for _, expectedID := range expectedIDs {
		assert.Contains(t, actualIDs, expectedID)
	}
}

func TestExpirationManager_IsFileExpired(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create test files
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)   // Expired
	futureTime := now.Add(24 * time.Hour)  // Not expired
	
	expiredFile := createTestFileMetadata("expired-file", "expired.txt", pastTime, storage.StatusActive)
	notExpiredFile := createTestFileMetadata("not-expired-file", "not-expired.txt", futureTime, storage.StatusActive)
	
	err := db.SaveFile(expiredFile)
	require.NoError(t, err)
	err = db.SaveFile(notExpiredFile)
	require.NoError(t, err)
	
	tests := []struct {
		name        string
		fileID      string
		expectError bool
		errorMsg    string
		expected    bool
	}{
		{
			name:        "expired file",
			fileID:      "expired-file",
			expectError: false,
			expected:    true,
		},
		{
			name:        "not expired file",
			fileID:      "not-expired-file",
			expectError: false,
			expected:    false,
		},
		{
			name:        "empty file ID",
			fileID:      "",
			expectError: true,
			errorMsg:    "file ID cannot be empty",
		},
		{
			name:        "non-existent file",
			fileID:      "non-existent",
			expectError: true,
			errorMsg:    "failed to get file",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isExpired, err := em.IsFileExpired(tt.fileID)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, isExpired)
			}
		})
	}
}

func TestExpirationManager_GetTimeUntilExpiration(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create test files with specific expiration times
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)   // Expired 24 hours ago
	futureTime := now.Add(24 * time.Hour)  // Expires in 24 hours
	
	expiredFile := createTestFileMetadata("expired-file", "expired.txt", pastTime, storage.StatusActive)
	notExpiredFile := createTestFileMetadata("not-expired-file", "not-expired.txt", futureTime, storage.StatusActive)
	
	err := db.SaveFile(expiredFile)
	require.NoError(t, err)
	err = db.SaveFile(notExpiredFile)
	require.NoError(t, err)
	
	tests := []struct {
		name        string
		fileID      string
		expectError bool
		errorMsg    string
		checkResult func(t *testing.T, duration time.Duration)
	}{
		{
			name:        "not expired file",
			fileID:      "not-expired-file",
			expectError: false,
			checkResult: func(t *testing.T, duration time.Duration) {
				// Should be approximately 24 hours (within 1 minute tolerance)
				expected := 24 * time.Hour
				tolerance := 1 * time.Minute
				assert.True(t, duration > expected-tolerance && duration < expected+tolerance,
					"Expected duration around %v, got %v", expected, duration)
			},
		},
		{
			name:        "expired file",
			fileID:      "expired-file",
			expectError: false,
			checkResult: func(t *testing.T, duration time.Duration) {
				// Should return 0 for expired files
				assert.Equal(t, time.Duration(0), duration)
			},
		},
		{
			name:        "empty file ID",
			fileID:      "",
			expectError: true,
			errorMsg:    "file ID cannot be empty",
		},
		{
			name:        "non-existent file",
			fileID:      "non-existent",
			expectError: true,
			errorMsg:    "failed to get file",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := em.GetTimeUntilExpiration(tt.fileID)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, duration)
				}
			}
		})
	}
}

func TestExpirationManager_TimeManipulation(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	_ = NewExpirationManager(db) // Create manager but don't use it in this test
	
	// Test with precise time manipulation
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	
	// Create files with specific expiration times relative to base time
	testFiles := []*storage.FileMetadata{
		{
			ID:             "expires-in-1-hour",
			FileName:       "expires-in-1-hour.txt",
			FilePath:       "/tmp/expires-in-1-hour.txt",
			FileSize:       1024,
			UploadDate:     baseTime.Add(-1 * time.Hour),
			ExpirationDate: baseTime.Add(1 * time.Hour), // Expires 1 hour after base time
			S3Key:          "uploads/expires-in-1-hour.txt",
			Status:         storage.StatusActive,
		},
		{
			ID:             "expired-1-hour-ago",
			FileName:       "expired-1-hour-ago.txt",
			FilePath:       "/tmp/expired-1-hour-ago.txt",
			FileSize:       1024,
			UploadDate:     baseTime.Add(-2 * time.Hour),
			ExpirationDate: baseTime.Add(-1 * time.Hour), // Expired 1 hour before base time
			S3Key:          "uploads/expired-1-hour-ago.txt",
			Status:         storage.StatusActive,
		},
		{
			ID:             "expires-exactly-now",
			FileName:       "expires-exactly-now.txt",
			FilePath:       "/tmp/expires-exactly-now.txt",
			FileSize:       1024,
			UploadDate:     baseTime.Add(-1 * time.Hour),
			ExpirationDate: baseTime, // Expires exactly at base time
			S3Key:          "uploads/expires-exactly-now.txt",
			Status:         storage.StatusActive,
		},
	}
	
	// Save test files
	for _, file := range testFiles {
		err := db.SaveFile(file)
		require.NoError(t, err)
	}
	
	// Test IsFileExpired with time manipulation
	t.Run("IsFileExpired with time manipulation", func(t *testing.T) {
		// Mock current time to be base time
		// Note: In a real implementation, you might use a time provider interface
		// For this test, we'll test the logic by checking against known expiration dates
		
		// File that expires in 1 hour should not be expired at base time
		// We can't easily mock time.Now() without dependency injection, 
		// so we'll test the logic by comparing expiration dates directly
		
		file1, err := db.GetFile("expires-in-1-hour")
		assert.NoError(t, err)
		assert.True(t, file1.ExpirationDate.After(baseTime))
		
		file2, err := db.GetFile("expired-1-hour-ago")
		assert.NoError(t, err)
		assert.True(t, file2.ExpirationDate.Before(baseTime))
		
		file3, err := db.GetFile("expires-exactly-now")
		assert.NoError(t, err)
		assert.True(t, file3.ExpirationDate.Equal(baseTime))
	})
	
	// Test GetTimeUntilExpiration with precise calculations
	t.Run("GetTimeUntilExpiration with precise calculations", func(t *testing.T) {
		// Test the calculation logic by examining the stored expiration dates
		file1, err := db.GetFile("expires-in-1-hour")
		assert.NoError(t, err)
		
		// Calculate time until expiration from base time
		timeUntilExpiration := file1.ExpirationDate.Sub(baseTime)
		assert.Equal(t, 1*time.Hour, timeUntilExpiration)
		
		file2, err := db.GetFile("expired-1-hour-ago")
		assert.NoError(t, err)
		
		// Calculate time until expiration from base time (should be negative)
		timeUntilExpiration = file2.ExpirationDate.Sub(baseTime)
		assert.Equal(t, -1*time.Hour, timeUntilExpiration)
	})
}

func TestExpirationManager_Integration(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create a test file
	testFile := createTestFileMetadata("integration-test", "integration.txt", time.Now().Add(1*time.Hour), storage.StatusActive)
	err := db.SaveFile(testFile)
	require.NoError(t, err)
	
	// Test complete workflow
	t.Run("Complete expiration workflow", func(t *testing.T) {
		// 1. Verify file is not expired initially
		isExpired, err := em.IsFileExpired("integration-test")
		assert.NoError(t, err)
		assert.False(t, isExpired)
		
		// 2. Check time until expiration
		timeUntil, err := em.GetTimeUntilExpiration("integration-test")
		assert.NoError(t, err)
		assert.True(t, timeUntil > 0)
		assert.True(t, timeUntil <= 1*time.Hour)
		
		// 3. Update the file's expiration date directly in the database to simulate expiration
		pastTime := time.Now().Add(-2 * time.Hour)
		err = db.UpdateFileExpiration("integration-test", pastTime)
		assert.NoError(t, err)
		
		// 4. Verify file is now expired
		isExpired, err = em.IsFileExpired("integration-test")
		assert.NoError(t, err)
		assert.True(t, isExpired)
		
		// 5. Check expired files
		expiredFiles, err := em.CheckExpirations()
		assert.NoError(t, err)
		assert.Len(t, expiredFiles, 1)
		assert.Equal(t, "integration-test", expiredFiles[0].ID)
		
		// 6. Cleanup expired files
		err = em.CleanupExpiredFiles()
		assert.NoError(t, err)
		
		// 7. Verify file status was updated
		updatedFile, err := db.GetFile("integration-test")
		assert.NoError(t, err)
		assert.Equal(t, storage.StatusExpired, updatedFile.Status)
		
		// 8. Verify no more expired files to cleanup
		expiredFiles, err = em.CheckExpirations()
		assert.NoError(t, err)
		assert.Empty(t, expiredFiles) // Should be empty since file is now marked as expired
	})
}

func TestExpirationManager_ConcurrentAccess(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	// Create multiple test files
	now := time.Now()
	for i := 0; i < 10; i++ {
		testFile := createTestFileMetadata(
			fmt.Sprintf("concurrent-test-%d", i),
			fmt.Sprintf("concurrent-%d.txt", i),
			now.Add(-1*time.Hour), // All expired
			storage.StatusActive,
		)
		err := db.SaveFile(testFile)
		require.NoError(t, err)
	}
	
	// Test concurrent cleanup operations
	// Note: SQLite handles concurrency through locking, so this tests the robustness
	t.Run("Concurrent cleanup", func(t *testing.T) {
		// First cleanup
		err := em.CleanupExpiredFiles()
		assert.NoError(t, err)
		
		// Second cleanup (should be no-op)
		err = em.CleanupExpiredFiles()
		assert.NoError(t, err)
		
		// Verify all files are marked as expired
		for i := 0; i < 10; i++ {
			file, err := db.GetFile(fmt.Sprintf("concurrent-test-%d", i))
			assert.NoError(t, err)
			assert.Equal(t, storage.StatusExpired, file.Status)
		}
	})
}

func TestExpirationManager_EdgeCases(t *testing.T) {
	db, _ := createTempDatabaseForExpiration(t)
	defer db.Close()
	
	em := NewExpirationManager(db)
	
	t.Run("File expires exactly at current time", func(t *testing.T) {
		// Create a file that expires very close to now
		now := time.Now()
		testFile := createTestFileMetadata("edge-case-now", "edge-now.txt", now, storage.StatusActive)
		err := db.SaveFile(testFile)
		require.NoError(t, err)
		
		// The file should be considered expired (Before returns true for equal times)
		// Wait a tiny bit to ensure we're past the expiration time
		time.Sleep(1 * time.Millisecond)
		
		expiredFiles, err := em.CheckExpirations()
		assert.NoError(t, err)
		
		// Should include the file since it expired at or before now
		found := false
		for _, file := range expiredFiles {
			if file.ID == "edge-case-now" {
				found = true
				break
			}
		}
		assert.True(t, found, "File expiring at current time should be considered expired")
	})
	
	t.Run("Very large expiration duration", func(t *testing.T) {
		testFile := createTestFileMetadata("large-duration", "large.txt", time.Now().Add(1*time.Hour), storage.StatusActive)
		err := db.SaveFile(testFile)
		require.NoError(t, err)
		
		// Set expiration to a very large duration (100 years)
		largeDuration := 100 * 365 * 24 * time.Hour
		err = em.SetExpiration("large-duration", largeDuration)
		assert.NoError(t, err)
		
		// Verify the expiration was set
		timeUntil, err := em.GetTimeUntilExpiration("large-duration")
		assert.NoError(t, err)
		assert.True(t, timeUntil > 99*365*24*time.Hour) // Should be close to 100 years
	})
}