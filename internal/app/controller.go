package app

import (
	"context"
	"fmt"
	"time"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/manager"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
	"file-sharing-app/pkg/errors"
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
	SetOnSaveSettings(callback func(settings *models.ApplicationSettings) error)
	SetOnLoadSettings(callback func() (*models.ApplicationSettings, error))
}

// Controller coordinates between UI and business logic layers
type Controller struct {
	// Business logic managers
	fileManager       manager.FileManager
	shareManager      manager.ShareManager
	expirationManager manager.ExpirationManager
	settingsManager   manager.SettingsManager
	syncManager       manager.SyncManager
	
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
	settingsManager manager.SettingsManager,
	syncManager manager.SyncManager,
	mainWindow MainWindowInterface,
) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	
	controller := &Controller{
		fileManager:       fileManager,
		shareManager:      shareManager,
		expirationManager: expirationManager,
		settingsManager:   settingsManager,
		syncManager:       syncManager,
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
	
	// Perform initial synchronization with S3
	c.mainWindow.SetStatus("Synchronizing with S3...")
	go c.performInitialSync()
	
	// Start background expiration checker
	go c.startExpirationChecker()
	
	// Load initial file list (this will work even if sync fails)
	if err := c.refreshFiles(); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to load initial files: %v", err))
		c.mainWindow.SetStatus("Error loading files: " + err.Error())
		return err
	}
	
	// Set initial status based on offline mode
	if c.syncManager.IsOfflineMode() {
		c.mainWindow.SetStatus("Ready (Offline Mode)")
	} else {
		c.mainWindow.SetStatus("Ready")
	}
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
	c.mainWindow.SetOnSaveSettings(c.handleSaveSettings)
	c.mainWindow.SetOnLoadSettings(c.handleLoadSettings)
}

// handleUploadFile handles file upload requests from UI
func (c *Controller) handleUploadFile(filePath string, expiration time.Duration) error {
	c.logger.Info(fmt.Sprintf("Starting file upload: %s", filePath))
	
	// Check if we're in offline mode
	if c.syncManager.IsOfflineMode() {
		c.logger.Error("Cannot upload files in offline mode")
		c.mainWindow.SetStatus("Upload failed: Application is in offline mode")
		return fmt.Errorf("cannot upload files in offline mode")
	}
	
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
			
			// Check if error is due to network issues and enter offline mode
			if isNetworkError(err) {
				c.logger.Info("Network error detected, entering offline mode")
				c.syncManager.SetOfflineMode(true)
				c.mainWindow.SetStatus("Upload failed: Network error - Entered offline mode")
			} else {
				c.mainWindow.SetStatus("Upload failed: " + err.Error())
			}
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
	
	// Check if we're in offline mode
	if c.syncManager.IsOfflineMode() {
		c.logger.Error("Cannot share files in offline mode")
		c.mainWindow.SetStatus("Sharing failed: Application is in offline mode")
		return fmt.Errorf("cannot share files in offline mode")
	}
	
	// Update UI to show sharing in progress
	c.mainWindow.SetStatus("Sharing file...")
	
	// Perform sharing in background goroutine
	go func() {
		shareRecord, err := c.shareManager.ShareFile(c.ctx, fileID, recipients, message)
		if err != nil {
			c.logger.Error(fmt.Sprintf("File sharing failed: %v", err))
			
			// Check if error is due to network issues and enter offline mode
			if isNetworkError(err) {
				c.logger.Info("Network error detected, entering offline mode")
				c.syncManager.SetOfflineMode(true)
				c.mainWindow.SetStatus("Sharing failed: Network error - Entered offline mode")
			} else {
				c.mainWindow.SetStatus("Sharing failed: " + err.Error())
			}
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
	
	// Check if we're in offline mode
	if c.syncManager.IsOfflineMode() {
		c.logger.Error("Cannot generate presigned URLs in offline mode")
		return "", fmt.Errorf("cannot generate presigned URLs in offline mode")
	}
	
	url, err := c.fileManager.GeneratePresignedURL(c.ctx, fileID, expiration)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to generate presigned URL: %v", err))
		
		// Check if error is due to network issues and enter offline mode
		if isNetworkError(err) {
			c.logger.Info("Network error detected, entering offline mode")
			c.syncManager.SetOfflineMode(true)
			return "", fmt.Errorf("network error - entered offline mode: %w", err)
		}
		
		return "", err
	}
	
	c.logger.Info(fmt.Sprintf("Generated presigned URL for file: %s", fileID))
	return url, nil
}

// GetShareHistory retrieves sharing history for a file
func (c *Controller) GetShareHistory(fileID string) ([]*storage.ShareRecord, error) {
	return c.shareManager.GetShareHistory(fileID)
}

// handleSaveSettings handles settings save requests from UI
func (c *Controller) handleSaveSettings(settings *models.ApplicationSettings) error {
	c.logger.Info("Saving application settings")
	
	err := c.settingsManager.SaveSettings(settings)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to save settings: %v", err))
		return fmt.Errorf("failed to save settings: %w", err)
	}
	
	c.logger.Info("Application settings saved successfully")
	return nil
}

