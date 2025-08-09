package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/models"
	"file-sharing-app/internal/storage"
)

// FileManager interface defines the contract for file metadata management
type FileManager interface {
	// SaveFile saves file metadata to local storage
	SaveFile(file *models.FileMetadata) error
	
	// GetFile retrieves file metadata by ID
	GetFile(fileID string) (*models.FileMetadata, error)
	
	// ListFiles retrieves all file metadata records
	ListFiles() ([]*models.FileMetadata, error)
	
	// UpdateFileStatus updates the status of a file
	UpdateFileStatus(fileID string, status models.FileStatus) error
	
	// DeleteFile removes file metadata from local storage
	DeleteFile(fileID string) error
	
	// CreateFileRecord creates a new file metadata record with generated ID
	CreateFileRecord(fileName, filePath string, fileSize int64, s3Key string, expirationDate time.Time) (*models.FileMetadata, error)
	
	// GetFilesByStatus retrieves files filtered by status
	GetFilesByStatus(status models.FileStatus) ([]*models.FileMetadata, error)
	
	// GetExpiredFiles retrieves files that have passed their expiration date
	GetExpiredFiles() ([]*models.FileMetadata, error)
	
	// UploadFile uploads a file to S3 and stores metadata locally
	UploadFile(ctx context.Context, filePath string, expiration time.Duration, progressCh chan<- aws.UploadProgress) (*models.FileMetadata, error)
}

// FileManagerImpl implements the FileManager interface
type FileManagerImpl struct {
	db        storage.Database
	s3Service aws.S3Service
}

// NewFileManager creates a new FileManager instance
func NewFileManager(db storage.Database, s3Service aws.S3Service) FileManager {
	return &FileManagerImpl{
		db:        db,
		s3Service: s3Service,
	}
}

// NewFileManagerWithoutS3 creates a new FileManager instance without S3 service (for testing)
func NewFileManagerWithoutS3(db storage.Database) FileManager {
	return &FileManagerImpl{
		db: db,
	}
}

// SaveFile saves file metadata to local storage
func (fm *FileManagerImpl) SaveFile(file *models.FileMetadata) error {
	if file == nil {
		return fmt.Errorf("file metadata cannot be nil")
	}
	
	if file.ID == "" {
		return fmt.Errorf("file ID cannot be empty")
	}
	
	if file.FileName == "" {
		return fmt.Errorf("file name cannot be empty")
	}
	
	if file.S3Key == "" {
		return fmt.Errorf("S3 key cannot be empty")
	}
	
	// Convert models.FileMetadata to storage.FileMetadata
	storageFile := &storage.FileMetadata{
		ID:             file.ID,
		FileName:       file.FileName,
		FilePath:       file.FilePath,
		FileSize:       file.FileSize,
		UploadDate:     file.UploadDate,
		ExpirationDate: file.ExpirationDate,
		S3Key:          file.S3Key,
		Status:         storage.FileStatus(file.Status),
	}
	
	return fm.db.SaveFile(storageFile)
}

// GetFile retrieves file metadata by ID
func (fm *FileManagerImpl) GetFile(fileID string) (*models.FileMetadata, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file ID cannot be empty")
	}
	
	storageFile, err := fm.db.GetFile(fileID)
	if err != nil {
		return nil, err
	}
	
	// Convert storage.FileMetadata to models.FileMetadata
	return &models.FileMetadata{
		ID:             storageFile.ID,
		FileName:       storageFile.FileName,
		FilePath:       storageFile.FilePath,
		FileSize:       storageFile.FileSize,
		UploadDate:     storageFile.UploadDate,
		ExpirationDate: storageFile.ExpirationDate,
		S3Key:          storageFile.S3Key,
		Status:         models.FileStatus(storageFile.Status),
	}, nil
}

// ListFiles retrieves all file metadata records
func (fm *FileManagerImpl) ListFiles() ([]*models.FileMetadata, error) {
	storageFiles, err := fm.db.ListFiles()
	if err != nil {
		return nil, err
	}
	
	files := make([]*models.FileMetadata, len(storageFiles))
	for i, storageFile := range storageFiles {
		files[i] = &models.FileMetadata{
			ID:             storageFile.ID,
			FileName:       storageFile.FileName,
			FilePath:       storageFile.FilePath,
			FileSize:       storageFile.FileSize,
			UploadDate:     storageFile.UploadDate,
			ExpirationDate: storageFile.ExpirationDate,
			S3Key:          storageFile.S3Key,
			Status:         models.FileStatus(storageFile.Status),
		}
	}
	
	return files, nil
}

// UpdateFileStatus updates the status of a file
func (fm *FileManagerImpl) UpdateFileStatus(fileID string, status models.FileStatus) error {
	if fileID == "" {
		return fmt.Errorf("file ID cannot be empty")
	}
	
	return fm.db.UpdateFileStatus(fileID, storage.FileStatus(status))
}

// DeleteFile removes file metadata from local storage
func (fm *FileManagerImpl) DeleteFile(fileID string) error {
	if fileID == "" {
		return fmt.Errorf("file ID cannot be empty")
	}
	
	return fm.db.DeleteFile(fileID)
}

