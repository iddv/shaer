package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	Operation string                 `json:"operation,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
}

// Logger provides structured logging capabilities without exposing sensitive information
type Logger struct {
	*log.Logger
	component string
	minLevel  LogLevel
}

// New creates a new logger instance
func New() *Logger {
	return &Logger{
		Logger:    log.New(os.Stdout, "", 0), // No prefix, we'll handle formatting
		component: "app",
		minLevel:  LevelInfo,
	}
}

// NewWithComponent creates a new logger instance with a specific component name
func NewWithComponent(component string) *Logger {
	return &Logger{
		Logger:    log.New(os.Stdout, "", 0),
		component: component,
		minLevel:  LevelInfo,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.minLevel = level
}

// shouldLog determines if a message should be logged based on level
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}
	return levels[level] >= levels[l.minLevel]
}

// log writes a structured log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}, err error, operation string) {
	if !l.shouldLog(level) {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if ok {
		// Extract just the filename, not the full path
		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Component: l.component,
		Operation: operation,
		Fields:    sanitizeFields(fields),
		File:      file,
		Line:      line,
	}

	if err != nil {
		entry.Error = sanitizeError(err).Error()
	}

	// Marshal to JSON for structured logging
	jsonBytes, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		// Fallback to simple logging if JSON marshaling fails
		l.Logger.Printf("MARSHAL_ERROR: %v | ORIGINAL: %s %s", marshalErr, level, message)
		return
	}

	l.Logger.Println(string(jsonBytes))
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(LevelDebug, message, nil, nil, "")
}

// DebugWithFields logs a debug message with additional fields
func (l *Logger) DebugWithFields(message string, fields map[string]interface{}) {
	l.log(LevelDebug, message, fields, nil, "")
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(LevelInfo, message, nil, nil, "")
}

// InfoWithFields logs an info message with additional fields
func (l *Logger) InfoWithFields(message string, fields map[string]interface{}) {
	l.log(LevelInfo, message, fields, nil, "")
}

// InfoWithOperation logs an info message with operation context
func (l *Logger) InfoWithOperation(operation, message string) {
	l.log(LevelInfo, message, nil, nil, operation)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(LevelWarn, message, nil, nil, "")
}

// WarnWithFields logs a warning message with additional fields
func (l *Logger) WarnWithFields(message string, fields map[string]interface{}) {
	l.log(LevelWarn, message, fields, nil, "")
}

// WarnWithError logs a warning message with an error
func (l *Logger) WarnWithError(message string, err error) {
	l.log(LevelWarn, message, nil, err, "")
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(LevelError, message, nil, nil, "")
}

// ErrorWithFields logs an error message with additional fields
func (l *Logger) ErrorWithFields(message string, fields map[string]interface{}) {
	l.log(LevelError, message, fields, nil, "")
}

// ErrorWithError logs an error message with an error
func (l *Logger) ErrorWithError(message string, err error) {
	l.log(LevelError, message, nil, err, "")
}

// ErrorWithOperation logs an error message with operation context
func (l *Logger) ErrorWithOperation(operation, message string, err error) {
	l.log(LevelError, message, nil, err, operation)
}

// LogOperation logs the start and completion of an operation
func (l *Logger) LogOperation(operation string, fn func() error) error {
	l.InfoWithOperation(operation, "Operation started")
	
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	fields := map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
	}
	
	if err != nil {
		l.log(LevelError, "Operation failed", fields, err, operation)
	} else {
		l.log(LevelInfo, "Operation completed successfully", fields, nil, operation)
	}
	
	return err
}

// sanitizeFields removes or masks sensitive information from log fields
func sanitizeFields(fields map[string]interface{}) map[string]interface{} {
	if fields == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	sensitiveKeys := map[string]bool{
		"password":     true,
		"secret":       true,
		"key":          true,
		"token":        true,
		"credential":   true,
		"access_key":   true,
		"secret_key":   true,
		"session_token": true,
		"presigned_url": true, // URLs may contain sensitive query parameters
	}

	for k, v := range fields {
		lowerKey := strings.ToLower(k)
		
		// Check if key contains sensitive information
		isSensitive := false
		for sensitiveKey := range sensitiveKeys {
			if strings.Contains(lowerKey, sensitiveKey) {
				isSensitive = true
				break
			}
		}
		
		if isSensitive {
			sanitized[k] = "[REDACTED]"
		} else {
			// For string values, check if they look like sensitive data
			if str, ok := v.(string); ok {
				sanitized[k] = sanitizeStringValue(str)
			} else {
				sanitized[k] = v
			}
		}
	}

	return sanitized
}

// sanitizeStringValue masks potentially sensitive string values
func sanitizeStringValue(value string) interface{} {
	// Mask AWS access keys (start with AKIA)
	if strings.HasPrefix(value, "AKIA") && len(value) == 20 {
		return "[AWS_ACCESS_KEY]"
	}
	
	// Mask long base64-like strings that might be secrets
	if len(value) > 40 && isBase64Like(value) {
		return "[MASKED_SECRET]"
	}
	
	// Mask URLs with query parameters (might contain sensitive tokens)
	if strings.Contains(value, "?") && (strings.Contains(value, "http://") || strings.Contains(value, "https://")) {
		parts := strings.Split(value, "?")
		if len(parts) > 1 {
			return parts[0] + "?[QUERY_PARAMS_REDACTED]"
		}
	}
	
	return value
}

// isBase64Like checks if a string looks like base64 encoding
func isBase64Like(s string) bool {
	if len(s) < 10 {
		return false
	}
	
	base64Chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	for _, char := range s {
		found := false
		for _, b64Char := range base64Chars {
			if char == b64Char {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// sanitizeError removes sensitive information from error messages
func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// Remove AWS access keys from error messages
	if strings.Contains(errMsg, "AKIA") {
		// Replace AWS access key pattern
		errMsg = strings.ReplaceAll(errMsg, "AKIA", "[AWS_ACCESS_KEY]")
	}
	
	// Remove file paths that might contain sensitive information
	if strings.Contains(errMsg, "/home/") || strings.Contains(errMsg, "C:\\Users\\") {
		errMsg = "Error with file operation (path redacted for security)"
	}
	
	// Remove presigned URLs from error messages
	if strings.Contains(errMsg, "amazonaws.com") && strings.Contains(errMsg, "?") {
		errMsg = "AWS S3 operation error (URL details redacted for security)"
	}
	
	return fmt.Errorf("%s", errMsg)
}