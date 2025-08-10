package errors

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestNewAppError(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := NewAppError(ErrFileNotFound, "test message", cause)
	
	if err.Code != ErrFileNotFound {
		t.Errorf("Expected code %s, got %s", ErrFileNotFound, err.Code)
	}
	
	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message)
	}
	
	if err.Cause != cause {
		t.Errorf("Expected cause to be set")
	}
	
	if err.UserMessage == "" {
		t.Errorf("Expected user message to be set")
	}
	
	if err.Timestamp.IsZero() {
		t.Errorf("Expected timestamp to be set")
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		message  string
		cause    error
		expected string
	}{
		{
			name:     "error with cause",
			code:     ErrFileNotFound,
			message:  "test message",
			cause:    fmt.Errorf("underlying error"),
			expected: "FILE_NOT_FOUND: test message (caused by: underlying error)",
		},
		{
			name:     "error without cause",
			code:     ErrFileNotFound,
			message:  "test message",
			cause:    nil,
			expected: "FILE_NOT_FOUND: test message",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAppError(tt.code, tt.message, tt.cause)
			if err.Error() != tt.expected {
				t.Errorf("Expected error string '%s', got '%s'", tt.expected, err.Error())
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := NewAppError(ErrFileNotFound, "test message", cause)
	
	if err.Unwrap() != cause {
		t.Errorf("Expected unwrap to return the cause")
	}
}

func TestAppError_IsRecoverable(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected bool
	}{
		{"network error is recoverable", ErrNetworkError, true},
		{"timeout is recoverable", ErrConnectionTimeout, true},
		{"invalid credentials is not recoverable", ErrInvalidCredentials, false},
		{"file not found is not recoverable", ErrFileNotFound, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAppError(tt.code, "test message", nil)
			if err.IsRecoverable() != tt.expected {
				t.Errorf("Expected IsRecoverable() to return %v for %s", tt.expected, tt.code)
			}
		})
	}
}

func TestAppError_GetUserMessage(t *testing.T) {
	err := NewAppError(ErrFileNotFound, "technical message", nil)
	userMsg := err.GetUserMessage()
	
	if userMsg == "" {
		t.Errorf("Expected user message to be set")
	}
	
	if userMsg == "technical message" {
		t.Errorf("Expected user message to be different from technical message")
	}
	
	// Test with custom user message
	err.UserMessage = "custom user message"
	if err.GetUserMessage() != "custom user message" {
		t.Errorf("Expected custom user message to be returned")
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name         string
		inputError   error
		expectedCode ErrorCode
	}{
		{
			name:         "nil error",
			inputError:   nil,
			expectedCode: "",
		},
		{
			name:         "context deadline exceeded",
			inputError:   context.DeadlineExceeded,
			expectedCode: ErrConnectionTimeout,
		},
		{
			name:         "context canceled",
			inputError:   context.Canceled,
			expectedCode: ErrUploadCanceled,
		},
		{
			name:         "access denied error",
			inputError:   fmt.Errorf("AccessDenied: permission denied"),
			expectedCode: ErrS3AccessDenied,
		},
		{
			name:         "no such bucket error",
			inputError:   fmt.Errorf("NoSuchBucket: bucket does not exist"),
			expectedCode: ErrS3BucketNotFound,
		},
		{
			name:         "no such key error",
			inputError:   fmt.Errorf("NoSuchKey: key does not exist"),
			expectedCode: ErrS3ObjectNotFound,
		},
		{
			name:         "invalid access key error",
			inputError:   fmt.Errorf("InvalidAccessKeyId: invalid key"),
			expectedCode: ErrInvalidCredentials,
		},
		{
			name:         "file not found error",
			inputError:   fmt.Errorf("no such file or directory"),
			expectedCode: ErrFileNotFound,
		},
		{
			name:         "permission denied error",
			inputError:   fmt.Errorf("permission denied"),
			expectedCode: ErrAccessDenied,
		},
		{
			name:         "database error",
			inputError:   fmt.Errorf("database connection failed"),
			expectedCode: ErrDatabaseError,
		},
		{
			name:         "no rows error",
			inputError:   fmt.Errorf("sql: no rows in result set"),
			expectedCode: ErrRecordNotFound,
		},
		{
			name:         "unknown error",
			inputError:   fmt.Errorf("some unknown error"),
			expectedCode: ErrUnknownError,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.inputError)
			
			if tt.expectedCode == "" {
				if result != nil {
					t.Errorf("Expected nil result for nil input, got %v", result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("Expected non-nil result, got nil")
				return
			}
			
			if result.Code != tt.expectedCode {
				t.Errorf("Expected code %s, got %s", tt.expectedCode, result.Code)
			}
		})
	}
}

