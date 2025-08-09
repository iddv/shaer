package ui

import (
	"testing"
	"time"

	"file-sharing-app/internal/models"

	"fyne.io/fyne/v2/test"
)

func TestSharingDialog_Creation(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create test window
	testWindow := testApp.NewWindow("Test")

	// Create test file
	testFile := models.FileMetadata{
		ID:             "test-1",
		FileName:       "test.txt",
		FileSize:       1024,
		UploadDate:     time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		Status:         models.StatusActive,
	}

	// Create sharing dialog
	sharingDialog := NewSharingDialog(testWindow, testFile, func(fileID string, recipients []string, message string) error {
		return nil
	})

	if sharingDialog == nil {
		t.Fatal("Failed to create sharing dialog")
	}

	if sharingDialog.dialog == nil {
		t.Error("Dialog not initialized")
	}

	if sharingDialog.emailEntry == nil {
		t.Error("Email entry not initialized")
	}

	if sharingDialog.recipientList == nil {
		t.Error("Recipient list not initialized")
	}

	if sharingDialog.shareBtn == nil {
		t.Error("Share button not initialized")
	}
}

func TestSharingDialog_EmailValidation(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create test window
	testWindow := testApp.NewWindow("Test")

	// Create test file
	testFile := models.FileMetadata{
		ID:       "test-1",
		FileName: "test.txt",
	}

	// Create sharing dialog
	sharingDialog := NewSharingDialog(testWindow, testFile, nil)

	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"invalid-email", false},
		{"@domain.com", false},
		{"user@", false},
		{"", false},
	}

	for _, test := range tests {
		result := sharingDialog.isValidEmail(test.email)
		if result != test.expected {
			t.Errorf("isValidEmail(%s) = %v, expected %v", test.email, result, test.expected)
		}
	}
}

func TestSharingDialog_AddRecipient(t *testing.T) {
	// Create test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create test window
	testWindow := testApp.NewWindow("Test")

	// Create test file
	testFile := models.FileMetadata{
		ID:       "test-1",
		FileName: "test.txt",
	}

	// Create sharing dialog
	sharingDialog := NewSharingDialog(testWindow, testFile, nil)

	// Test adding valid email
	sharingDialog.emailEntry.SetText("test@example.com")
	initialCount := len(sharingDialog.recipients)
	sharingDialog.addRecipient()

	if len(sharingDialog.recipients) != initialCount+1 {
		t.Errorf("Expected %d recipients, got %d", initialCount+1, len(sharingDialog.recipients))
	}

	if sharingDialog.recipients[0] != "test@example.com" {
		t.Errorf("Expected recipient 'test@example.com', got '%s'", sharingDialog.recipients[0])
	}

	// Test that share button is enabled after adding recipient
	if sharingDialog.shareBtn.Disabled() {
		t.Error("Share button should be enabled after adding recipient")
	}
}