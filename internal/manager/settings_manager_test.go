package manager

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsManager_LoadSettings_DefaultsWhenNotExists(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()
	
	// Create settings manager
	sm := NewSettingsManager(db)
	
	// Load settings (should return defaults since none exist)
	settings, err := sm.LoadSettings()
	require.NoError(t, err)
	
	// Verify default values
	expected := models.DefaultApplicationSettings()
	assert.Equal(t, expected.AWSRegion, settings.AWSRegion)
	assert.Equal(t, expected.S3Bucket, settings.S3Bucket)
	assert.Equal(t, expected.DefaultExpiration, settings.DefaultExpiration)
	assert.Equal(t, expected.MaxFileSize, settings.MaxFileSize)
	assert.Equal(t, expected.UITheme, settings.UITheme)
	assert.Equal(t, expected.AutoRefresh, settings.AutoRefresh)
	assert.Equal(t, expected.ShowNotifications, settings.ShowNotifications)
}

func TestSettingsManager_SaveAndLoadSettings(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()
	
	// Create settings manager
	sm := NewSettingsManager(db)
	
	// Create test settings
	testSettings := &models.ApplicationSettings{
		AWSRegion:         "eu-west-1",
		S3Bucket:          "test-bucket",
		DefaultExpiration: "1w",
		MaxFileSize:       50 * 1024 * 1024, // 50MB
		UITheme:           "dark",
		AutoRefresh:       false,
		ShowNotifications: false,
		LastUpdated:       time.Now(),
	}
	
	// Save settings
	err = sm.SaveSettings(testSettings)
	require.NoError(t, err)
	
	// Load settings
	loadedSettings, err := sm.LoadSettings()
	require.NoError(t, err)
	
	// Verify loaded settings match saved settings
	assert.Equal(t, testSettings.AWSRegion, loadedSettings.AWSRegion)
	assert.Equal(t, testSettings.S3Bucket, loadedSettings.S3Bucket)
	assert.Equal(t, testSettings.DefaultExpiration, loadedSettings.DefaultExpiration)
	assert.Equal(t, testSettings.MaxFileSize, loadedSettings.MaxFileSize)
	assert.Equal(t, testSettings.UITheme, loadedSettings.UITheme)
	assert.Equal(t, testSettings.AutoRefresh, loadedSettings.AutoRefresh)
	assert.Equal(t, testSettings.ShowNotifications, loadedSettings.ShowNotifications)
	
	// LastUpdated should be updated during save
	assert.True(t, loadedSettings.LastUpdated.After(testSettings.LastUpdated) || 
		loadedSettings.LastUpdated.Equal(testSettings.LastUpdated))
}

func TestSettingsManager_ValidateSettings(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()
	
	sm := NewSettingsManager(db)
	
	tests := []struct {
		name        string
		settings    *models.ApplicationSettings
		expectError bool
		errorField  string
	}{
		{
			name: "valid settings",
			settings: &models.ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
				AutoRefresh:       true,
				ShowNotifications: true,
			},
			expectError: false,
		},
		{
			name: "empty AWS region",
			settings: &models.ApplicationSettings{
				AWSRegion:         "",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
			},
			expectError: true,
			errorField:  "aws_region",
		},
		{
			name: "empty S3 bucket (basic validation allows this)",
			settings: &models.ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
			},
			expectError: false, // Basic validation allows empty S3 bucket
		},
		{
			name: "invalid expiration",
			settings: &models.ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "invalid",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
			},
			expectError: true,
			errorField:  "default_expiration",
		},
		{
			name: "invalid max file size",
			settings: &models.ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "1d",
				MaxFileSize:       -1,
				UITheme:           "light",
			},
			expectError: true,
			errorField:  "max_file_size",
		},
		{
			name: "invalid UI theme",
			settings: &models.ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "invalid",
			},
			expectError: true,
			errorField:  "ui_theme",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.ValidateSettings(tt.settings)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorField != "" {
					validationErr, ok := err.(*models.ValidationError)
					if ok {
						assert.Equal(t, tt.errorField, validationErr.Field)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettingsManager_UpdateSetting(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()
	
	sm := NewSettingsManager(db)
	
	// First save valid initial settings so we can update them
	initialSettings := &models.ApplicationSettings{
		AWSRegion:         "us-west-2",
		S3Bucket:          "initial-bucket",
		DefaultExpiration: "1d",
		MaxFileSize:       100 * 1024 * 1024,
		UITheme:           "light",
		AutoRefresh:       true,
		ShowNotifications: true,
	}
	err = sm.SaveSettings(initialSettings)
	require.NoError(t, err)
	
	// Test updating various settings
	tests := []struct {
		key   string
		value string
	}{
		{"aws_region", "eu-central-1"},
		{"s3_bucket", "updated-bucket"},
		{"default_expiration", "1w"},
		{"ui_theme", "dark"},
		{"auto_refresh", "false"},
		{"show_notifications", "false"},
	}
	
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// Update setting
			err := sm.UpdateSetting(tt.key, tt.value)
			require.NoError(t, err)
			
			// Verify setting was updated
			value, err := sm.GetSetting(tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.value, value)
		})
	}
}