// CreateFileRecord creates a new file metadata record with generated ID
func (fm *FileManagerImpl) CreateFileRecord(fileName, filePath string, fileSize int64, s3Key string, expirationDate time.Time) (*models.FileMetadata, error) {
	if fileName == "" {
		return nil, fmt.Errorf("file name cannot be empty")
	}
	
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}
	
	if fileSize <= 0 {
		return nil, fmt.Errorf("file size must be greater than 0")
	}
	
	if s3Key == "" {
		return nil, fmt.Errorf("S3 key cannot be empty")
	}
	
	if expirationDate.IsZero() {
		return nil, fmt.Errorf("expiration date cannot be zero")
	}
	
	file := &models.FileMetadata{
		ID:             uuid.New().String(),
		FileName:       fileName,
		FilePath:       filePath,
		FileSize:       fileSize,
		UploadDate:     time.Now(),
		ExpirationDate: expirationDate,
		S3Key:          s3Key,
		Status:         models.StatusUploading,
	}
	
	err := fm.SaveFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}
	
	return file, nil
}

// GetFilesByStatus retrieves files filtered by status
func (fm *FileManagerImpl) GetFilesByStatus(status models.FileStatus) ([]*models.FileMetadata, error) {
	allFiles, err := fm.ListFiles()
	if err != nil {
		return nil, err
	}
	
	var filteredFiles []*models.FileMetadata
	for _, file := range allFiles {
		if file.Status == status {
			filteredFiles = append(filteredFiles, file)
		}
	}
	
	return filteredFiles, nil
}

// GetExpiredFiles retrieves files that have passed their expiration date
func (fm *FileManagerImpl) GetExpiredFiles() ([]*models.FileMetadata, error) {
	allFiles, err := fm.ListFiles()
	if err != nil {
		return nil, err
	}
	
	now := time.Now()
	var expiredFiles []*models.FileMetadata
	
	for _, file := range allFiles {
		if file.ExpirationDate.Before(now) && file.Status != models.StatusExpired && file.Status != models.StatusDeleted {
			expiredFiles = append(expiredFiles, file)
		}
	}
	
	return expiredFiles, nil
}

// UploadFile uploads a file to S3 and stores metadata locally
func (fm *FileManagerImpl) UploadFile(ctx context.Context, filePath string, expiration time.Duration, progressCh chan<- aws.UploadProgress) (*models.FileMetadata, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}
	
	if fm.s3Service == nil {
		return nil, fmt.Errorf("S3 service not configured")
	}
	
	// Check if file exists and get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	if fileInfo.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}
	
	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return nil, fmt.Errorf("file is empty")
	}
	
	// Check file size limit (100MB as per requirement 2.5)
	const maxFileSize = 100 * 1024 * 1024 // 100MB
	if fileSize > maxFileSize {
		return nil, fmt.Errorf("file size (%d bytes) exceeds maximum allowed size (%d bytes)", fileSize, maxFileSize)
	}
	
	fileName := filepath.Base(filePath)
	
	// Generate UUID-based S3 key with timestamp prefix
	s3Key := generateS3Key(fileName)
	
	// Calculate expiration date
	expirationDate := time.Now().Add(expiration)
	
	// Create file record in database with uploading status
	fileRecord, err := fm.CreateFileRecord(fileName, filePath, fileSize, s3Key, expirationDate)
	if err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}
	
	// Prepare metadata for S3
	metadata := map[string]string{
		"file-id":         fileRecord.ID,
		"original-name":   fileName,
		"expiration-date": expirationDate.UTC().Format(time.RFC3339),
	}
	
	// Upload file to S3
	err = fm.s3Service.UploadFile(ctx, s3Key, filePath, metadata, progressCh)
	if err != nil {
		// Update file status to error
		updateErr := fm.UpdateFileStatus(fileRecord.ID, models.StatusError)
		if updateErr != nil {
			return nil, fmt.Errorf("upload failed: %w, and failed to update status: %w", err, updateErr)
		}
		return nil, fmt.Errorf("failed to upload file to S3: %w", err)
	}
	
	// Update file status to active after successful upload
	err = fm.UpdateFileStatus(fileRecord.ID, models.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("file uploaded successfully but failed to update status: %w", err)
	}
	
	// Return updated file record
	updatedFile, err := fm.GetFile(fileRecord.ID)
	if err != nil {
		return nil, fmt.Errorf("file uploaded successfully but failed to retrieve updated record: %w", err)
	}
	
	return updatedFile, nil
}

// generateS3Key generates a UUID-based S3 key with timestamp prefix
func generateS3Key(fileName string) string {
	// Create timestamp prefix (YYYY/MM/DD format for organization)
	now := time.Now().UTC()
	timestampPrefix := now.Format("2006/01/02")
	
	// Generate UUID for uniqueness
	fileUUID := uuid.New().String()
	
	// Get file extension
	ext := filepath.Ext(fileName)
	
	// Create S3 key: uploads/YYYY/MM/DD/UUID.ext
	s3Key := fmt.Sprintf("uploads/%s/%s%s", timestampPrefix, fileUUID, ext)
	
	return s3Key
}