// handleLoadSettings handles settings load requests from UI
func (c *Controller) handleLoadSettings() (*models.ApplicationSettings, error) {
	c.logger.Info("Loading application settings")
	
	settings, err := c.settingsManager.LoadSettings()
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to load settings: %v", err))
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}
	
	c.logger.Info("Application settings loaded successfully")
	return settings, nil
}

// performInitialSync performs initial synchronization with S3 on startup
func (c *Controller) performInitialSync() {
	c.logger.Info("Starting initial synchronization with S3")
	
	// Use a timeout for the initial sync
	syncCtx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	
	result, err := c.syncManager.SyncWithS3(syncCtx)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Initial sync failed: %v", err))
		if result != nil && result.OfflineMode {
			c.mainWindow.SetStatus("Ready (Offline Mode - S3 unavailable)")
		} else {
			c.mainWindow.SetStatus("Ready (Sync failed)")
		}
		return
	}
	
	// Log sync results
	c.logger.Info(fmt.Sprintf("Initial sync completed: %d total, %d verified, %d missing, %d errors in %v", 
		result.TotalFiles, result.VerifiedFiles, result.MissingFiles, result.ErrorFiles, result.SyncDuration))
	
	// Update UI status based on sync results
	if result.OfflineMode {
		c.mainWindow.SetStatus("Ready (Offline Mode)")
	} else if result.ErrorFiles > 0 || result.MissingFiles > 0 {
		c.mainWindow.SetStatus(fmt.Sprintf("Ready (Sync completed with %d issues)", result.ErrorFiles+result.MissingFiles))
	} else {
		c.mainWindow.SetStatus("Ready (Synced)")
	}
	
	// Refresh file list to show any status updates
	if err := c.refreshFiles(); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to refresh files after sync: %v", err))
	}
}

// SyncWithS3 manually triggers synchronization with S3 (can be called from UI)
func (c *Controller) SyncWithS3() (*manager.SyncResult, error) {
	c.logger.Info("Manual sync with S3 requested")
	
	c.mainWindow.SetStatus("Synchronizing with S3...")
	c.mainWindow.EnableActions(false)
	
	defer func() {
		c.mainWindow.EnableActions(true)
	}()
	
	// Use a timeout for manual sync
	syncCtx, cancel := context.WithTimeout(c.ctx, 60*time.Second)
	defer cancel()
	
	result, err := c.syncManager.SyncWithS3(syncCtx)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Manual sync failed: %v", err))
		if result != nil && result.OfflineMode {
			c.mainWindow.SetStatus("Ready (Offline Mode - S3 unavailable)")
		} else {
			c.mainWindow.SetStatus("Sync failed: " + err.Error())
		}
		return result, err
	}
	
	// Log sync results
	c.logger.Info(fmt.Sprintf("Manual sync completed: %d total, %d verified, %d missing, %d errors in %v", 
		result.TotalFiles, result.VerifiedFiles, result.MissingFiles, result.ErrorFiles, result.SyncDuration))
	
	// Update UI status based on sync results
	if result.OfflineMode {
		c.mainWindow.SetStatus("Ready (Offline Mode)")
	} else if result.ErrorFiles > 0 || result.MissingFiles > 0 {
		c.mainWindow.SetStatus(fmt.Sprintf("Sync completed with %d issues", result.ErrorFiles+result.MissingFiles))
	} else {
		c.mainWindow.SetStatus("Sync completed successfully")
	}
	
	// Refresh file list to show any status updates
	if err := c.refreshFiles(); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to refresh files after manual sync: %v", err))
	}
	
	return result, nil
}

// IsOfflineMode returns true if the application is in offline mode
func (c *Controller) IsOfflineMode() bool {
	return c.syncManager.IsOfflineMode()
}

// GetLastSyncTime returns the timestamp of the last successful sync
func (c *Controller) GetLastSyncTime() (time.Time, error) {
	return c.syncManager.GetLastSyncTime()
}

// VerifyFileExists checks if a specific file exists in S3
func (c *Controller) VerifyFileExists(fileID string) (*manager.FileVerificationResult, error) {
	return c.syncManager.VerifyFileExists(c.ctx, fileID)
}

// isNetworkError checks if an error is related to network connectivity
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	networkKeywords := []string{
		"timeout",
		"connection",
		"network",
		"dial",
		"no such host",
		"connection refused",
		"connection reset",
		"context deadline exceeded",
	}
	
	for _, keyword := range networkKeywords {
		if containsIgnoreCase(errStr, keyword) {
			return true
		}
	}
	
	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive substring search
	sLower := toLower(s)
	substrLower := toLower(substr)
	
	return len(sLower) >= len(substrLower) && 
		   (sLower == substrLower || 
		    (len(sLower) > len(substrLower) && 
		     (sLower[:len(substrLower)] == substrLower || 
		      sLower[len(sLower)-len(substrLower):] == substrLower || 
		      containsSubstringIgnoreCase(sLower, substrLower))))
}

// toLower converts a string to lowercase
func toLower(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}

// containsSubstringIgnoreCase performs a simple substring search
func containsSubstringIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Error Recovery Mechanisms

