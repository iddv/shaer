package app

import (
	"context"
	"fmt"
	"time"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/manager"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
	"file-sharing-app/pkg/logger"
)

// MainWindowInterface defines the interface for main window operations
type MainWindowInterface interface {
	SetStatus(status string)
	EnableActions(enabled bool)
	UpdateFiles(files []models.FileMetadata)
	
	// Callback setters
	SetOnUploadFile(callback func(filePath string, expiration time.Duration) error)
	SetOnShareFile(callback func(fileID string, recipients []string, message string) error)
	SetOnDeleteFile(callback func(fileID string) error)
	SetOnRefreshFiles(callback func() ([]models.FileMetadata, error))
	SetOnGeneratePresignedURL(callback func(fileID string, expiration time.Duration) (string, error))
}

// Controller coordinates between UI and business logic layers
type Controller struct {
	// Business logic managers
	fileManager       manager.FileManager
	shareManager      manager.ShareManager
	expirationManager manager.ExpirationManager
	
	// UI components
	mainWindow MainWindowInterface
	
	// Services
	logger *logger.Logger
	
	// Background context for operations
	ctx    context.Context
	cancel context.CancelFunc
}

// NewController creates a new application controller
func NewController(
	fileManager manager.FileManager,
	shareManager manager.ShareManager,
	expirationManager manager.ExpirationManager,
	mainWindow MainWindowInterface,
) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	
	controller := &Controller{
		fileManager:       fileManager,
		shareManager:      shareManager,
		expirationManager: expirationManager,
		mainWindow:        mainWindow,
		logger:            logger.New(),
		ctx:               ctx,
		cancel:            cancel,
	}
	
	// Connect UI callbacks to controller methods
	controller.setupUICallbacks()
	
	return controller
}

// Start initializes the controller and starts background operations
func (c *Controller) Start() error {
	c.logger.Info("Starting application controller")
	
	// Start background expiration checker
	go c.startExpirationChecker()
	
	// Load initial file list
	if err := c.refreshFiles(); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to load initial files: %v", err))
		c.mainWindow.SetStatus("Error loading files: " + err.Error())
		return err
	}
	
	c.mainWindow.SetStatus("Ready")
	c.mainWindow.EnableActions(true)
	
	return nil
}

// Stop gracefully shuts down the controller
func (c *Controller) Stop() {
	c.logger.Info("Stopping application controller")
	c.cancel()
}

// setupUICallbacks connects UI callbacks to controller methods
func (c *Controller) setupUICallbacks() {
	c.mainWindow.SetOnUploadFile(c.handleUploadFile)
	c.mainWindow.SetOnShareFile(c.handleShareFile)
	c.mainWindow.SetOnDeleteFile(c.handleDeleteFile)
	c.mainWindow.SetOnRefreshFiles(c.handleRefreshFiles)
	c.mainWindow.SetOnGeneratePresignedURL(c.GeneratePresignedURL)
}

// handleUploadFile handles file upload requests from UI
func (c *Controller) handleUploadFile(filePath string, expiration time.Duration) error {
	c.logger.Info(fmt.Sprintf("Starting file upload: %s", filePath))
	
	// Update UI to show upload in progress
	c.mainWindow.SetStatus("Uploading file...")
	c.mainWindow.EnableActions(false)
	
	// Create progress channel for upload progress updates
	progressCh := make(chan aws.UploadProgress, 10)
	
	// Start upload in background goroutine
	go func() {
		defer close(progressCh)
		defer func() {
			// Re-enable UI actions when upload completes
			c.mainWindow.EnableActions(true)
		}()
		
		// Perform upload
		fileMetadata, err := c.fileManager.UploadFile(c.ctx, filePath, expiration, progressCh)
		if err != nil {
			c.logger.Error(fmt.Sprintf("File upload failed: %v", err))
			c.mainWindow.SetStatus("Upload failed: " + err.Error())
			return
		}
		
		c.logger.Info(fmt.Sprintf("File upload completed: %s", fileMetadata.ID))
		c.mainWindow.SetStatus("Upload completed successfully")
		
		// Refresh file list to show new file
		if err := c.refreshFiles(); err != nil {
			c.logger.Error(fmt.Sprintf("Failed to refresh files after upload: %v", err))
		}
	}()
	
	// Handle progress updates in separate goroutine
	go func() {
		for progress := range progressCh {
			// Update UI with progress (this would be connected to progress bars in dialogs)
			c.logger.Info(fmt.Sprintf("Upload progress: %.1f%% (%d/%d bytes)", 
				progress.Percentage, progress.BytesUploaded, progress.TotalBytes))
		}
	}()
	
	return nil
}

