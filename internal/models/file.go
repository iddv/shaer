package models

import "time"

// FileStatus represents the current status of a file
type FileStatus string

const (
	StatusUploading FileStatus = "uploading"
	StatusActive    FileStatus = "active"
	StatusExpired   FileStatus = "expired"
	StatusDeleted   FileStatus = "deleted"
	StatusError     FileStatus = "error"
)

// FileMetadata represents file information stored locally
type FileMetadata struct {
	ID             string    `json:"id"`
	FileName       string    `json:"filename"`
	FilePath       string    `json:"filepath"`
	FileSize       int64     `json:"filesize"`
	UploadDate     time.Time `json:"upload_date"`
	ExpirationDate time.Time `json:"expiration_date"`
	S3Key          string    `json:"s3_key"`
	Status         FileStatus `json:"status"`
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
}