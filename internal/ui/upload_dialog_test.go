package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
)

func TestFileUploadDialog_Creation(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create test window
	testWindow := testApp.NewWindow("Test")

	// Create upload dialog
	uploadDialog := NewFileUploadDialog(testWindow, func(filePath string, expiration time.Duration) error {
		return nil
	})

	if uploadDialog == nil {
		t.Fatal("Failed to create upload dialog")
	}

	if uploadDialog.dialog == nil {
		t.Error("Dialog not initialized")
	}

	if uploadDialog.fileLabel == nil {
		t.Error("File label not initialized")
	}

	if uploadDialog.expirationSelect == nil {
		t.Error("Expiration select not initialized")
	}

	if uploadDialog.uploadBtn == nil {
		t.Error("Upload button not initialized")
	}
}

func TestFileUploadDialog_ParseExpiration(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create test window
	testWindow := testApp.NewWindow("Test")

	// Create upload dialog
	uploadDialog := NewFileUploadDialog(testWindow, nil)

	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1 hour", time.Hour},
		{"1 day", 24 * time.Hour},
		{"1 week", 7 * 24 * time.Hour},
		{"1 month", 30 * 24 * time.Hour},
		{"invalid", 24 * time.Hour}, // Default
	}

	for _, test := range tests {
		result := uploadDialog.parseExpiration(test.input)
		if result != test.expected {
			t.Errorf("parseExpiration(%s) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestFileUploadDialog_SetProgress(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create test window
	testWindow := testApp.NewWindow("Test")

	// Create upload dialog
	uploadDialog := NewFileUploadDialog(testWindow, nil)

	// Test setting progress
	uploadDialog.SetProgress(0.5)
	if uploadDialog.progressBar.Value != 0.5 {
		t.Errorf("Expected progress 0.5, got %f", uploadDialog.progressBar.Value)
	}

	// Test completion
	uploadDialog.SetProgress(1.0)
	if uploadDialog.progressBar.Value != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", uploadDialog.progressBar.Value)
	}
}