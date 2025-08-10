package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	logger := New()
	
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
	
	if logger.component != "app" {
		t.Errorf("Expected default component 'app', got '%s'", logger.component)
	}
	
	if logger.minLevel != LevelInfo {
		t.Errorf("Expected default level INFO, got %s", logger.minLevel)
	}
}

func TestNewWithComponent(t *testing.T) {
	logger := NewWithComponent("test-component")
	
	if logger.component != "test-component" {
		t.Errorf("Expected component 'test-component', got '%s'", logger.component)
	}
}

func TestSetLevel(t *testing.T) {
	logger := New()
	logger.SetLevel(LevelDebug)
	
	if logger.minLevel != LevelDebug {
		t.Errorf("Expected level DEBUG, got %s", logger.minLevel)
	}
}

func TestShouldLog(t *testing.T) {
	logger := New()
	logger.SetLevel(LevelWarn)
	
	tests := []struct {
		level    LogLevel
		expected bool
	}{
		{LevelDebug, false},
		{LevelInfo, false},
		{LevelWarn, true},
		{LevelError, true},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := logger.shouldLog(tt.level)
			if result != tt.expected {
				t.Errorf("Expected shouldLog(%s) to return %v, got %v", tt.level, tt.expected, result)
			}
		})
	}
}

func TestLogStructuredOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger:    log.New(&buf, "", 0),
		component: "test",
		minLevel:  LevelDebug,
	}
	
	logger.Info("test message")
	
	output := buf.String()
	if output == "" {
		t.Fatal("Expected log output, got empty string")
	}
	
	// Parse JSON output
	var entry LogEntry
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}
	
	if entry.Level != LevelInfo {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	
	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", entry.Message)
	}
	
	if entry.Component != "test" {
		t.Errorf("Expected component 'test', got '%s'", entry.Component)
	}
	
	if entry.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestLogWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger:    log.New(&buf, "", 0),
		component: "test",
		minLevel:  LevelDebug,
	}
	
	fields := map[string]interface{}{
		"user_id": "123",
		"action":  "upload",
		"count":   42,
	}
	
	logger.InfoWithFields("test message", fields)
	
	output := buf.String()
	var entry LogEntry
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}
	
	if entry.Fields == nil {
		t.Fatal("Expected fields to be set")
	}
	
	if entry.Fields["user_id"] != "123" {
		t.Errorf("Expected user_id '123', got '%v'", entry.Fields["user_id"])
	}
	
	if entry.Fields["action"] != "upload" {
		t.Errorf("Expected action 'upload', got '%v'", entry.Fields["action"])
	}
	
	if entry.Fields["count"] != float64(42) { // JSON unmarshals numbers as float64
		t.Errorf("Expected count 42, got '%v'", entry.Fields["count"])
	}
}

func TestLogWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger:    log.New(&buf, "", 0),
		component: "test",
		minLevel:  LevelDebug,
	}
	
	testErr := fmt.Errorf("test error")
	logger.ErrorWithError("operation failed", testErr)
	
	output := buf.String()
	var entry LogEntry
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}
	
	if entry.Error == "" {
		t.Error("Expected error field to be set")
	}
	
	if !strings.Contains(entry.Error, "test error") {
		t.Errorf("Expected error to contain 'test error', got '%s'", entry.Error)
	}
}

func TestLogWithOperation(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger:    log.New(&buf, "", 0),
		component: "test",
		minLevel:  LevelDebug,
	}
	
	logger.InfoWithOperation("file_upload", "uploading file")
	
	output := buf.String()
	var entry LogEntry
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}
	
	if entry.Operation != "file_upload" {
		t.Errorf("Expected operation 'file_upload', got '%s'", entry.Operation)
	}
}

