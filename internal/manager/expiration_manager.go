package manager

import (
	"fmt"
	"time"

	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
	"file-sharing-app/pkg/logger"
)

// ExpirationManager interface defines the contract for handling file lifecycle and expiration
type ExpirationManager interface {
	// SetExpiration sets the expiration date for a file
	SetExpiration(fileID string, duration time.Duration) error
	
	// CheckExpirations checks for files that have expired and returns them
	CheckExpirations() ([]*models.FileMetadata, error)
	
	// CleanupExpiredFiles updates the status of expired files to expired
	CleanupExpiredFiles() error
	
	// GetExpiredFiles retrieves files that have passed their expiration date but are not marked as expired
	GetExpiredFiles() ([]*models.FileMetadata, error)
	
	// IsFileExpired checks if a specific file has expired
	IsFileExpired(fileID string) (bool, error)
	
	// GetTimeUntilExpiration returns the duration until a file expires
	GetTimeUntilExpiration(fileID string) (time.Duration, error)
}

// ExpirationManagerImpl implements the ExpirationManager interface
type ExpirationManagerImpl struct {
	db     storage.Database
	logger *logger.Logger
}

// NewExpirationManager creates a new ExpirationManager instance
func NewExpirationManager(db storage.Database) ExpirationManager {
	return &ExpirationManagerImpl{
		db:     db,
		logger: logger.New(),
	}
}

// SetExpiration sets the expiration date for a file
func (em *ExpirationManagerImpl) SetExpiration(fileID string, duration time.Duration) error {
	if fileID == "" {
		return fmt.Errorf("file ID cannot be empty")
	}
	
	if duration <= 0 {
		return fmt.Errorf("expiration duration must be positive")
	}
	
	// Check if file exists
	_, err := em.db.GetFile(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}
	
	// Calculate new expiration date
	expirationDate := time.Now().Add(duration)
	
	// Update file expiration in database
	err = em.db.UpdateFileExpiration(fileID, expirationDate)
	if err != nil {
		return fmt.Errorf("failed to update file expiration: %w", err)
	}
	
	em.logger.Info(fmt.Sprintf("Updated expiration for file %s to %s", fileID, expirationDate.Format(time.RFC3339)))
	
	return nil
}

// CheckExpirations checks for files that have expired and returns them
func (em *ExpirationManagerImpl) CheckExpirations() ([]*models.FileMetadata, error) {
	// Get all files from database
	storageFiles, err := em.db.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	
	now := time.Now()
	var expiredFiles []*models.FileMetadata
	
	// Check each file for expiration
	for _, storageFile := range storageFiles {
		// Only check files that are not already marked as expired or deleted
		if storageFile.Status != storage.StatusExpired && storageFile.Status != storage.StatusDeleted {
			if storageFile.ExpirationDate.Before(now) {
				// Convert storage.FileMetadata to models.FileMetadata
				expiredFile := &models.FileMetadata{
					ID:             storageFile.ID,
					FileName:       storageFile.FileName,
					FilePath:       storageFile.FilePath,
					FileSize:       storageFile.FileSize,
					UploadDate:     storageFile.UploadDate,
					ExpirationDate: storageFile.ExpirationDate,
					S3Key:          storageFile.S3Key,
					Status:         models.FileStatus(storageFile.Status),
				}
				expiredFiles = append(expiredFiles, expiredFile)
			}
		}
	}
	
	em.logger.Info(fmt.Sprintf("Found %d expired files", len(expiredFiles)))
	
	return expiredFiles, nil
}

// CleanupExpiredFiles updates the status of expired files to expired
func (em *ExpirationManagerImpl) CleanupExpiredFiles() error {
	// Get expired files
	expiredFiles, err := em.CheckExpirations()
	if err != nil {
		return fmt.Errorf("failed to check expirations: %w", err)
	}
	
	if len(expiredFiles) == 0 {
		em.logger.Info("No expired files to cleanup")
		return nil
	}
	
	// Update status for each expired file
	var cleanupErrors []string
	cleanedCount := 0
	
	for _, file := range expiredFiles {
		err := em.db.UpdateFileStatus(file.ID, storage.StatusExpired)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to update status for file %s: %v", file.ID, err))
			em.logger.Error(fmt.Sprintf("Failed to update status for expired file %s: %v", file.ID, err))
		} else {
			cleanedCount++
			em.logger.Info(fmt.Sprintf("Updated status to expired for file %s (%s)", file.ID, file.FileName))
		}
	}
	
	em.logger.Info(fmt.Sprintf("Cleaned up %d expired files", cleanedCount))
	
	// Return error if any cleanup operations failed
	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup completed with errors: %v", cleanupErrors)
	}
	
	return nil
}

// GetExpiredFiles retrieves files that have passed their expiration date but are not marked as expired
func (em *ExpirationManagerImpl) GetExpiredFiles() ([]*models.FileMetadata, error) {
	return em.CheckExpirations()
}

// IsFileExpired checks if a specific file has expired
func (em *ExpirationManagerImpl) IsFileExpired(fileID string) (bool, error) {
	if fileID == "" {
		return false, fmt.Errorf("file ID cannot be empty")
	}
	
	// Get file from database
	storageFile, err := em.db.GetFile(fileID)
	if err != nil {
		return false, fmt.Errorf("failed to get file: %w", err)
	}
	
	// Check if file has expired
	now := time.Now()
	isExpired := storageFile.ExpirationDate.Before(now)
	
	return isExpired, nil
}

// GetTimeUntilExpiration returns the duration until a file expires
func (em *ExpirationManagerImpl) GetTimeUntilExpiration(fileID string) (time.Duration, error) {
	if fileID == "" {
		return 0, fmt.Errorf("file ID cannot be empty")
	}
	
	// Get file from database
	storageFile, err := em.db.GetFile(fileID)
	if err != nil {
		return 0, fmt.Errorf("failed to get file: %w", err)
	}
	
	// Calculate time until expiration
	now := time.Now()
	timeUntilExpiration := storageFile.ExpirationDate.Sub(now)
	
	// If already expired, return 0
	if timeUntilExpiration < 0 {
		return 0, nil
	}
	
	return timeUntilExpiration, nil
}