func TestClassifyError_AlreadyAppError(t *testing.T) {
	originalErr := NewAppError(ErrFileNotFound, "original message", nil)
	result := ClassifyError(originalErr)
	
	if result != originalErr {
		t.Errorf("Expected ClassifyError to return the same AppError instance")
	}
}

func TestClassifyError_NetworkError(t *testing.T) {
	// Create a mock network error
	netErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: fmt.Errorf("connection refused"),
	}
	
	result := ClassifyError(netErr)
	
	if result.Code != ErrNetworkError {
		t.Errorf("Expected code %s, got %s", ErrNetworkError, result.Code)
	}
}

func TestClassifyError_TimeoutError(t *testing.T) {
	// Create a mock timeout error
	timeoutErr := &timeoutError{message: "timeout occurred"}
	
	result := ClassifyError(timeoutErr)
	
	if result.Code != ErrConnectionTimeout {
		t.Errorf("Expected code %s, got %s", ErrConnectionTimeout, result.Code)
	}
}

func TestWrapError(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrappedErr := WrapError(originalErr, ErrUploadFailed, "upload failed")
	
	if wrappedErr.Code != ErrUploadFailed {
		t.Errorf("Expected code %s, got %s", ErrUploadFailed, wrappedErr.Code)
	}
	
	if wrappedErr.Message != "upload failed" {
		t.Errorf("Expected message 'upload failed', got '%s'", wrappedErr.Message)
	}
	
	if wrappedErr.Cause != originalErr {
		t.Errorf("Expected cause to be the original error")
	}
}

func TestWrapError_NilError(t *testing.T) {
	result := WrapError(nil, ErrUploadFailed, "upload failed")
	
	if result != nil {
		t.Errorf("Expected nil result for nil input error")
	}
}

func TestWrapError_ExistingAppError(t *testing.T) {
	originalAppErr := NewAppError(ErrFileNotFound, "file not found", nil)
	
	// Wrapping with empty code should preserve original
	result := WrapError(originalAppErr, "", "new message")
	
	if result != originalAppErr {
		t.Errorf("Expected to return the original AppError when code is empty")
	}
	
	// Wrapping with new code should create new error
	result = WrapError(originalAppErr, ErrUploadFailed, "new message")
	
	if result.Code != ErrUploadFailed {
		t.Errorf("Expected new code %s, got %s", ErrUploadFailed, result.Code)
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	
	operation := func() error {
		attempts++
		if attempts < 2 {
			return fmt.Errorf("temporary failure")
		}
		return nil // Success on second attempt
	}
	
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
	}
	
	err := RetryWithBackoff(ctx, operation, config)
	
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_NonRecoverableError(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	
	operation := func() error {
		attempts++
		return NewAppError(ErrInvalidCredentials, "invalid credentials", nil)
	}
	
	config := DefaultRetryConfig()
	config.BaseDelay = 1 * time.Millisecond
	
	err := RetryWithBackoff(ctx, operation, config)
	
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-recoverable error, got %d", attempts)
	}
	
	appErr, ok := err.(*AppError)
	if !ok {
		t.Errorf("Expected AppError, got %T", err)
	}
	
	if appErr.Code != ErrInvalidCredentials {
		t.Errorf("Expected code %s, got %s", ErrInvalidCredentials, appErr.Code)
	}
}

