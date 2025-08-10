# File Sharing App

A secure, cross-platform desktop application for sharing files using AWS S3. Upload files, generate secure sharing links, and manage file expiration with a simple desktop interface.

## Features

- **Cross-Platform**: Native desktop application for Windows, macOS, and Linux
- **Secure File Sharing**: Upload files to AWS S3 with secure presigned URLs
- **Time-Based Expiration**: Automatic file cleanup after 1 hour, 1 day, 1 week, or 1 month
- **Local File Management**: Track and manage your shared files offline
- **Secure Credentials**: AWS credentials stored securely in OS keychain
- **Simple Interface**: Minimal, user-friendly desktop UI built with Fyne

## Quick Start

> **🚀 New User?** Check out the [Quick Start Guide](QUICK_START.md) for a 10-minute setup!

### 1. Download and Install

#### Windows
1. Download `file-sharing-app.exe` from the [releases page](../../releases)
2. Run the executable - no installation required
3. Windows may show a security warning - click "More info" then "Run anyway"

#### macOS
1. Download `file-sharing-app-darwin` from the [releases page](../../releases)
2. Open Terminal and make it executable: `chmod +x file-sharing-app-darwin`
3. Run: `./file-sharing-app-darwin`
4. If macOS blocks it, go to System Preferences > Security & Privacy and click "Open Anyway"

#### Linux
1. Download `file-sharing-app-linux` from the [releases page](../../releases)
2. Make it executable: `chmod +x file-sharing-app-linux`
3. Run: `./file-sharing-app-linux`

### 2. Set Up AWS Infrastructure

Before using the app, you need to deploy AWS infrastructure:

1. **Install AWS CLI** (if not already installed):
   ```bash
   # Windows (using PowerShell)
   msiexec.exe /i https://awscli.amazonaws.com/AWSCLIV2.msi
   
   # macOS
   curl "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o "AWSCLIV2.pkg"
   sudo installer -pkg AWSCLIV2.pkg -target /
   
   # Linux
   curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
   unzip awscliv2.zip
   sudo ./aws/install
   ```

2. **Configure AWS CLI** with your credentials:
   ```bash
   aws configure
   ```
   Enter your AWS Access Key ID, Secret Access Key, default region (e.g., `us-east-1`), and output format (`json`).

3. **Deploy Infrastructure**:
   ```bash
   # Clone this repository
   git clone <repository-url>
   cd file-sharing-app
   
   # Generate parameters and deploy
   ./infrastructure/scripts/generate-params.sh
   ./infrastructure/scripts/deploy.sh
   ```

4. **Save the Output**: After deployment, save the displayed AWS credentials and S3 bucket name - you'll need these for the app.

### 3. Configure the Application

1. Launch the File Sharing App
2. Click the "Settings" button
3. Enter your AWS configuration:
   - **AWS Region**: The region where you deployed (e.g., `us-east-1`)
   - **S3 Bucket Name**: From the deployment output
   - **AWS Access Key ID**: From the deployment output
   - **AWS Secret Access Key**: From the deployment output
4. Click "Save" - credentials are stored securely in your OS keychain

### 4. Start Sharing Files

1. Click "Upload File" or drag a file onto the upload area
2. Select expiration time (1 hour to 1 month)
3. Click "Upload" and wait for completion
4. Click "Share" next to your uploaded file
5. Add recipient email addresses and an optional message
6. Copy the generated sharing link and send it to recipients

## User Guide

### Uploading Files

1. **File Selection**: Click "Upload File" or drag files directly onto the upload area
2. **File Limits**: Maximum file size is 100MB per file
3. **Expiration**: Choose how long the file should remain accessible:
   - **1 Hour**: File deleted after 1 day (minimum AWS lifecycle period)
   - **1 Day**: File deleted after 1 day
   - **1 Week**: File deleted after 7 days
   - **1 Month**: File deleted after 30 days
4. **Progress**: Watch the progress bar during upload
5. **Completion**: File appears in your file list when upload completes

### Sharing Files

1. **Select File**: Click the "Share" button next to any uploaded file
2. **Add Recipients**: Enter email addresses (one per line or comma-separated)
3. **Custom Message**: Add an optional message for recipients
4. **Generate Link**: Click "Generate Share Link"
5. **Copy Link**: Copy the secure presigned URL and send to recipients
6. **Link Expiration**: Share links expire based on your file's expiration setting

### Managing Files