func TestLogOperation(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger:    log.New(&buf, "", 0),
		component: "test",
		minLevel:  LevelDebug,
	}
	
	// Test successful operation
	err := logger.LogOperation("test_operation", func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 log lines, got %d", len(lines))
	}
	
	// Check start log
	var startEntry LogEntry
	err = json.Unmarshal([]byte(lines[0]), &startEntry)
	if err != nil {
		t.Fatalf("Failed to parse start log entry: %v", err)
	}
	
	if startEntry.Operation != "test_operation" {
		t.Errorf("Expected operation 'test_operation', got '%s'", startEntry.Operation)
	}
	
	if !strings.Contains(startEntry.Message, "started") {
		t.Errorf("Expected start message to contain 'started', got '%s'", startEntry.Message)
	}
	
	// Check completion log
	var endEntry LogEntry
	err = json.Unmarshal([]byte(lines[1]), &endEntry)
	if err != nil {
		t.Fatalf("Failed to parse end log entry: %v", err)
	}
	
	if endEntry.Level != LevelInfo {
		t.Errorf("Expected success level INFO, got %s", endEntry.Level)
	}
	
	if !strings.Contains(endEntry.Message, "completed") {
		t.Errorf("Expected completion message to contain 'completed', got '%s'", endEntry.Message)
	}
	
	if endEntry.Fields["duration_ms"] == nil {
		t.Error("Expected duration_ms field to be set")
	}
}

func TestLogOperation_WithError(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger:    log.New(&buf, "", 0),
		component: "test",
		minLevel:  LevelDebug,
	}
	
	testErr := fmt.Errorf("operation failed")
	
	// Test failed operation
	err := logger.LogOperation("test_operation", func() error {
		return testErr
	})
	
	if err != testErr {
		t.Errorf("Expected error to be returned, got %v", err)
	}
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 log lines, got %d", len(lines))
	}
	
	// Check failure log
	var endEntry LogEntry
	err = json.Unmarshal([]byte(lines[1]), &endEntry)
	if err != nil {
		t.Fatalf("Failed to parse end log entry: %v", err)
	}
	
	if endEntry.Level != LevelError {
		t.Errorf("Expected failure level ERROR, got %s", endEntry.Level)
	}
	
	if !strings.Contains(endEntry.Message, "failed") {
		t.Errorf("Expected failure message to contain 'failed', got '%s'", endEntry.Message)
	}
	
	if endEntry.Error == "" {
		t.Error("Expected error field to be set")
	}
}

func TestSanitizeFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil fields",
			input:    nil,
			expected: nil,
		},
		{
			name: "sensitive keys",
			input: map[string]interface{}{
				"password":     "secret123",
				"access_key":   "AKIAIOSFODNN7EXAMPLE",
				"secret_key":   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				"normal_field": "normal_value",
			},
			expected: map[string]interface{}{
				"password":     "[REDACTED]",
				"access_key":   "[REDACTED]",
				"secret_key":   "[REDACTED]",
				"normal_field": "normal_value",
			},
		},
		{
			name: "case insensitive sensitive keys",
			input: map[string]interface{}{
				"PASSWORD":   "secret123",
				"Secret_Key": "secret",
				"Token":      "token123",
			},
			expected: map[string]interface{}{
				"PASSWORD":   "[REDACTED]",
				"Secret_Key": "[REDACTED]",
				"Token":      "[REDACTED]",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFields(tt.input)
			
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d fields, got %d", len(tt.expected), len(result))
			}
			
			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("Expected field '%s' to be '%v', got '%v'", key, expectedValue, result[key])
				}
			}
		})
	}
}

func TestSanitizeStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "AWS access key",
			input:    "AKIAIOSFODNN7EXAMPLE",
			expected: "[AWS_ACCESS_KEY]",
		},
		{
			name:     "normal string",
			input:    "normal value",
			expected: "normal value",
		},
		{
			name:     "URL with query params",
			input:    "https://example.com/file?token=secret123&expires=123456",
			expected: "https://example.com/file?[QUERY_PARAMS_REDACTED]",
		},
		{
			name:     "URL without query params",
			input:    "https://example.com/file",
			expected: "https://example.com/file",
		},
		{
			name:     "long base64-like string",
			input:    "dGhpcyBpcyBhIHZlcnkgbG9uZyBiYXNlNjQgZW5jb2RlZCBzdHJpbmcgdGhhdCBtaWdodCBiZSBhIHNlY3JldA==",
			expected: "[MASKED_SECRET]",
		},
		{
			name:     "short base64-like string",
			input:    "dGVzdA==",
			expected: "dGVzdA==", // Too short to be considered sensitive
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeStringValue(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%v', got '%v'", tt.expected, result)
			}
		})
	}
}

func TestIsBase64Like(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid base64",
			input:    "dGhpcyBpcyBhIHRlc3Q=",
			expected: true,
		},
		{
			name:     "valid base64 without padding",
			input:    "dGhpcyBpcyBhIHRlc3Q",
			expected: true,
		},
		{
			name:     "too short",
			input:    "dGVzdA",
			expected: false,
		},
		{
			name:     "contains invalid characters",
			input:    "this is not base64!",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBase64Like(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected string
	}{
		{
			name:     "nil error",
			input:    nil,
			expected: "",
		},
		{
			name:     "error with AWS access key",
			input:    fmt.Errorf("failed to authenticate with key AKIAIOSFODNN7EXAMPLE"),
			expected: "failed to authenticate with key [AWS_ACCESS_KEY]IOSFODNN7EXAMPLE",
		},
		{
			name:     "error with file path",
			input:    fmt.Errorf("failed to read file /home/user/secret.txt"),
			expected: "Error with file operation (path redacted for security)",
		},
		{
			name:     "error with presigned URL",
			input:    fmt.Errorf("failed to access https://bucket.amazonaws.com/file?token=secret"),
			expected: "AWS S3 operation error (URL details redacted for security)",
		},
		{
			name:     "normal error",
			input:    fmt.Errorf("normal error message"),
			expected: "normal error message",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeError(tt.input)
			
			if tt.input == nil {
				if result != nil {
					t.Errorf("Expected nil result for nil input, got %v", result)
				}
				return
			}
			
			if result.Error() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result.Error())
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger:    log.New(&buf, "", 0),
		component: "test",
		minLevel:  LevelDebug,
	}
	
	// Test all log levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	if len(lines) != 4 {
		t.Fatalf("Expected 4 log lines, got %d", len(lines))
	}
	
	expectedLevels := []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError}
	
	for i, line := range lines {
		var entry LogEntry
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			t.Fatalf("Failed to parse log line %d: %v", i, err)
		}
		
		if entry.Level != expectedLevels[i] {
			t.Errorf("Expected level %s at line %d, got %s", expectedLevels[i], i, entry.Level)
		}
	}
}

func TestLogFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger:    log.New(&buf, "", 0),
		component: "test",
		minLevel:  LevelWarn, // Only WARN and ERROR should be logged
	}
	
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// Should only have 2 lines (WARN and ERROR)
	if len(lines) != 2 {
		t.Fatalf("Expected 2 log lines, got %d", len(lines))
	}
	
	// Check first line is WARN
	var warnEntry LogEntry
	err := json.Unmarshal([]byte(lines[0]), &warnEntry)
	if err != nil {
		t.Fatalf("Failed to parse warn log entry: %v", err)
	}
	
	if warnEntry.Level != LevelWarn {
		t.Errorf("Expected first line to be WARN, got %s", warnEntry.Level)
	}
	
	// Check second line is ERROR
	var errorEntry LogEntry
	err = json.Unmarshal([]byte(lines[1]), &errorEntry)
	if err != nil {
		t.Fatalf("Failed to parse error log entry: %v", err)
	}
	
	if errorEntry.Level != LevelError {
		t.Errorf("Expected second line to be ERROR, got %s", errorEntry.Level)
	}
}