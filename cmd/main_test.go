package main

import (
	"file-sharing-app/internal/config"
	"file-sharing-app/pkg/logger"
	"testing"
)

func TestBasicSetup(t *testing.T) {
	// Test logger creation
	log := logger.New()
	if log == nil {
		t.Fatal("Failed to create logger")
	}

	// Test configuration loading
	cfg := config.DefaultConfig()
	if cfg == nil {
		t.Fatal("Failed to load default configuration")
	}

	// Verify default configuration values
	if cfg.MaxFileSize != 100*1024*1024 {
		t.Errorf("Expected MaxFileSize to be 100MB, got %d", cfg.MaxFileSize)
	}

	if cfg.DefaultExpiration != "24h" {
		t.Errorf("Expected DefaultExpiration to be '24h', got %s", cfg.DefaultExpiration)
	}

	if cfg.UITheme != "light" {
		t.Errorf("Expected UITheme to be 'light', got %s", cfg.UITheme)
	}
}