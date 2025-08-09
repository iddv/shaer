package logger

import (
	"log"
	"os"
)

// Logger provides structured logging capabilities
type Logger struct {
	*log.Logger
}

// New creates a new logger instance
func New() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[FILE-SHARING] ", log.LstdFlags|log.Lshortfile),
	}
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.Printf("ERROR: %s", msg)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.Printf("DEBUG: %s", msg)
}