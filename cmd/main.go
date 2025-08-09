package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"file-sharing-app/internal/app"
	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/config"
	"file-sharing-app/internal/manager"
	"file-sharing-app/internal/storage"
	"file-sharing-app/internal/ui"
	"file-sharing-app/pkg/logger"

	fyneApp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	// Initialize logging
	log := logger.New()
	log.Info("File Sharing App starting...")

	// Load configuration
	cfg := config.DefaultConfig()
	log.Info("Configuration loaded")

	// Create Fyne application
	myApp := fyneApp.New()

	// Create main window
	mainWindow := ui.NewMainWindow(myApp)

	// Initialize application components
	controller, err := initializeApplication(cfg, mainWindow, log)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to initialize application: %v", err))
		
		// Show error dialog and exit gracefully
		dialog.ShowError(fmt.Errorf("Failed to initialize application: %v", err), mainWindow.GetWindow())
		mainWindow.SetStatus("Initialization failed: " + err.Error())
		mainWindow.EnableActions(false)
		
		// Still show the window so user can see the error
		mainWindow.Show()
		return
	}

	// Start the controller
	if err := controller.Start(); err != nil {
		log.Error(fmt.Sprintf("Failed to start application controller: %v", err))
		dialog.ShowError(fmt.Errorf("Failed to start application: %v", err), mainWindow.GetWindow())
		mainWindow.SetStatus("Startup failed: " + err.Error())
		mainWindow.EnableActions(false)
	}

	log.Info("Application initialized successfully")
	
	// Show the main window (this blocks until the application exits)
	mainWindow.Show()
	
	// Cleanup when application exits
	controller.Stop()
	log.Info("Application shutdown complete")
}

// initializeApplication sets up all the application components
func initializeApplication(cfg *config.AppConfig, mainWindow *ui.MainWindow, log *logger.Logger) (*app.Controller, error) {
	// Initialize database
	database, err := initializeDatabase(log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize AWS services (with graceful fallback if credentials not configured)
	s3Service, credentialsConfigured, err := initializeAWSServices(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS services: %w", err)
	}

	// Initialize business logic managers
	fileManager := manager.NewFileManager(database, s3Service)
	shareManager := manager.NewShareManager(database, s3Service)
	expirationManager := manager.NewExpirationManager(database)

	// Create application controller
	controller := app.NewController(fileManager, shareManager, expirationManager, mainWindow)

	// Update UI status based on AWS credentials configuration
	if !credentialsConfigured {
		mainWindow.SetStatus("AWS credentials not configured - some features will be limited")
		log.Info("AWS credentials not configured - running in limited mode")
	}

	return controller, nil
}

// initializeDatabase sets up the SQLite database
func initializeDatabase(log *logger.Logger) (storage.Database, error) {
	// Create data directory if it doesn't exist
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize database
	dbPath := filepath.Join(dataDir, "file-sharing-app.db")
	database, err := storage.NewSQLiteDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	log.Info(fmt.Sprintf("Database initialized: %s", dbPath))
	return database, nil
}

// initializeAWSServices sets up AWS S3 service and credential provider
func initializeAWSServices(cfg *config.AppConfig, log *logger.Logger) (aws.S3Service, bool, error) {
	// Try to initialize credential provider
	credProvider, err := aws.NewSecureCredentialProvider()
	if err != nil {
		log.Info(fmt.Sprintf("AWS credential provider initialization failed: %v", err))
		// Return nil service but don't fail - app can run in limited mode
		return nil, false, nil
	}

	// Validate credentials
	if err := credProvider.ValidateCredentials(context.Background()); err != nil {
		log.Info(fmt.Sprintf("AWS credentials validation failed: %v", err))
		// Return nil service but don't fail - app can run in limited mode
		return nil, false, nil
	}

	// Initialize S3 service
	s3Service, err := aws.NewS3Service(credProvider, cfg.S3Bucket)
	if err != nil {
		log.Info(fmt.Sprintf("S3 service initialization failed: %v", err))
		// Return nil service but don't fail - app can run in limited mode
		return nil, false, nil
	}

	log.Info("AWS services initialized successfully")
	return s3Service, true, nil
}