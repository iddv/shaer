package config

// AppConfig holds application configuration
type AppConfig struct {
	AWSRegion         string `json:"aws_region"`
	S3Bucket          string `json:"s3_bucket"`
	DefaultExpiration string `json:"default_expiration"`
	MaxFileSize       int64  `json:"max_file_size"`
	UITheme           string `json:"ui_theme"`
}

// DefaultConfig returns default application configuration
func DefaultConfig() *AppConfig {
	return &AppConfig{
		AWSRegion:         "us-east-1",
		S3Bucket:          "",
		DefaultExpiration: "24h",
		MaxFileSize:       100 * 1024 * 1024, // 100MB
		UITheme:           "light",
	}
}