// handleShareFile handles file sharing requests from UI
func (c *Controller) handleShareFile(fileID string, recipients []string, message string) error {
	c.logger.Info(fmt.Sprintf("Starting file share: %s with %d recipients", fileID, len(recipients)))
	
	// Update UI to show sharing in progress
	c.mainWindow.SetStatus("Sharing file...")
	
	// Perform sharing in background goroutine
	go func() {
		shareRecord, err := c.shareManager.ShareFile(c.ctx, fileID, recipients, message)
		if err != nil {
			c.logger.Error(fmt.Sprintf("File sharing failed: %v", err))
			c.mainWindow.SetStatus("Sharing failed: " + err.Error())
			return
		}
		
		c.logger.Info(fmt.Sprintf("File shared successfully: %s", shareRecord.ID))
		c.mainWindow.SetStatus("File shared successfully")
		
		// Refresh file list to update sharing status
		if err := c.refreshFiles(); err != nil {
			c.logger.Error(fmt.Sprintf("Failed to refresh files after sharing: %v", err))
		}
	}()
	
	return nil
}

// handleDeleteFile handles file deletion requests from UI
func (c *Controller) handleDeleteFile(fileID string) error {
	c.logger.Info(fmt.Sprintf("Starting file deletion: %s", fileID))
	
	// Update UI to show deletion in progress
	c.mainWindow.SetStatus("Deleting file...")
	c.mainWindow.EnableActions(false)
	
	// Perform deletion in background goroutine
	go func() {
		defer func() {
			// Re-enable UI actions when deletion completes
			c.mainWindow.EnableActions(true)
		}()
		
		err := c.fileManager.DeleteFile(fileID)
		if err != nil {
			c.logger.Error(fmt.Sprintf("File deletion failed: %v", err))
			c.mainWindow.SetStatus("Deletion failed: " + err.Error())
			return
		}
		
		c.logger.Info(fmt.Sprintf("File deleted successfully: %s", fileID))
		c.mainWindow.SetStatus("File deleted successfully")
		
		// Refresh file list to remove deleted file
		if err := c.refreshFiles(); err != nil {
			c.logger.Error(fmt.Sprintf("Failed to refresh files after deletion: %v", err))
		}
	}()
	
	return nil
}

// handleRefreshFiles handles file list refresh requests from UI
func (c *Controller) handleRefreshFiles() ([]models.FileMetadata, error) {
	err := c.refreshFiles()
	if err != nil {
		return nil, err
	}
	
	// Return the current file list
	files, err := c.fileManager.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	
	// Convert []*models.FileMetadata to []models.FileMetadata for UI
	fileList := make([]models.FileMetadata, len(files))
	for i, file := range files {
		fileList[i] = *file
	}
	
	return fileList, nil
}

// refreshFiles loads the current file list and updates the UI
func (c *Controller) refreshFiles() error {
	files, err := c.fileManager.ListFiles()
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}
	
	// Convert []*models.FileMetadata to []models.FileMetadata for UI
	fileList := make([]models.FileMetadata, len(files))
	for i, file := range files {
		fileList[i] = *file
	}
	
	// Update UI with file list
	c.mainWindow.UpdateFiles(fileList)
	
	c.logger.Info(fmt.Sprintf("Refreshed file list: %d files", len(files)))
	return nil
}

// startExpirationChecker runs a background process to check for expired files
func (c *Controller) startExpirationChecker() {
	c.logger.Info("Starting background expiration checker")
	
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Stopping expiration checker")
			return
		case <-ticker.C:
			c.checkAndCleanupExpiredFiles()
		}
	}
}

// checkAndCleanupExpiredFiles checks for expired files and updates their status
func (c *Controller) checkAndCleanupExpiredFiles() {
	c.logger.Info("Checking for expired files")
	
	err := c.expirationManager.CleanupExpiredFiles()
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to cleanup expired files: %v", err))
		return
	}
	
	// Refresh file list to update UI with any status changes
	if err := c.refreshFiles(); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to refresh files after expiration cleanup: %v", err))
	}
}

// GeneratePresignedURL generates a presigned URL for a file (used by UI for copy link functionality)
func (c *Controller) GeneratePresignedURL(fileID string, expiration time.Duration) (string, error) {
	c.logger.Info(fmt.Sprintf("Generating presigned URL for file: %s", fileID))
	
	url, err := c.fileManager.GeneratePresignedURL(c.ctx, fileID, expiration)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to generate presigned URL: %v", err))
		return "", err
	}
	
	c.logger.Info(fmt.Sprintf("Generated presigned URL for file: %s", fileID))
	return url, nil
}

// GetShareHistory retrieves sharing history for a file
func (c *Controller) GetShareHistory(fileID string) ([]*storage.ShareRecord, error) {
	return c.shareManager.GetShareHistory(fileID)
}