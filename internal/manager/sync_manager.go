package manager

import (
	"context"
	"fmt"
	"time"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
	"file-sharing-app/pkg/logger"
)

// SyncManager interface defines the contract for data synchronization
type SyncManager interface {
	// SyncWithS3 synchronizes local file metadata with S3 state
	SyncWithS3(ctx context.Context) (*SyncResult, error)
	
	// VerifyFileExists checks if a specific file exists in S3
	VerifyFileExists(ctx context.Context, fileID string) (*FileVerificationResult, error)
	
	// IsOfflineMode returns true if the application is in offline mode
	IsOfflineMode() bool
	
	// SetOfflineMode sets the offline mode state
	SetOfflineMode(offline bool)
	
	// GetLastSyncTime returns the timestamp of the last successful sync
	GetLastSyncTime() (time.Time, error)
}

// SyncResult contains the results of a synchronization operation
type SyncResult struct {
	TotalFiles      int                        `json:"total_files"`
	VerifiedFiles   int                        `json:"verified_files"`
	MissingFiles    int                        `json:"missing_files"`
	ErrorFiles      int                        `json:"error_files"`
	UpdatedFiles    []string                   `json:"updated_files"`
	MissingFileIDs  []string                   `json:"missing_file_ids"`
	Errors          []FileVerificationError    `json:"errors"`
	SyncDuration    time.Duration              `json:"sync_duration"`
	OfflineMode     bool                       `json:"offline_mode"`
}

// FileVerificationResult contains the result of verifying a single file
type FileVerificationResult struct {
	FileID      string                   `json:"file_id"`
	Exists      bool                     `json:"exists"`
	OldStatus   models.FileStatus        `json:"old_status"`
	NewStatus   models.FileStatus        `json:"new_status"`
	Error       *FileVerificationError   `json:"error,omitempty"`
}

// FileVerificationError represents an error during file verification
type FileVerificationError struct {
	FileID  string `json:"file_id"`
	Message string `json:"message"`
	Type    string `json:"type"` // "network", "access_denied", "not_found", "unknown"
}

// SyncManagerImpl implements the SyncManager interface
type SyncManagerImpl struct {
	db          storage.Database
	s3Service   aws.S3Service
	logger      *logger.Logger
	offlineMode bool
}

// NewSyncManager creates a new SyncManager instance
func NewSyncManager(db storage.Database, s3Service aws.S3Service) SyncManager {
	return &SyncManagerImpl{
		db:          db,
		s3Service:   s3Service,
		logger:      logger.New(),
		offlineMode: false,
	}
}

// NewSyncManagerWithoutS3 creates a new SyncManager instance without S3 service (for testing or offline mode)
func NewSyncManagerWithoutS3(db storage.Database) SyncManager {
	return &SyncManagerImpl{
		db:          db,
		s3Service:   nil,
		logger:      logger.New(),
		offlineMode: true,
	}
}

// SyncWithS3 synchronizes local file metadata with S3 state
func (sm *SyncManagerImpl) SyncWithS3(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()
	
	result := &SyncResult{
		UpdatedFiles:   []string{},
		MissingFileIDs: []string{},
		Errors:         []FileVerificationError{},
		OfflineMode:    sm.offlineMode,
	}
	
	sm.logger.Info("Starting synchronization with S3")
	
	// If S3 service is not available, enter offline mode
	if sm.s3Service == nil {
		sm.logger.Info("S3 service not available, entering offline mode")
		sm.offlineMode = true
		result.OfflineMode = true
		result.SyncDuration = time.Since(startTime)
		return result, nil
	}
	
	// Test S3 connection first
	if err := sm.testS3Connection(ctx); err != nil {
		sm.logger.Error(fmt.Sprintf("S3 connection test failed: %v", err))
		sm.offlineMode = true
		result.OfflineMode = true
		result.SyncDuration = time.Since(startTime)
		return result, fmt.Errorf("S3 connection failed, entering offline mode: %w", err)
	}
	
	// S3 is available, exit offline mode
	sm.offlineMode = false
	result.OfflineMode = false
	
	// Get all files from local database
	files, err := sm.getFilesFromDatabase()
	if err != nil {
		result.SyncDuration = time.Since(startTime)
		return result, fmt.Errorf("failed to get files from database: %w", err)
	}
	
	result.TotalFiles = len(files)
	sm.logger.Info(fmt.Sprintf("Found %d files in local database", result.TotalFiles))
	
	// Verify each file in S3
	for _, file := range files {
		// Skip files that are already marked as deleted or error
		if file.Status == storage.StatusDeleted || file.Status == storage.StatusError {
			continue
		}
		
		verificationResult, err := sm.verifyFileInS3(ctx, file)
		if err != nil {
			result.ErrorFiles++
			result.Errors = append(result.Errors, FileVerificationError{
				FileID:  file.ID,
				Message: err.Error(),
				Type:    "unknown",
			})
			continue
		}
		
		if verificationResult.Exists {
			result.VerifiedFiles++
		} else {
			result.MissingFiles++
			result.MissingFileIDs = append(result.MissingFileIDs, file.ID)
		}
		
		// Update file status if it changed
		if verificationResult.NewStatus != verificationResult.OldStatus {
			err := sm.updateFileStatus(file.ID, verificationResult.NewStatus)
			if err != nil {
				sm.logger.Error(fmt.Sprintf("Failed to update status for file %s: %v", file.ID, err))
				result.ErrorFiles++
				result.Errors = append(result.Errors, FileVerificationError{
					FileID:  file.ID,
					Message: fmt.Sprintf("failed to update status: %v", err),
					Type:    "database",
				})
			} else {
				result.UpdatedFiles = append(result.UpdatedFiles, file.ID)
				sm.logger.Info(fmt.Sprintf("Updated file %s status from %s to %s", 
					file.ID, verificationResult.OldStatus, verificationResult.NewStatus))
			}
		}
	}
	
	// Save last sync time
	if err := sm.saveLastSyncTime(time.Now()); err != nil {
		sm.logger.Error(fmt.Sprintf("Failed to save last sync time: %v", err))
	}
	
	result.SyncDuration = time.Since(startTime)
	
	sm.logger.Info(fmt.Sprintf("Synchronization completed: %d total, %d verified, %d missing, %d errors in %v", 
		result.TotalFiles, result.VerifiedFiles, result.MissingFiles, result.ErrorFiles, result.SyncDuration))
	
	return result, nil
}