func TestSettingsManager_GetSetting(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()
	
	sm := NewSettingsManager(db)
	
	// Load default settings first
	_, err = sm.LoadSettings()
	require.NoError(t, err)
	
	// Test getting various settings
	tests := []struct {
		key           string
		expectedValue string
	}{
		{"aws_region", "us-west-2"},
		{"s3_bucket", ""},
		{"default_expiration", "1d"},
		{"ui_theme", "auto"},
		{"auto_refresh", "true"},
		{"show_notifications", "true"},
	}
	
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value, err := sm.GetSetting(tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
	
	// Test getting unknown setting
	_, err = sm.GetSetting("unknown_key")
	assert.Error(t, err)
}

func TestSettingsManager_SaveSettings_ValidationForSave(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()
	
	sm := NewSettingsManager(db)
	
	// Try to save settings with empty S3 bucket (should fail save validation)
	settingsWithEmptyBucket := &models.ApplicationSettings{
		AWSRegion:         "us-west-2",
		S3Bucket:          "", // Empty bucket should fail save validation
		DefaultExpiration: "1d",
		MaxFileSize:       100 * 1024 * 1024,
		UITheme:           "light",
		AutoRefresh:       true,
		ShowNotifications: true,
	}
	
	err = sm.SaveSettings(settingsWithEmptyBucket)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "S3 bucket name cannot be empty")
}

func TestSettingsManager_ResetToDefaults(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()
	
	sm := NewSettingsManager(db)
	
	// Save custom settings
	customSettings := &models.ApplicationSettings{
		AWSRegion:         "eu-west-1",
		S3Bucket:          "custom-bucket",
		DefaultExpiration: "1w",
		MaxFileSize:       50 * 1024 * 1024,
		UITheme:           "dark",
		AutoRefresh:       false,
		ShowNotifications: false,
		LastUpdated:       time.Now(),
	}
	
	err = sm.SaveSettings(customSettings)
	require.NoError(t, err)
	
	// Reset to defaults
	err = sm.ResetToDefaults()
	require.NoError(t, err)
	
	// Load settings and verify they are defaults (with placeholder bucket)
	settings, err := sm.LoadSettings()
	require.NoError(t, err)
	
	defaults := models.DefaultApplicationSettings()
	assert.Equal(t, defaults.AWSRegion, settings.AWSRegion)
	assert.Equal(t, "your-bucket-name-here", settings.S3Bucket) // Placeholder bucket name
	assert.Equal(t, defaults.DefaultExpiration, settings.DefaultExpiration)
	assert.Equal(t, defaults.MaxFileSize, settings.MaxFileSize)
	assert.Equal(t, defaults.UITheme, settings.UITheme)
	assert.Equal(t, defaults.AutoRefresh, settings.AutoRefresh)
	assert.Equal(t, defaults.ShowNotifications, settings.ShowNotifications)
}

func TestSettingsManager_NilSettings(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()
	
	sm := NewSettingsManager(db)
	
	// Test validating nil settings
	err = sm.ValidateSettings(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "settings cannot be nil")
	
	// Test saving nil settings
	err = sm.SaveSettings(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "settings cannot be nil")
}

func TestSettingsManager_DatabaseError(t *testing.T) {
	// Create temporary database and then remove it to simulate database errors
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := storage.NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	
	sm := NewSettingsManager(db)
	
	// Close database to simulate error
	db.Close()
	
	// Remove database file
	os.Remove(dbPath)
	
	// Test operations with closed database
	_, err = sm.LoadSettings()
	// Should return defaults even if database fails
	assert.NoError(t, err)
	
	settings := models.DefaultApplicationSettings()
	err = sm.SaveSettings(settings)
	assert.Error(t, err)
}