func TestRetryWithBackoff_MaxAttemptsReached(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	
	operation := func() error {
		attempts++
		return fmt.Errorf("network error") // Recoverable error
	}
	
	config := RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
	}
	
	err := RetryWithBackoff(ctx, operation, config)
	
	if err == nil {
		t.Errorf("Expected error after max attempts, got nil")
	}
	
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	
	operation := func() error {
		attempts++
		if attempts == 1 {
			// Cancel context after first attempt
			cancel()
		}
		return fmt.Errorf("network error") // Recoverable error
	}
	
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond, // Long enough to be canceled
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
	}
	
	err := RetryWithBackoff(ctx, operation, config)
	
	if err == nil {
		t.Errorf("Expected error due to context cancellation, got nil")
	}
	
	appErr, ok := err.(*AppError)
	if !ok {
		t.Errorf("Expected AppError, got %T", err)
	}
	
	if appErr.Code != ErrUploadCanceled {
		t.Errorf("Expected code %s, got %s", ErrUploadCanceled, appErr.Code)
	}
}

func TestIsTemporary(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected bool
	}{
		{
			name:     "recoverable app error",
			error:    NewAppError(ErrNetworkError, "network error", nil),
			expected: true,
		},
		{
			name:     "non-recoverable app error",
			error:    NewAppError(ErrInvalidCredentials, "invalid credentials", nil),
			expected: false,
		},
		{
			name:     "temporary network error",
			error:    &temporaryError{message: "temporary error"},
			expected: true,
		},
		{
			name:     "non-temporary network error",
			error:    &nonTemporaryError{message: "permanent error"},
			expected: false,
		},
		{
			name:     "regular error",
			error:    fmt.Errorf("regular error"),
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTemporary(tt.error)
			if result != tt.expected {
				t.Errorf("Expected IsTemporary() to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected bool
	}{
		{
			name:     "connection timeout app error",
			error:    NewAppError(ErrConnectionTimeout, "timeout", nil),
			expected: true,
		},
		{
			name:     "upload timeout app error",
			error:    NewAppError(ErrUploadTimeout, "upload timeout", nil),
			expected: true,
		},
		{
			name:     "non-timeout app error",
			error:    NewAppError(ErrFileNotFound, "file not found", nil),
			expected: false,
		},
		{
			name:     "timeout network error",
			error:    &timeoutError{message: "timeout error"},
			expected: true,
		},
		{
			name:     "non-timeout network error",
			error:    &nonTimeoutError{message: "non-timeout error"},
			expected: false,
		},
		{
			name:     "regular error",
			error:    fmt.Errorf("regular error"),
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTimeout(tt.error)
			if result != tt.expected {
				t.Errorf("Expected IsTimeout() to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsCanceled(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected bool
	}{
		{
			name:     "upload canceled app error",
			error:    NewAppError(ErrUploadCanceled, "canceled", nil),
			expected: true,
		},
		{
			name:     "non-canceled app error",
			error:    NewAppError(ErrFileNotFound, "file not found", nil),
			expected: false,
		},
		{
			name:     "context canceled",
			error:    context.Canceled,
			expected: true,
		},
		{
			name:     "regular error",
			error:    fmt.Errorf("regular error"),
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCanceled(tt.error)
			if result != tt.expected {
				t.Errorf("Expected IsCanceled() to return %v, got %v", tt.expected, result)
			}
		})
	}
}

// Mock error types for testing

type timeoutError struct {
	message string
}

func (e *timeoutError) Error() string   { return e.message }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return false }

type nonTimeoutError struct {
	message string
}

func (e *nonTimeoutError) Error() string   { return e.message }
func (e *nonTimeoutError) Timeout() bool   { return false }
func (e *nonTimeoutError) Temporary() bool { return false }

type temporaryError struct {
	message string
}

func (e *temporaryError) Error() string   { return e.message }
func (e *temporaryError) Timeout() bool   { return false }
func (e *temporaryError) Temporary() bool { return true }

type nonTemporaryError struct {
	message string
}

func (e *nonTemporaryError) Error() string   { return e.message }
func (e *nonTemporaryError) Timeout() bool   { return false }
func (e *nonTemporaryError) Temporary() bool { return false }