// VerifyFileExists checks if a specific file exists in S3
func (sm *SyncManagerImpl) VerifyFileExists(ctx context.Context, fileID string) (*FileVerificationResult, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file ID cannot be empty")
	}
	
	// Get file metadata from database
	file, err := sm.getFileFromDatabase(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from database: %w", err)
	}
	
	// If in offline mode, return current status without verification
	if sm.offlineMode || sm.s3Service == nil {
		return &FileVerificationResult{
			FileID:    fileID,
			Exists:    file.Status == storage.StatusActive, // Assume active files exist in offline mode
			OldStatus: models.FileStatus(file.Status),
			NewStatus: models.FileStatus(file.Status),
		}, nil
	}
	
	return sm.verifyFileInS3(ctx, file)
}

// IsOfflineMode returns true if the application is in offline mode
func (sm *SyncManagerImpl) IsOfflineMode() bool {
	return sm.offlineMode
}

// SetOfflineMode sets the offline mode state
func (sm *SyncManagerImpl) SetOfflineMode(offline bool) {
	sm.offlineMode = offline
	if offline {
		sm.logger.Info("Entered offline mode")
	} else {
		sm.logger.Info("Exited offline mode")
	}
}

// GetLastSyncTime returns the timestamp of the last successful sync
func (sm *SyncManagerImpl) GetLastSyncTime() (time.Time, error) {
	value, err := sm.db.GetConfig("last_sync_time")
	if err != nil {
		return time.Time{}, err
	}
	
	return time.Parse(time.RFC3339, value)
}

// testS3Connection tests the S3 connection
func (sm *SyncManagerImpl) testS3Connection(ctx context.Context) error {
	if sm.s3Service == nil {
		return fmt.Errorf("S3 service not available")
	}
	
	// Use a timeout for connection test
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	return sm.s3Service.TestConnection(testCtx)
}

// getFilesFromDatabase retrieves all files from the local database
func (sm *SyncManagerImpl) getFilesFromDatabase() ([]*storage.FileMetadata, error) {
	return sm.db.ListFiles()
}

// getFileFromDatabase retrieves a specific file from the local database
func (sm *SyncManagerImpl) getFileFromDatabase(fileID string) (*storage.FileMetadata, error) {
	return sm.db.GetFile(fileID)
}

// verifyFileInS3 checks if a file exists in S3 and determines its correct status
func (sm *SyncManagerImpl) verifyFileInS3(ctx context.Context, file *storage.FileMetadata) (*FileVerificationResult, error) {
	result := &FileVerificationResult{
		FileID:    file.ID,
		OldStatus: models.FileStatus(file.Status),
		NewStatus: models.FileStatus(file.Status),
	}
	
	// Use a timeout for S3 operations
	verifyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Try to get object metadata from S3
	_, err := sm.s3Service.HeadObject(verifyCtx, file.S3Key)
	if err != nil {
		// Check if it's a "not found" error
		if isNotFoundError(err) {
			result.Exists = false
			// If file was active but doesn't exist in S3, mark as deleted
			if file.Status == storage.StatusActive {
				result.NewStatus = models.StatusDeleted
			}
		} else {
			// Other errors (network, access denied, etc.)
			result.Error = &FileVerificationError{
				FileID:  file.ID,
				Message: err.Error(),
				Type:    categorizeError(err),
			}
			return result, fmt.Errorf("failed to verify file in S3: %w", err)
		}
	} else {
		result.Exists = true
		
		// Check if file has expired based on local expiration date
		if time.Now().After(file.ExpirationDate) && file.Status != storage.StatusExpired {
			result.NewStatus = models.StatusExpired
		} else if file.Status == storage.StatusUploading {
			// If file exists in S3 but local status is still uploading, mark as active
			result.NewStatus = models.StatusActive
		}
	}
	
	return result, nil
}

// updateFileStatus updates the status of a file in the database
func (sm *SyncManagerImpl) updateFileStatus(fileID string, status models.FileStatus) error {
	return sm.db.UpdateFileStatus(fileID, storage.FileStatus(status))
}

// saveLastSyncTime saves the timestamp of the last successful sync
func (sm *SyncManagerImpl) saveLastSyncTime(syncTime time.Time) error {
	return sm.db.SaveConfig("last_sync_time", syncTime.Format(time.RFC3339))
}

// isNotFoundError checks if an error indicates that the object was not found
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	return contains(errStr, "NoSuchKey") || 
		   contains(errStr, "not found") || 
		   contains(errStr, "404")
}

// categorizeError categorizes an error for better handling
func categorizeError(err error) string {
	if err == nil {
		return "unknown"
	}
	
	errStr := err.Error()
	
	if contains(errStr, "AccessDenied") || contains(errStr, "403") {
		return "access_denied"
	}
	if contains(errStr, "NoSuchKey") || contains(errStr, "404") {
		return "not_found"
	}
	if contains(errStr, "timeout") || contains(errStr, "connection") {
		return "network"
	}
	
	return "unknown"
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      containsSubstring(s, substr))))
}

// containsSubstring performs a simple substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}