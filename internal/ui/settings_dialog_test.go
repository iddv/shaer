package ui

import (
	"errors"
	"testing"
	"time"

	"file-sharing-app/internal/models"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingsDialog(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	
	dialog := NewSettingsDialog(window)
	
	assert.NotNil(t, dialog)
	assert.Equal(t, window, dialog.parent)
	assert.NotNil(t, dialog.dialog)
	assert.NotNil(t, dialog.awsRegionEntry)
	assert.NotNil(t, dialog.s3BucketEntry)
	assert.NotNil(t, dialog.defaultExpirationSelect)
	assert.NotNil(t, dialog.maxFileSizeEntry)
	assert.NotNil(t, dialog.uiThemeSelect)
	assert.NotNil(t, dialog.autoRefreshCheck)
	assert.NotNil(t, dialog.showNotificationsCheck)
}

func TestSettingsDialog_SetCallbacks(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Mock callbacks
	saveCallback := func(settings *models.ApplicationSettings) error {
		return nil
	}
	loadCallback := func() (*models.ApplicationSettings, error) {
		return models.DefaultApplicationSettings(), nil
	}
	
	dialog.SetCallbacks(saveCallback, loadCallback)
	
	assert.NotNil(t, dialog.OnSaveSettings)
	assert.NotNil(t, dialog.OnLoadSettings)
}

func TestSettingsDialog_PopulateForm(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Create test settings
	testSettings := &models.ApplicationSettings{
		AWSRegion:         "eu-west-1",
		S3Bucket:          "test-bucket",
		DefaultExpiration: "1w",
		MaxFileSize:       50 * 1024 * 1024, // 50MB
		UITheme:           "dark",
		AutoRefresh:       false,
		ShowNotifications: true,
		LastUpdated:       time.Now(),
	}
	
	dialog.settings = testSettings
	dialog.populateForm()
	
	// Verify form is populated correctly
	assert.Equal(t, "eu-west-1", dialog.awsRegionEntry.Text)
	assert.Equal(t, "test-bucket", dialog.s3BucketEntry.Text)
	assert.Equal(t, "1w", dialog.defaultExpirationSelect.Selected)
	assert.Equal(t, "50", dialog.maxFileSizeEntry.Text)
	assert.Equal(t, "dark", dialog.uiThemeSelect.Selected)
	assert.False(t, dialog.autoRefreshCheck.Checked)
	assert.True(t, dialog.showNotificationsCheck.Checked)
}

func TestSettingsDialog_ValidateForm(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	tests := []struct {
		name           string
		setupForm      func()
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid form",
			setupForm: func() {
				dialog.awsRegionEntry.SetText("us-west-2")
				dialog.s3BucketEntry.SetText("test-bucket")
				dialog.defaultExpirationSelect.SetSelected("1d")
				dialog.maxFileSizeEntry.SetText("100")
				dialog.uiThemeSelect.SetSelected("light")
			},
			expectError: false,
		},
		{
			name: "empty AWS region",
			setupForm: func() {
				dialog.awsRegionEntry.SetText("")
				dialog.s3BucketEntry.SetText("test-bucket")
				dialog.defaultExpirationSelect.SetSelected("1d")
				dialog.maxFileSizeEntry.SetText("100")
				dialog.uiThemeSelect.SetSelected("light")
			},
			expectError:   true,
			errorContains: "AWS region cannot be empty",
		},
		{
			name: "empty S3 bucket",
			setupForm: func() {
				dialog.awsRegionEntry.SetText("us-west-2")
				dialog.s3BucketEntry.SetText("")
				dialog.defaultExpirationSelect.SetSelected("1d")
				dialog.maxFileSizeEntry.SetText("100")
				dialog.uiThemeSelect.SetSelected("light")
			},
			expectError:   true,
			errorContains: "S3 bucket name cannot be empty",
		},
		{
			name: "no expiration selected",
			setupForm: func() {
				dialog.awsRegionEntry.SetText("us-west-2")
				dialog.s3BucketEntry.SetText("test-bucket")
				// Don't set expiration - it will be empty by default
				dialog.maxFileSizeEntry.SetText("100")
				dialog.uiThemeSelect.SetSelected("light")
			},
			expectError:   true,
			errorContains: "Please select a default expiration period",
		},
		{
			name: "empty max file size",
			setupForm: func() {
				dialog.awsRegionEntry.SetText("us-west-2")
				dialog.s3BucketEntry.SetText("test-bucket")
				dialog.defaultExpirationSelect.SetSelected("1d")
				dialog.maxFileSizeEntry.SetText("")
				dialog.uiThemeSelect.SetSelected("light")
			},
			expectError:   true,
			errorContains: "Max file size cannot be empty",
		},
		{
			name: "no theme selected",
			setupForm: func() {
				dialog.awsRegionEntry.SetText("us-west-2")
				dialog.s3BucketEntry.SetText("test-bucket")
				dialog.defaultExpirationSelect.SetSelected("1d")
				dialog.maxFileSizeEntry.SetText("100")
				// Don't set theme - it will be empty by default
			},
			expectError:   true,
			errorContains: "Please select a UI theme",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupForm()
			
			err := dialog.validateForm()
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettingsDialog_UpdateSettingsFromForm(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Set up form with test values
	dialog.awsRegionEntry.SetText("eu-central-1")
	dialog.s3BucketEntry.SetText("updated-bucket")
	dialog.defaultExpirationSelect.SetSelected("1w")
	dialog.maxFileSizeEntry.SetText("75")
	dialog.uiThemeSelect.SetSelected("dark")
	dialog.autoRefreshCheck.SetChecked(false)
	dialog.showNotificationsCheck.SetChecked(true)
	
	// Initialize settings
	dialog.settings = models.DefaultApplicationSettings()
	
	// Update settings from form
	dialog.updateSettingsFromForm()
	
	// Verify settings were updated
	assert.Equal(t, "eu-central-1", dialog.settings.AWSRegion)
	assert.Equal(t, "updated-bucket", dialog.settings.S3Bucket)
	assert.Equal(t, "1w", dialog.settings.DefaultExpiration)
	assert.Equal(t, int64(75*1024*1024), dialog.settings.MaxFileSize) // 75MB in bytes
	assert.Equal(t, "dark", dialog.settings.UITheme)
	assert.False(t, dialog.settings.AutoRefresh)
	assert.True(t, dialog.settings.ShowNotifications)
}

func TestSettingsDialog_UpdateSettingsFromForm_NilSettings(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Set up form with test values
	dialog.awsRegionEntry.SetText("us-east-1")
	dialog.s3BucketEntry.SetText("test-bucket")
	dialog.defaultExpirationSelect.SetSelected("1h")
	dialog.maxFileSizeEntry.SetText("50")
	dialog.uiThemeSelect.SetSelected("light")
	dialog.autoRefreshCheck.SetChecked(true)
	dialog.showNotificationsCheck.SetChecked(false)
	
	// Don't initialize settings (nil)
	dialog.settings = nil
	
	// Update settings from form
	dialog.updateSettingsFromForm()
	
	// Verify settings were created and updated
	require.NotNil(t, dialog.settings)
	assert.Equal(t, "us-east-1", dialog.settings.AWSRegion)
	assert.Equal(t, "test-bucket", dialog.settings.S3Bucket)
	assert.Equal(t, "1h", dialog.settings.DefaultExpiration)
	assert.Equal(t, int64(50*1024*1024), dialog.settings.MaxFileSize)
	assert.Equal(t, "light", dialog.settings.UITheme)
	assert.True(t, dialog.settings.AutoRefresh)
	assert.False(t, dialog.settings.ShowNotifications)
}

func TestSettingsDialog_Show_WithLoadCallback(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Mock successful load callback
	testSettings := &models.ApplicationSettings{
		AWSRegion:         "ap-southeast-1",
		S3Bucket:          "loaded-bucket",
		DefaultExpiration: "1m",
		MaxFileSize:       200 * 1024 * 1024,
		UITheme:           "dark",
		AutoRefresh:       false,
		ShowNotifications: false,
		LastUpdated:       time.Now(),
	}
	
	loadCallback := func() (*models.ApplicationSettings, error) {
		return testSettings, nil
	}
	
	dialog.SetCallbacks(nil, loadCallback)
	
	// Show dialog (this would normally display the UI)
	dialog.Show()
	
	// Verify settings were loaded and form populated
	assert.Equal(t, testSettings, dialog.settings)
	assert.Equal(t, "ap-southeast-1", dialog.awsRegionEntry.Text)
	assert.Equal(t, "loaded-bucket", dialog.s3BucketEntry.Text)
	assert.Equal(t, "1m", dialog.defaultExpirationSelect.Selected)
	assert.Equal(t, "200", dialog.maxFileSizeEntry.Text)
	assert.Equal(t, "dark", dialog.uiThemeSelect.Selected)
	assert.False(t, dialog.autoRefreshCheck.Checked)
	assert.False(t, dialog.showNotificationsCheck.Checked)
}

func TestSettingsDialog_Show_WithLoadError(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Mock failing load callback
	loadCallback := func() (*models.ApplicationSettings, error) {
		return nil, errors.New("database error")
	}
	
	dialog.SetCallbacks(nil, loadCallback)
	
	// Show dialog
	dialog.Show()
	
	// Verify default settings were used
	defaults := models.DefaultApplicationSettings()
	assert.Equal(t, defaults.AWSRegion, dialog.settings.AWSRegion)
	assert.Equal(t, defaults.S3Bucket, dialog.settings.S3Bucket)
	assert.Equal(t, defaults.DefaultExpiration, dialog.settings.DefaultExpiration)
}

func TestSettingsDialog_Show_NoLoadCallback(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Don't set load callback
	
	// Show dialog
	dialog.Show()
	
	// Verify default settings were used
	defaults := models.DefaultApplicationSettings()
	assert.Equal(t, defaults.AWSRegion, dialog.settings.AWSRegion)
	assert.Equal(t, defaults.S3Bucket, dialog.settings.S3Bucket)
	assert.Equal(t, defaults.DefaultExpiration, dialog.settings.DefaultExpiration)
}

func TestSettingsDialog_SaveSettings_Success(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Set up valid form
	dialog.awsRegionEntry.SetText("us-west-2")
	dialog.s3BucketEntry.SetText("test-bucket")
	dialog.defaultExpirationSelect.SetSelected("1d")
	dialog.maxFileSizeEntry.SetText("100")
	dialog.uiThemeSelect.SetSelected("light")
	
	// Mock successful save callback
	var savedSettings *models.ApplicationSettings
	saveCallback := func(settings *models.ApplicationSettings) error {
		savedSettings = settings
		return nil
	}
	
	dialog.SetCallbacks(saveCallback, nil)
	dialog.settings = models.DefaultApplicationSettings()
	
	// Save settings
	dialog.saveSettings()
	
	// Verify save callback was called with correct settings
	require.NotNil(t, savedSettings)
	assert.Equal(t, "us-west-2", savedSettings.AWSRegion)
	assert.Equal(t, "test-bucket", savedSettings.S3Bucket)
	assert.Equal(t, "1d", savedSettings.DefaultExpiration)
	assert.Equal(t, int64(100*1024*1024), savedSettings.MaxFileSize)
	assert.Equal(t, "light", savedSettings.UITheme)
}

func TestSettingsDialog_SaveSettings_ValidationError(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Set up invalid form (empty AWS region)
	dialog.awsRegionEntry.SetText("")
	dialog.s3BucketEntry.SetText("test-bucket")
	dialog.defaultExpirationSelect.SetSelected("1d")
	dialog.maxFileSizeEntry.SetText("100")
	dialog.uiThemeSelect.SetSelected("light")
	
	// Mock save callback (should not be called)
	saveCallbackCalled := false
	saveCallback := func(settings *models.ApplicationSettings) error {
		saveCallbackCalled = true
		return nil
	}
	
	dialog.SetCallbacks(saveCallback, nil)
	dialog.settings = models.DefaultApplicationSettings()
	
	// Save settings (should fail validation)
	dialog.saveSettings()
	
	// Verify save callback was not called due to validation error
	assert.False(t, saveCallbackCalled)
}

func TestSettingsDialog_SaveSettings_SaveError(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Test")
	dialog := NewSettingsDialog(window)
	
	// Set up valid form
	dialog.awsRegionEntry.SetText("us-west-2")
	dialog.s3BucketEntry.SetText("test-bucket")
	dialog.defaultExpirationSelect.SetSelected("1d")
	dialog.maxFileSizeEntry.SetText("100")
	dialog.uiThemeSelect.SetSelected("light")
	
	// Mock failing save callback
	saveCallback := func(settings *models.ApplicationSettings) error {
		return errors.New("database error")
	}
	
	dialog.SetCallbacks(saveCallback, nil)
	dialog.settings = models.DefaultApplicationSettings()
	
	// Save settings (should handle error gracefully)
	dialog.saveSettings()
	
	// Test passes if no panic occurs
}