package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// TestConfig holds configuration for integration and e2e tests
type TestConfig struct {
	// AWS Configuration
	S3Bucket          string
	AWSRegion         string
	AWSAccessKeyID    string
	AWSSecretAccessKey string
	
	// Test Configuration
	TestTimeout       time.Duration
	MaxFileSize       int64
	TestFilePrefix    string
	CleanupAfterTests bool
	
	// Security Test Configuration
	MaxPresignedURLExpiration time.Duration
	MinPresignedURLExpiration time.Duration
}

// LoadTestConfig loads test configuration from environment variables
func LoadTestConfig() (*TestConfig, error) {
	config := &TestConfig{
		// Default values
		TestTimeout:               30 * time.Minute,
		MaxFileSize:               100 * 1024 * 1024, // 100MB
		TestFilePrefix:            "integration-test",
		CleanupAfterTests:         true,
		MaxPresignedURLExpiration: 7 * 24 * time.Hour, // 7 days
		MinPresignedURLExpiration: 5 * time.Minute,
	}
	
	// Required environment variables
	requiredVars := map[string]*string{
		"S3_BUCKET":             &config.S3Bucket,
		"AWS_REGION":            &config.AWSRegion,
		"AWS_ACCESS_KEY_ID":     &config.AWSAccessKeyID,
		"AWS_SECRET_ACCESS_KEY": &config.AWSSecretAccessKey,
	}
	
	var missingVars []string
	for envVar, configField := range requiredVars {
		value := os.Getenv(envVar)
		if value == "" {
			missingVars = append(missingVars, envVar)
		} else {
			*configField = value
		}
	}
	
	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}
	
	// Optional environment variables
	if timeoutStr := os.Getenv("TEST_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.TestTimeout = timeout
		}
	}
	
	if maxSizeStr := os.Getenv("MAX_FILE_SIZE"); maxSizeStr != "" {
		if maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64); err == nil {
			config.MaxFileSize = maxSize
		}
	}
	
	if prefix := os.Getenv("TEST_FILE_PREFIX"); prefix != "" {
		config.TestFilePrefix = prefix
	}
	
	if cleanupStr := os.Getenv("CLEANUP_AFTER_TESTS"); cleanupStr != "" {
		if cleanup, err := strconv.ParseBool(cleanupStr); err == nil {
			config.CleanupAfterTests = cleanup
		}
	}
	
	return config, nil
}

// Validate checks if the configuration is valid
func (c *TestConfig) Validate() error {
	if c.S3Bucket == "" {
		return fmt.Errorf("S3 bucket name is required")
	}
	
	if c.AWSRegion == "" {
		return fmt.Errorf("AWS region is required")
	}
	
	if c.AWSAccessKeyID == "" {
		return fmt.Errorf("AWS access key ID is required")
	}
	
	if c.AWSSecretAccessKey == "" {
		return fmt.Errorf("AWS secret access key is required")
	}
	
	if c.TestTimeout <= 0 {
		return fmt.Errorf("test timeout must be positive")
	}
	
	if c.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}
	
	return nil
}

// GetTestKeyPrefix returns a unique prefix for test objects
func (c *TestConfig) GetTestKeyPrefix() string {
	return fmt.Sprintf("%s/%d", c.TestFilePrefix, time.Now().Unix())
}