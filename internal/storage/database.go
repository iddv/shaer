package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FileStatus represents the status of a file
type FileStatus string

const (
	StatusUploading FileStatus = "uploading"
	StatusActive    FileStatus = "active"
	StatusExpired   FileStatus = "expired"
	StatusDeleted   FileStatus = "deleted"
	StatusError     FileStatus = "error"
)

// FileMetadata represents file information stored in the database
type FileMetadata struct {
	ID             string    `json:"id"`
	FileName       string    `json:"filename"`
	FilePath       string    `json:"filepath"`
	FileSize       int64     `json:"filesize"`
	UploadDate     time.Time `json:"upload_date"`
	ExpirationDate time.Time `json:"expiration_date"`
	S3Key          string    `json:"s3_key"`
	Status         FileStatus `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ShareRecord represents a file sharing record
type ShareRecord struct {
	ID            string    `json:"id"`
	FileID        string    `json:"file_id"`
	Recipients    []string  `json:"recipients"`
	Message       string    `json:"message"`
	SharedDate    time.Time `json:"shared_date"`
	PresignedURL  string    `json:"presigned_url"`
	URLExpiration time.Time `json:"url_expiration"`
	CreatedAt     time.Time `json:"created_at"`
}

// Database interface defines the contract for database operations
type Database interface {
	// File operations
	SaveFile(file *FileMetadata) error
	GetFile(id string) (*FileMetadata, error)
	ListFiles() ([]*FileMetadata, error)
	UpdateFileStatus(id string, status FileStatus) error
	DeleteFile(id string) error

	// Share operations
	SaveShare(share *ShareRecord) error
	GetShareHistory(fileID string) ([]*ShareRecord, error)

	// Configuration operations
	SaveConfig(key, value string) error
	GetConfig(key string) (string, error)

	// Database management
	Close() error
}

// SQLiteDatabase implements the Database interface using SQLite
type SQLiteDatabase struct {
	db *sql.DB
}

// NewSQLiteDatabase creates a new SQLite database instance
func NewSQLiteDatabase(dbPath string) (*SQLiteDatabase, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ping to ensure the database file is created
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set proper file permissions (600 - read/write for owner only)
	if err := os.Chmod(dbPath, 0600); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set database file permissions: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	sqliteDB := &SQLiteDatabase{db: db}

	// Initialize schema
	if err := sqliteDB.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return sqliteDB, nil
}

// initSchema creates the database tables if they don't exist
func (s *SQLiteDatabase) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		filepath TEXT NOT NULL,
		filesize INTEGER NOT NULL,
		upload_date DATETIME NOT NULL,
		expiration_date DATETIME NOT NULL,
		s3_key TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_files_upload_date ON files(upload_date);
	CREATE INDEX IF NOT EXISTS idx_files_expiration_date ON files(expiration_date);
	CREATE INDEX IF NOT EXISTS idx_files_status ON files(status);

	CREATE TABLE IF NOT EXISTS shares (
		id TEXT PRIMARY KEY,
		file_id TEXT NOT NULL,
		recipients TEXT NOT NULL,
		message TEXT,
		shared_date DATETIME NOT NULL,
		presigned_url TEXT NOT NULL,
		url_expiration DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_shares_file_id ON shares(file_id);
	CREATE INDEX IF NOT EXISTS idx_shares_shared_date ON shares(shared_date);

	CREATE TABLE IF NOT EXISTS app_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// File operations

// SaveFile saves a file metadata record to the database
func (s *SQLiteDatabase) SaveFile(file *FileMetadata) error {
	now := time.Now()
	file.CreatedAt = now
	file.UpdatedAt = now

	query := `
		INSERT INTO files (id, filename, filepath, filesize, upload_date, expiration_date, s3_key, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		file.ID, file.FileName, file.FilePath, file.FileSize,
		file.UploadDate, file.ExpirationDate, file.S3Key, string(file.Status),
		file.CreatedAt, file.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// GetFile retrieves a file metadata record by ID
func (s *SQLiteDatabase) GetFile(id string) (*FileMetadata, error) {
	query := `
		SELECT id, filename, filepath, filesize, upload_date, expiration_date, s3_key, status, created_at, updated_at
		FROM files WHERE id = ?
	`

	row := s.db.QueryRow(query, id)

	var file FileMetadata
	var status string

	err := row.Scan(
		&file.ID, &file.FileName, &file.FilePath, &file.FileSize,
		&file.UploadDate, &file.ExpirationDate, &file.S3Key, &status,
		&file.CreatedAt, &file.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	file.Status = FileStatus(status)
	return &file, nil
}

// ListFiles retrieves all file metadata records
func (s *SQLiteDatabase) ListFiles() ([]*FileMetadata, error) {
	query := `
		SELECT id, filename, filepath, filesize, upload_date, expiration_date, s3_key, status, created_at, updated_at
		FROM files ORDER BY upload_date DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []*FileMetadata

	for rows.Next() {
		var file FileMetadata
		var status string

		err := rows.Scan(
			&file.ID, &file.FileName, &file.FilePath, &file.FileSize,
			&file.UploadDate, &file.ExpirationDate, &file.S3Key, &status,
			&file.CreatedAt, &file.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan file row: %w", err)
		}

		file.Status = FileStatus(status)
		files = append(files, &file)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating file rows: %w", err)
	}

	return files, nil
}

// UpdateFileStatus updates the status of a file
func (s *SQLiteDatabase) UpdateFileStatus(id string, status FileStatus) error {
	query := `UPDATE files SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

	result, err := s.db.Exec(query, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("file not found: %s", id)
	}

	return nil
}

// DeleteFile removes a file metadata record from the database
func (s *SQLiteDatabase) DeleteFile(id string) error {
	query := `DELETE FROM files WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("file not found: %s", id)
	}

	return nil
}

// Share operations

// SaveShare saves a share record to the database
func (s *SQLiteDatabase) SaveShare(share *ShareRecord) error {
	now := time.Now()
	share.CreatedAt = now

	// Convert recipients slice to JSON
	recipientsJSON, err := json.Marshal(share.Recipients)
	if err != nil {
		return fmt.Errorf("failed to marshal recipients: %w", err)
	}

	query := `
		INSERT INTO shares (id, file_id, recipients, message, shared_date, presigned_url, url_expiration, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		share.ID, share.FileID, string(recipientsJSON), share.Message,
		share.SharedDate, share.PresignedURL, share.URLExpiration, share.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save share: %w", err)
	}

	return nil
}

// GetShareHistory retrieves all share records for a file
func (s *SQLiteDatabase) GetShareHistory(fileID string) ([]*ShareRecord, error) {
	query := `
		SELECT id, file_id, recipients, message, shared_date, presigned_url, url_expiration, created_at
		FROM shares WHERE file_id = ? ORDER BY shared_date DESC
	`

	rows, err := s.db.Query(query, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get share history: %w", err)
	}
	defer rows.Close()

	var shares []*ShareRecord

	for rows.Next() {
		var share ShareRecord
		var recipientsJSON string

		err := rows.Scan(
			&share.ID, &share.FileID, &recipientsJSON, &share.Message,
			&share.SharedDate, &share.PresignedURL, &share.URLExpiration, &share.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan share row: %w", err)
		}

		// Unmarshal recipients JSON
		if err := json.Unmarshal([]byte(recipientsJSON), &share.Recipients); err != nil {
			return nil, fmt.Errorf("failed to unmarshal recipients: %w", err)
		}

		shares = append(shares, &share)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating share rows: %w", err)
	}

	return shares, nil
}

// Configuration operations

// SaveConfig saves a configuration key-value pair
func (s *SQLiteDatabase) SaveConfig(key, value string) error {
	query := `
		INSERT OR REPLACE INTO app_config (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`

	_, err := s.db.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetConfig retrieves a configuration value by key
func (s *SQLiteDatabase) GetConfig(key string) (string, error) {
	query := `SELECT value FROM app_config WHERE key = ?`

	var value string
	err := s.db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("config key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	return value, nil
}

// Close closes the database connection
func (s *SQLiteDatabase) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}