// handleError provides centralized error handling with user-friendly messages and recovery suggestions
func (c *Controller) handleError(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Classify the error using our error system
	appErr := errors.ClassifyError(err)
	
	c.logger.ErrorWithFields("Operation failed", map[string]interface{}{
		"operation":        operation,
		"error_code":       string(appErr.Code),
		"user_message":     appErr.GetUserMessage(),
		"suggested_action": appErr.GetSuggestedAction(),
		"recoverable":      appErr.IsRecoverable(),
	})

	// Update UI with user-friendly status message
	c.mainWindow.SetStatus(fmt.Sprintf("Error: %s", appErr.GetUserMessage()))

	// For recoverable errors, suggest retry or recovery actions
	if appErr.IsRecoverable() {
		c.logger.InfoWithFields("Error is recoverable", map[string]interface{}{
			"operation":        operation,
			"suggested_action": appErr.GetSuggestedAction(),
		})
		
		// For network errors, check if we should switch to offline mode
		if appErr.Code == errors.ErrNetworkError || appErr.Code == errors.ErrConnectionTimeout {
			c.handleNetworkError(operation)
		}
	}

	return appErr
}

// handleNetworkError handles network-related errors with automatic offline mode switching
func (c *Controller) handleNetworkError(operation string) {
	c.logger.WarnWithFields("Network error detected, checking offline mode", map[string]interface{}{
		"operation": operation,
	})

	// Check if we should switch to offline mode
	if !c.syncManager.IsOfflineMode() {
		c.logger.Info("Switching to offline mode due to network issues")
		c.mainWindow.SetStatus("Network unavailable - switched to offline mode")
		
		// Refresh the file list to show only locally available files
		go func() {
			if err := c.refreshFiles(); err != nil {
				c.logger.ErrorWithError("Failed to refresh files in offline mode", err)
			}
		}()
	}
}

// retryOperation attempts to retry a recoverable operation with exponential backoff
func (c *Controller) retryOperation(operation string, fn func() error) error {
	config := errors.DefaultRetryConfig()
	config.BaseDelay = 2 * time.Second  // Slightly longer delay for UI operations
	config.MaxDelay = 30 * time.Second
	
	return c.logger.LogOperation(fmt.Sprintf("retry_%s", operation), func() error {
		return errors.RetryWithBackoff(c.ctx, fn, config)
	})
}

// recoverFromError attempts to recover from specific error conditions
func (c *Controller) recoverFromError(err error) bool {
	appErr := errors.ClassifyError(err)
	
	switch appErr.Code {
	case errors.ErrInvalidCredentials, errors.ErrCredentialsExpired:
		c.logger.Info("Attempting to recover from credential error")
		c.mainWindow.SetStatus("AWS credentials need to be updated - please check settings")
		return true
		
	case errors.ErrS3BucketNotFound:
		c.logger.Info("Attempting to recover from bucket not found error")
		c.mainWindow.SetStatus("S3 bucket not found - please check settings")
		return true
		
	case errors.ErrNetworkError, errors.ErrConnectionTimeout:
		c.logger.Info("Attempting to recover from network error")
		c.handleNetworkError("recovery")
		return true
		
	case errors.ErrDatabaseError, errors.ErrDatabaseConnection:
		c.logger.Info("Database error detected - application may need restart")
		c.mainWindow.SetStatus("Database error - please restart the application")
		return false // Cannot recover from database errors
		
	default:
		return false
	}
}

// validateOperation performs pre-operation validation to prevent common errors
func (c *Controller) validateOperation(operation string) error {
	switch operation {
	case "upload", "share", "delete":
		// Check if we're in offline mode for operations that require network
		if c.syncManager.IsOfflineMode() {
			return errors.NewAppError(
				errors.ErrServiceUnavailable,
				"Operation not available in offline mode",
				nil,
			)
		}
		
		// Check if we have valid settings
		settings, err := c.settingsManager.LoadSettings()
		if err != nil {
			return errors.WrapError(err, errors.ErrConfigurationError, "failed to load application settings")
		}
		
		if err := settings.ValidateForSave(); err != nil {
			return errors.WrapError(err, errors.ErrInvalidConfig, "application settings are invalid")
		}
		
	case "refresh", "list":
		// These operations can work in offline mode, no additional validation needed
		break
		
	default:
		c.logger.WarnWithFields("Unknown operation validation requested", map[string]interface{}{
			"operation": operation,
		})
	}
	
	return nil
}

// handleOperationWithRecovery wraps operations with error handling and recovery
func (c *Controller) handleOperationWithRecovery(operation string, fn func() error) error {
	// Pre-operation validation
	if err := c.validateOperation(operation); err != nil {
		return c.handleError(operation, err)
	}
	
	// Execute the operation with retry logic for recoverable errors
	err := c.retryOperation(operation, fn)
	
	if err != nil {
		// Attempt recovery
		if c.recoverFromError(err) {
			c.logger.InfoWithFields("Error recovery attempted", map[string]interface{}{
				"operation": operation,
			})
		}
		
		// Handle and log the error
		return c.handleError(operation, err)
	}
	
	return nil
}