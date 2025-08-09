# File Sharing App

A cross-platform desktop application for secure file sharing using AWS S3.

## Features

- Cross-platform desktop application (Windows, macOS, Linux)
- Secure file upload to AWS S3
- Time-based file expiration
- Presigned URL generation for secure sharing
- Local file tracking and management
- Secure credential storage using OS keychain

## Project Structure

```
file-sharing-app/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── aws/                 # AWS S3 integration
│   ├── config/              # Configuration management
│   ├── models/              # Data models
│   ├── storage/             # Local database layer
│   └── ui/                  # User interface components
├── pkg/
│   └── logger/              # Logging utilities
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

- **Fyne v2**: Cross-platform GUI framework
- **AWS SDK v2**: AWS S3 integration
- **SQLite**: Local database storage
- **99designs/keyring**: Secure credential storage

## Development

### Prerequisites

- Go 1.19 or later
- AWS account with S3 access

### Building

```bash
# Build for current platform
go build -o file-sharing-app ./cmd

# Cross-compile for different platforms
GOOS=windows GOARCH=amd64 go build -o file-sharing-app.exe ./cmd
GOOS=darwin GOARCH=amd64 go build -o file-sharing-app-darwin ./cmd
GOOS=linux GOARCH=amd64 go build -o file-sharing-app-linux ./cmd
```

### Running

```bash
go run ./cmd
```

## Configuration

The application requires AWS credentials to be configured. Supported methods:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. AWS credentials file (`~/.aws/credentials`)
3. IAM roles (for EC2 instances)

## License

MIT License