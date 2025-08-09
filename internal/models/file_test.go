package models

import (
	"testing"
	"time"
)

func TestFileMetadata(t *testing.T) {
	// Test FileMetadata creation
	file := &FileMetadata{
		ID:             "test-id",
		FileName:       "test.txt",
		FilePath:       "/path/to/test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		S3Key:          "test-key",
		Status:         StatusActive,
	}

	if file.ID != "test-id" {
		t.Errorf("Expected ID to be 'test-id', got %s", file.ID)
	}

	if file.Status != StatusActive {
		t.Errorf("Expected Status to be StatusActive, got %s", file.Status)
	}
}

func TestFileStatus(t *testing.T) {
	// Test FileStatus constants
	statuses := []FileStatus{
		StatusUploading,
		StatusActive,
		StatusExpired,
		StatusDeleted,
		StatusError,
	}

	expectedValues := []string{
		"uploading",
		"active",
		"expired",
		"deleted",
		"error",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("Expected status %d to be '%s', got '%s'", i, expectedValues[i], string(status))
		}
	}
}

func TestShareRecord(t *testing.T) {
	// Test ShareRecord creation
	share := &ShareRecord{
		ID:            "share-id",
		FileID:        "file-id",
		Recipients:    []string{"user@example.com"},
		Message:       "Test message",
		SharedDate:    time.Now(),
		PresignedURL:  "https://example.com/presigned-url",
		URLExpiration: time.Now().Add(time.Hour),
	}

	if len(share.Recipients) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(share.Recipients))
	}

	if share.Recipients[0] != "user@example.com" {
		t.Errorf("Expected recipient to be 'user@example.com', got %s", share.Recipients[0])
	}
}