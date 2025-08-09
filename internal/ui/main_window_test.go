package ui

import (
	"testing"
	"time"

	"file-sharing-app/internal/models"

	"fyne.io/fyne/v2/test"
)

func TestMainWindow_Creation(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create main window
	mainWindow := NewMainWindow(testApp)
	if mainWindow == nil {
		t.Fatal("Failed to create main window")
	}

	// Test that window is properly initialized
	if mainWindow.window == nil {
		t.Error("Window not initialized")
	}

	if mainWindow.fileList == nil {
		t.Error("File list not initialized")
	}

	if mainWindow.statusLabel == nil {
		t.Error("Status label not initialized")
	}

	if mainWindow.uploadBtn == nil {
		t.Error("Upload button not initialized")
	}
}

func TestMainWindow_UpdateFiles(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create main window
	mainWindow := NewMainWindow(testApp)

	// Test updating files
	testFiles := []models.FileMetadata{
		{
			ID:             "test-1",
			FileName:       "test.txt",
			FileSize:       1024,
			UploadDate:     time.Now(),
			ExpirationDate: time.Now().Add(24 * time.Hour),
			Status:         models.StatusActive,
		},
	}

	mainWindow.UpdateFiles(testFiles)

	if len(mainWindow.files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(mainWindow.files))
	}

	if mainWindow.files[0].FileName != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", mainWindow.files[0].FileName)
	}
}

func TestMainWindow_SetStatus(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create main window
	mainWindow := NewMainWindow(testApp)

	// Test setting status
	testStatus := "Test status message"
	mainWindow.SetStatus(testStatus)

	if mainWindow.statusLabel.Text != testStatus {
		t.Errorf("Expected status '%s', got '%s'", testStatus, mainWindow.statusLabel.Text)
	}
}

func TestMainWindow_EnableActions(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create main window
	mainWindow := NewMainWindow(testApp)

	// Test enabling actions
	mainWindow.EnableActions(true)
	if mainWindow.uploadBtn.Disabled() {
		t.Error("Upload button should be enabled")
	}

	// Test disabling actions
	mainWindow.EnableActions(false)
	if !mainWindow.uploadBtn.Disabled() {
		t.Error("Upload button should be disabled")
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, test := range tests {
		result := formatFileSize(test.bytes)
		if result != test.expected {
			t.Errorf("formatFileSize(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status   models.FileStatus
		expected string
	}{
		{models.StatusUploading, "Uploading..."},
		{models.StatusActive, "Ready"},
		{models.StatusExpired, "Expired"},
		{models.StatusDeleted, "Deleted"},
		{models.StatusError, "Error"},
	}

	for _, test := range tests {
		result := formatStatus(test.status)
		if result != test.expected {
			t.Errorf("formatStatus(%s) = %s, expected %s", test.status, result, test.expected)
		}
	}
}