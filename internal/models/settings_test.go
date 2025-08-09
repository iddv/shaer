package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultApplicationSettings(t *testing.T) {
	settings := DefaultApplicationSettings()
	
	// Verify default values
	assert.Equal(t, "us-west-2", settings.AWSRegion)
	assert.Equal(t, "", settings.S3Bucket)
	assert.Equal(t, "1d", settings.DefaultExpiration)
	assert.Equal(t, int64(100*1024*1024), settings.MaxFileSize) // 100MB
	assert.Equal(t, "auto", settings.UITheme)
	assert.True(t, settings.AutoRefresh)
	assert.True(t, settings.ShowNotifications)
	assert.False(t, settings.LastUpdated.IsZero())
}

func TestApplicationSettings_ToJSON_FromJSON(t *testing.T) {
	// Create test settings
	original := &ApplicationSettings{
		AWSRegion:         "eu-west-1",
		S3Bucket:          "test-bucket",
		DefaultExpiration: "1w",
		MaxFileSize:       50 * 1024 * 1024,
		UITheme:           "dark",
		AutoRefresh:       false,
		ShowNotifications: true,
		LastUpdated:       time.Now().Truncate(time.Second), // Truncate for comparison
	}
	
	// Convert to JSON
	jsonStr, err := original.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonStr)
	
	// Convert back from JSON
	restored := &ApplicationSettings{}
	err = restored.FromJSON(jsonStr)
	require.NoError(t, err)
	
	// Verify all fields match
	assert.Equal(t, original.AWSRegion, restored.AWSRegion)
	assert.Equal(t, original.S3Bucket, restored.S3Bucket)
	assert.Equal(t, original.DefaultExpiration, restored.DefaultExpiration)
	assert.Equal(t, original.MaxFileSize, restored.MaxFileSize)
	assert.Equal(t, original.UITheme, restored.UITheme)
	assert.Equal(t, original.AutoRefresh, restored.AutoRefresh)
	assert.Equal(t, original.ShowNotifications, restored.ShowNotifications)
	assert.True(t, original.LastUpdated.Equal(restored.LastUpdated))
}

func TestApplicationSettings_GetExpirationDuration(t *testing.T) {
	tests := []struct {
		expiration string
		expected   time.Duration
	}{
		{"1h", time.Hour},
		{"1d", 24 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
		{"1m", 30 * 24 * time.Hour},
		{"invalid", 24 * time.Hour}, // default to 1 day
		{"", 24 * time.Hour},        // default to 1 day
	}
	
	for _, tt := range tests {
		t.Run(tt.expiration, func(t *testing.T) {
			settings := &ApplicationSettings{
				DefaultExpiration: tt.expiration,
			}
			
			duration := settings.GetExpirationDuration()
			assert.Equal(t, tt.expected, duration)
		})
	}
}

func TestApplicationSettings_Validate(t *testing.T) {
	tests := []struct {
		name        string
		settings    *ApplicationSettings
		expectError bool
		errorField  string
	}{
		{
			name: "valid settings",
			settings: &ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "auto",
				AutoRefresh:       true,
				ShowNotifications: true,
			},
			expectError: false,
		},
		{
			name: "empty AWS region",
			settings: &ApplicationSettings{
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
			name: "empty S3 bucket (allowed in basic validation)",
			settings: &ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
			},
			expectError: false, // Basic validation allows empty S3 bucket
		},
		{
			name: "invalid expiration format",
			settings: &ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "2d", // invalid format
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
			},
			expectError: true,
			errorField:  "default_expiration",
		},
		{
			name: "zero max file size",
			settings: &ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "1d",
				MaxFileSize:       0,
				UITheme:           "light",
			},
			expectError: true,
			errorField:  "max_file_size",
		},
		{
			name: "negative max file size",
			settings: &ApplicationSettings{
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
			settings: &ApplicationSettings{
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
			err := tt.settings.Validate()
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorField != "" {
					validationErr, ok := err.(*ValidationError)
					require.True(t, ok, "Expected ValidationError")
					assert.Equal(t, tt.errorField, validationErr.Field)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "Test error message",
	}
	
	assert.Equal(t, "Test error message", err.Error())
}

func TestApplicationSettings_JSONErrorHandling(t *testing.T) {
	// Test invalid JSON
	settings := &ApplicationSettings{}
	err := settings.FromJSON("invalid json")
	assert.Error(t, err)
	
	// Test ToJSON with valid settings
	validSettings := DefaultApplicationSettings()
	jsonStr, err := validSettings.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonStr)
	
	// Verify JSON contains expected fields
	assert.Contains(t, jsonStr, "aws_region")
	assert.Contains(t, jsonStr, "s3_bucket")
	assert.Contains(t, jsonStr, "default_expiration")
	assert.Contains(t, jsonStr, "max_file_size")
	assert.Contains(t, jsonStr, "ui_theme")
	assert.Contains(t, jsonStr, "auto_refresh")
	assert.Contains(t, jsonStr, "show_notifications")
	assert.Contains(t, jsonStr, "last_updated")
}

func TestApplicationSettings_ValidateForSave(t *testing.T) {
	tests := []struct {
		name        string
		settings    *ApplicationSettings
		expectError bool
		errorField  string
	}{
		{
			name: "valid settings for save",
			settings: &ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
			},
			expectError: false,
		},
		{
			name: "empty S3 bucket fails save validation",
			settings: &ApplicationSettings{
				AWSRegion:         "us-west-2",
				S3Bucket:          "",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
			},
			expectError: true,
			errorField:  "s3_bucket",
		},
		{
			name: "empty AWS region fails save validation",
			settings: &ApplicationSettings{
				AWSRegion:         "",
				S3Bucket:          "test-bucket",
				DefaultExpiration: "1d",
				MaxFileSize:       100 * 1024 * 1024,
				UITheme:           "light",
			},
			expectError: true,
			errorField:  "aws_region",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.ValidateForSave()
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorField != "" {
					validationErr, ok := err.(*ValidationError)
					require.True(t, ok, "Expected ValidationError")
					assert.Equal(t, tt.errorField, validationErr.Field)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplicationSettings_EdgeCases(t *testing.T) {
	// Test with boundary values
	settings := &ApplicationSettings{
		AWSRegion:         "a", // minimum valid region
		S3Bucket:          "b", // minimum valid bucket
		DefaultExpiration: "1h",
		MaxFileSize:       1, // minimum positive size
		UITheme:           "light",
		AutoRefresh:       false,
		ShowNotifications: false,
		LastUpdated:       time.Time{}, // zero time
	}
	
	// Should be valid for both basic and save validation
	err := settings.Validate()
	assert.NoError(t, err)
	
	err = settings.ValidateForSave()
	assert.NoError(t, err)
	
	// Test JSON serialization with edge cases
	jsonStr, err := settings.ToJSON()
	assert.NoError(t, err)
	
	restored := &ApplicationSettings{}
	err = restored.FromJSON(jsonStr)
	assert.NoError(t, err)
	
	assert.Equal(t, settings.AWSRegion, restored.AWSRegion)
	assert.Equal(t, settings.S3Bucket, restored.S3Bucket)
	assert.Equal(t, settings.MaxFileSize, restored.MaxFileSize)
}