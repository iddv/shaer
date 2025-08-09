package models

import (
	"encoding/json"
	"time"
)

// ApplicationSettings represents user preferences stored locally
type ApplicationSettings struct {
	// AWS Configuration
	AWSRegion   string `json:"aws_region"`
	S3Bucket    string `json:"s3_bucket"`
	
	// Default Settings
	DefaultExpiration string `json:"default_expiration"` // "1h", "1d", "1w", "1m"
	MaxFileSize       int64  `json:"max_file_size"`      // in bytes
	
	// UI Settings
	UITheme           string `json:"ui_theme"`           // "light", "dark", "auto"
	
	// Application Settings
	AutoRefresh       bool   `json:"auto_refresh"`       // auto refresh file list
	ShowNotifications bool   `json:"show_notifications"` // show system notifications
	
	// Internal tracking
	LastUpdated       time.Time `json:"last_updated"`
}

// DefaultApplicationSettings returns the default application settings
func DefaultApplicationSettings() *ApplicationSettings {
	return &ApplicationSettings{
		AWSRegion:         "us-west-2",
		S3Bucket:          "",
		DefaultExpiration: "1d",
		MaxFileSize:       100 * 1024 * 1024, // 100MB
		UITheme:           "auto",
		AutoRefresh:       true,
		ShowNotifications: true,
		LastUpdated:       time.Now(),
	}
}

// ToJSON converts settings to JSON string for database storage
func (s *ApplicationSettings) ToJSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON loads settings from JSON string
func (s *ApplicationSettings) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), s)
}

// GetExpirationDuration converts the string expiration to time.Duration
func (s *ApplicationSettings) GetExpirationDuration() time.Duration {
	switch s.DefaultExpiration {
	case "1h":
		return time.Hour
	case "1d":
		return 24 * time.Hour
	case "1w":
		return 7 * 24 * time.Hour
	case "1m":
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour // default to 1 day
	}
}

// Validate checks if the settings are valid
func (s *ApplicationSettings) Validate() error {
	// Validate AWS region format (basic check)
	if s.AWSRegion == "" {
		return &ValidationError{Field: "aws_region", Message: "AWS region cannot be empty"}
	}
	
	// Validate expiration format
	validExpirations := map[string]bool{
		"1h": true, "1d": true, "1w": true, "1m": true,
	}
	if !validExpirations[s.DefaultExpiration] {
		return &ValidationError{Field: "default_expiration", Message: "Invalid expiration format"}
	}
	
	// Validate max file size
	if s.MaxFileSize <= 0 {
		return &ValidationError{Field: "max_file_size", Message: "Max file size must be positive"}
	}
	
	// Validate UI theme
	validThemes := map[string]bool{
		"light": true, "dark": true, "auto": true,
	}
	if !validThemes[s.UITheme] {
		return &ValidationError{Field: "ui_theme", Message: "Invalid UI theme"}
	}
	
	return nil
}

// ValidateForSave checks if the settings are valid for saving (stricter validation)
func (s *ApplicationSettings) ValidateForSave() error {
	if s == nil {
		return &ValidationError{Field: "settings", Message: "settings cannot be nil"}
	}
	
	// First run basic validation
	if err := s.Validate(); err != nil {
		return err
	}
	
	// Additional validation for saving - require S3 bucket
	if s.S3Bucket == "" {
		return &ValidationError{Field: "s3_bucket", Message: "S3 bucket name cannot be empty"}
	}
	
	return nil
}

// ValidationError represents a settings validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}