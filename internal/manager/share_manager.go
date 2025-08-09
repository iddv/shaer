package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"file-sharing-app/internal/aws"
	"file-sharing-app/internal/storage"
)

// ShareManager defines the interface for managing file shares
type ShareManager interface {
	// ShareFile creates a new share record for a file with recipients and custom message
	ShareFile(ctx context.Context, fileID string, recipients []string, message string) (*storage.ShareRecord, error)
	
	// GetShareHistory retrieves all share records for a file
	GetShareHistory(fileID string) ([]*storage.ShareRecord, error)
	
	// RevokeShare revokes a share by updating its status (not implemented in simplified version)
	RevokeShare(shareID string) error
	
	// GeneratePresignedURL generates a presigned URL for a file with specified expiration
	GeneratePresignedURL(ctx context.Context, fileID string, expiration time.Duration) (string, error)
}

// ShareManagerImpl implements ShareManager interface
type ShareManagerImpl struct {
	db        storage.Database
	s3Service aws.S3Service
}

// NewShareManager creates a new ShareManager instance
func NewShareManager(db storage.Database, s3Service aws.S3Service) *ShareManagerImpl {
	return &ShareManagerImpl{
		db:        db,
		s3Service: s3Service,
	}
}

// ShareFile creates a new share record for a file with recipients and custom message
func (sm *ShareManagerImpl) ShareFile(ctx context.Context, fileID string, recipients []string, message string) (*storage.ShareRecord, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file ID cannot be empty")
	}
	
	if len(recipients) == 0 {
		return nil, fmt.Errorf("at least one recipient must be specified")
	}

	// Validate recipients (basic email format validation)
	for _, recipient := range recipients {
		if err := validateEmail(recipient); err != nil {
			return nil, fmt.Errorf("invalid recipient email '%s': %w", recipient, err)
		}
	}

	// Get file metadata to ensure file exists and get S3 key
	file, err := sm.db.GetFile(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	// Check if file is active (not expired or deleted)
	if file.Status != storage.StatusActive {
		return nil, fmt.Errorf("cannot share file with status: %s", file.Status)
	}

	// Check if file has expired
	if time.Now().After(file.ExpirationDate) {
		return nil, fmt.Errorf("cannot share expired file")
	}

	// Calculate URL expiration (should not exceed file expiration)
	urlExpiration := calculateURLExpiration(file.ExpirationDate)
	
	// Generate presigned URL
	presignedURL, err := sm.s3Service.GeneratePresignedURL(ctx, file.S3Key, time.Until(urlExpiration))
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Create share record
	shareRecord := &storage.ShareRecord{
		ID:            uuid.New().String(),
		FileID:        fileID,
		Recipients:    recipients,
		Message:       message,
		SharedDate:    time.Now(),
		PresignedURL:  presignedURL,
		URLExpiration: urlExpiration,
	}

	// Save share record to database
	if err := sm.db.SaveShare(shareRecord); err != nil {
		return nil, fmt.Errorf("failed to save share record: %w", err)
	}

	return shareRecord, nil
}

// GetShareHistory retrieves all share records for a file
func (sm *ShareManagerImpl) GetShareHistory(fileID string) ([]*storage.ShareRecord, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file ID cannot be empty")
	}

	shares, err := sm.db.GetShareHistory(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get share history: %w", err)
	}

	return shares, nil
}

// RevokeShare revokes a share by updating its status
// Note: In the simplified version, we don't implement actual revocation
// This would require additional database schema changes
func (sm *ShareManagerImpl) RevokeShare(shareID string) error {
	if shareID == "" {
		return fmt.Errorf("share ID cannot be empty")
	}

	// In a full implementation, this would:
	// 1. Update share record status to "revoked"
	// 2. Potentially invalidate the presigned URL (not possible with S3 presigned URLs)
	// For now, return not implemented error
	return fmt.Errorf("share revocation not implemented in simplified version")
}

// GeneratePresignedURL generates a presigned URL for a file with specified expiration
func (sm *ShareManagerImpl) GeneratePresignedURL(ctx context.Context, fileID string, expiration time.Duration) (string, error) {
	if fileID == "" {
		return "", fmt.Errorf("file ID cannot be empty")
	}

	if expiration <= 0 {
		return "", fmt.Errorf("expiration duration must be positive")
	}

	// Get file metadata to get S3 key
	file, err := sm.db.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("failed to get file metadata: %w", err)
	}

	// Check if file is active
	if file.Status != storage.StatusActive {
		return "", fmt.Errorf("cannot generate URL for file with status: %s", file.Status)
	}

	// Check if file has expired
	if time.Now().After(file.ExpirationDate) {
		return "", fmt.Errorf("cannot generate URL for expired file")
	}

	// Ensure expiration doesn't exceed file expiration
	maxExpiration := time.Until(file.ExpirationDate)
	if expiration > maxExpiration {
		expiration = maxExpiration
	}

	// Generate presigned URL
	presignedURL, err := sm.s3Service.GeneratePresignedURL(ctx, file.S3Key, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL, nil
}

// validateEmail performs basic email format validation
func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	// Basic email validation - contains @ and has parts before and after
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			if atIndex != -1 {
				return fmt.Errorf("email contains multiple @ symbols")
			}
			atIndex = i
		}
	}

	if atIndex == -1 {
		return fmt.Errorf("email must contain @ symbol")
	}

	if atIndex == 0 {
		return fmt.Errorf("email must have local part before @")
	}

	if atIndex == len(email)-1 {
		return fmt.Errorf("email must have domain part after @")
	}

	// Check for basic domain format (contains at least one dot after @)
	domainPart := email[atIndex+1:]
	hasDot := false
	for _, char := range domainPart {
		if char == '.' {
			hasDot = true
			break
		}
	}

	if !hasDot {
		return fmt.Errorf("email domain must contain at least one dot")
	}

	return nil
}

// calculateURLExpiration calculates the appropriate URL expiration time
// ensuring it doesn't exceed the file's expiration date
func calculateURLExpiration(fileExpiration time.Time) time.Time {
	now := time.Now()
	
	// Default URL expiration is 24 hours
	defaultExpiration := now.Add(24 * time.Hour)
	
	// If file expires before default expiration, use file expiration
	if fileExpiration.Before(defaultExpiration) {
		return fileExpiration
	}
	
	return defaultExpiration
}