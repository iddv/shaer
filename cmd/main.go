package main

import (
	"time"

	"file-sharing-app/internal/config"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/ui"
	"file-sharing-app/pkg/logger"

	"fyne.io/fyne/v2/app"
)

func main() {
	// Initialize logging
	log := logger.New()
	log.Info("File Sharing App starting...")

	// Load default configuration
	_ = config.DefaultConfig()
	log.Info("Configuration loaded")

	// Create Fyne application
	myApp := app.New()

	// Create main window
	mainWindow := ui.NewMainWindow(myApp)
	
	// Set up callback functions (placeholders for now - will be connected to business logic in task 11)
	mainWindow.OnUploadFile = func(filePath string, expiration time.Duration) error {
		log.Info("Upload file callback called (placeholder)")
		// This will be implemented in task 11 when integrating with business logic
		return nil
	}
	
	mainWindow.OnShareFile = func(fileID string, recipients []string, message string) error {
		log.Info("Share file callback called (placeholder)")
		// This will be implemented in task 11 when integrating with business logic
		return nil
	}
	
	mainWindow.OnDeleteFile = func(fileID string) error {
		log.Info("Delete file callback called (placeholder)")
		// This will be implemented in task 11 when integrating with business logic
		return nil
	}
	
	mainWindow.OnRefreshFiles = func() ([]models.FileMetadata, error) {
		log.Info("Refresh files callback called (placeholder)")
		// This will be implemented in task 11 when integrating with business logic
		// For now, return some sample data for UI testing
		return []models.FileMetadata{
			{
				ID:             "sample-1",
				FileName:       "sample-document.pdf",
				FileSize:       1024 * 1024, // 1MB
				UploadDate:     time.Now().Add(-2 * time.Hour),
				ExpirationDate: time.Now().Add(22 * time.Hour),
				Status:         models.StatusActive,
			},
			{
				ID:             "sample-2", 
				FileName:       "uploading-file.jpg",
				FileSize:       2 * 1024 * 1024, // 2MB
				UploadDate:     time.Now().Add(-5 * time.Minute),
				ExpirationDate: time.Now().Add(7 * 24 * time.Hour),
				Status:         models.StatusUploading,
			},
		}, nil
	}
	
	// Initialize with sample data
	if files, err := mainWindow.OnRefreshFiles(); err == nil {
		mainWindow.UpdateFiles(files)
		mainWindow.EnableActions(true)
		mainWindow.SetStatus("Ready - Sample data loaded")
	}

	log.Info("Application UI initialized")
	mainWindow.Show()
}