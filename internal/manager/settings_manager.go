package manager

import (
	"fmt"
	"time"

	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
)

const (
	settingsConfigKey = "application_settings"
)

// SettingsManager interface defines the contract for settings management
type SettingsManager interface {
	LoadSettings() (*models.ApplicationSettings, error)
	SaveSettings(settings *models.ApplicationSettings) error
	GetDefaultSettings() *models.ApplicationSettings
	ValidateSettings(settings *models.ApplicationSettings) error
}

// SettingsManagerImpl implements the SettingsManager interface
type SettingsManagerImpl struct {
	db storage.Database
}

// NewSettingsManager creates a new settings manager
func NewSettingsManager(db storage.Database) *SettingsManagerImpl {
	return &SettingsManagerImpl{
		db: db,
	}
}

// LoadSettings loads application settings from the database
func (sm *SettingsManagerImpl) LoadSettings() (*models.ApplicationSettings, error) {
	// Try to load settings from database
	settingsJSON, err := sm.db.GetConfig(settingsConfigKey)
	if err != nil {
		// If settings don't exist, return default settings without saving them
		// (they will be saved when user first configures them)
		return models.DefaultApplicationSettings(), nil
	}
	
	// Parse settings from JSON
	settings := &models.ApplicationSettings{}
	if err := settings.FromJSON(settingsJSON); err != nil {
		return nil, fmt.Errorf("failed to parse settings from database: %w", err)
	}
	
	return settings, nil
}

// SaveSettings saves application settings to the database
func (sm *SettingsManagerImpl) SaveSettings(settings *models.ApplicationSettings) error {
	// Validate settings before saving (use stricter validation for save)
	if err := settings.ValidateForSave(); err != nil {
		return fmt.Errorf("settings validation failed: %w", err)
	}
	
	// Update last updated timestamp
	settings.LastUpdated = time.Now()
	
	// Convert settings to JSON
	settingsJSON, err := settings.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize settings: %w", err)
	}
	
	// Save to database
	if err := sm.db.SaveConfig(settingsConfigKey, settingsJSON); err != nil {
		return fmt.Errorf("failed to save settings to database: %w", err)
	}
	
	return nil
}

// GetDefaultSettings returns the default application settings
func (sm *SettingsManagerImpl) GetDefaultSettings() *models.ApplicationSettings {
	return models.DefaultApplicationSettings()
}

// ValidateSettings validates the provided settings (basic validation)
func (sm *SettingsManagerImpl) ValidateSettings(settings *models.ApplicationSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}
	
	return settings.Validate()
}

// UpdateSetting updates a specific setting by key
func (sm *SettingsManagerImpl) UpdateSetting(key, value string) error {
	// Load current settings
	settings, err := sm.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load current settings: %w", err)
	}
	
	// Update the specific setting
	switch key {
	case "aws_region":
		settings.AWSRegion = value
	case "s3_bucket":
		settings.S3Bucket = value
	case "default_expiration":
		settings.DefaultExpiration = value
	case "ui_theme":
		settings.UITheme = value
	case "auto_refresh":
		settings.AutoRefresh = value == "true"
	case "show_notifications":
		settings.ShowNotifications = value == "true"
	default:
		return fmt.Errorf("unknown setting key: %s", key)
	}
	
	// Save updated settings
	return sm.SaveSettings(settings)
}

// GetSetting retrieves a specific setting by key
func (sm *SettingsManagerImpl) GetSetting(key string) (string, error) {
	settings, err := sm.LoadSettings()
	if err != nil {
		return "", fmt.Errorf("failed to load settings: %w", err)
	}
	
	switch key {
	case "aws_region":
		return settings.AWSRegion, nil
	case "s3_bucket":
		return settings.S3Bucket, nil
	case "default_expiration":
		return settings.DefaultExpiration, nil
	case "ui_theme":
		return settings.UITheme, nil
	case "auto_refresh":
		if settings.AutoRefresh {
			return "true", nil
		}
		return "false", nil
	case "show_notifications":
		if settings.ShowNotifications {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("unknown setting key: %s", key)
	}
}

// ResetToDefaults resets all settings to their default values
func (sm *SettingsManagerImpl) ResetToDefaults() error {
	// Create default settings with a placeholder bucket name for saving
	defaultSettings := models.DefaultApplicationSettings()
	defaultSettings.S3Bucket = "your-bucket-name-here" // Placeholder that passes validation
	return sm.SaveSettings(defaultSettings)
}