- **View Files**: All your uploaded files appear in the main list
- **File Details**: Click on a file to see upload date, expiration, and sharing history
- **Delete Files**: Click "Delete" to remove files from S3 and your local list
- **Offline Access**: View your file history even when offline
- **Status Tracking**: See file status (uploading, active, expired, error)

### Settings Configuration

Access settings via the "Settings" button:

- **AWS Region**: AWS region for your S3 bucket (e.g., `us-east-1`)
- **S3 Bucket Name**: Your unique S3 bucket name from infrastructure deployment
- **AWS Credentials**: Access Key ID and Secret Access Key (stored securely)
- **Default Expiration**: Default expiration time for new uploads
- **Theme**: Light or dark UI theme (if available)

## AWS Credential Configuration

The application supports multiple methods for AWS credential configuration:

### Method 1: Application Settings (Recommended)
1. Open the app and click "Settings"
2. Enter your AWS Access Key ID and Secret Access Key
3. Credentials are stored securely in your OS keychain:
   - **Windows**: Windows Credential Manager
   - **macOS**: Keychain Access
   - **Linux**: Secret Service (GNOME Keyring, KDE Wallet)

### Method 2: Environment Variables
Set these environment variables before launching the app:
```bash
export AWS_ACCESS_KEY_ID="your-access-key-id"
export AWS_SECRET_ACCESS_KEY="your-secret-access-key"
export AWS_DEFAULT_REGION="us-east-1"
```

### Method 3: AWS Credentials File
Create `~/.aws/credentials` file:
```ini
[default]
aws_access_key_id = your-access-key-id
aws_secret_access_key = your-secret-access-key
```

And `~/.aws/config` file:
```ini
[default]
region = us-east-1
```

### Method 4: IAM Roles (EC2/ECS)
If running on AWS infrastructure, the app can use IAM roles automatically.

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
├── infrastructure/          # AWS CloudFormation templates
├── go.mod
├── go.sum
└── README.md
```

## Development

For detailed development and building instructions, see [BUILD.md](BUILD.md).

### Prerequisites

- Go 1.19 or later
- AWS account with appropriate permissions
- Git (for cloning the repository)

### Building from Source

```bash
# Clone the repository
git clone <repository-url>
cd file-sharing-app

# Install dependencies
go mod download

# Build for current platform
make build-local

# Cross-compile for all platforms
make build

# Create distribution packages
make package
```

### Running from Source

```bash
go run ./cmd
```

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests (requires AWS credentials)
go test -tags=integration ./...

# Test builds
make test-build
```

## Documentation

### Getting Started
- **[Quick Start Guide](QUICK_START.md)**: Get up and running in 10 minutes
- **[AWS Setup Guide](AWS_SETUP_GUIDE.md)**: Complete guide for setting up AWS account and infrastructure
- **[Infrastructure README](infrastructure/README.md)**: Quick start for AWS infrastructure deployment

### Advanced Guides
- **[CloudFormation Guide](CLOUDFORMATION_GUIDE.md)**: Detailed CloudFormation deployment and management
- **[Build Guide](BUILD.md)**: Development, building, and packaging instructions

### Support
- **[Troubleshooting Guide](TROUBLESHOOTING.md)**: Solutions for common issues and problems
- **Application Logs**: Check logs for detailed error information:
  - Windows: `%APPDATA%\file-sharing-app\logs\`
  - macOS: `~/Library/Application Support/file-sharing-app/logs/`
  - Linux: `~/.local/share/file-sharing-app/logs/`

### Security
- AWS credentials stored securely in OS keychain
- All file transfers use HTTPS encryption
- Presigned URLs with time-based expiration
- Minimal IAM permissions following least-privilege principle
- Audit logging via AWS CloudTrail

### Cost Considerations
- **Free Tier**: Most usage covered by AWS Free Tier for first 12 months
- **Typical Costs**: $0.50-$8.00/month depending on usage
- **Cost Control**: Automatic file expiration and lifecycle policies
- **Monitoring**: Set up billing alerts to avoid unexpected charges

## Contributing

This is a personal project, but feedback and suggestions are welcome:

1. Check existing issues before creating new ones
2. Provide detailed information about bugs or feature requests
3. Include system information and logs when reporting issues
4. Test thoroughly before suggesting changes

## License

This project is licensed under a Non-Commercial License. You may use, modify, and distribute this software for personal, educational, or research purposes. Commercial use requires explicit written permission from the copyright holder.

See the [LICENSE](LICENSE) file for details.