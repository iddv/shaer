package storage

import (
	"file-sharing-app/internal/models"
)

// Database interface defines the contract for data storage operations
type Database interface {
	// File operations
	SaveFile(file *models.FileMetadata) error
	GetFile(id string) (*models.FileMetadata, error)
	ListFiles() ([]*models.FileMetadata, error)
	UpdateFileStatus(id string, status models.FileStatus) error
	DeleteFile(id string) error

	// Share operations
	SaveShare(share *models.ShareRecord) error
	GetShareHistory(fileID string) ([]*models.ShareRecord, error)

	// Configuration operations
	SaveConfig(key, value string) error
	GetConfig(key string) (string, error)

	// Database lifecycle
	Close() error
}

// SQLiteDatabase implements Database interface using SQLite
type SQLiteDatabase struct {
	// Will be implemented in task 2
}

// NewSQLiteDatabase creates a new SQLite database instance
func NewSQLiteDatabase(dbPath string) (*SQLiteDatabase, error) {
	// Will be implemented in task 2
	return &SQLiteDatabase{}, nil
}