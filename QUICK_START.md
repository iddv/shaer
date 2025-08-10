# Quick Start Guide

Get the File Sharing App running in under 10 minutes.

## Prerequisites

- AWS account (free tier eligible)
- Basic command line knowledge
- Internet connection

## Step 1: Download the App

1. Go to the [releases page](../../releases)
2. Download the file for your operating system:
   - **Windows**: `file-sharing-app.exe`
   - **macOS**: `file-sharing-app-darwin`
   - **Linux**: `file-sharing-app-linux`

## Step 2: Set Up AWS

### Install AWS CLI

**Windows (PowerShell as Administrator):**
```powershell
msiexec.exe /i https://awscli.amazonaws.com/AWSCLIV2.msi /quiet
```

**macOS:**
```bash
curl "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o "AWSCLIV2.pkg"
sudo installer -pkg AWSCLIV2.pkg -target /
```

**Linux:**
```bash
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install
```

### Configure AWS

1. **Create AWS account** at [aws.amazon.com](https://aws.amazon.com) if you don't have one
2. **Get AWS credentials**:
   - Go to AWS Console → IAM → Users → Add users
   - Username: `file-sharing-admin`
   - Access type: "Programmatic access"
   - Permissions: Attach existing policies → Select:
     - `AmazonS3FullAccess`
     - `CloudFormationFullAccess`
     - `IAMFullAccess`
     - `CloudTrailFullAccess`
   - Save the Access Key ID and Secret Access Key

3. **Configure AWS CLI**:
   ```bash
   aws configure
   ```
   Enter:
   - AWS Access Key ID: [Your Access Key]
   - AWS Secret Access Key: [Your Secret Key]
   - Default region: `us-east-1`
   - Default output format: `json`

## Step 3: Deploy Infrastructure

```bash
# Download the project
git clone <repository-url>
cd file-sharing-app/infrastructure

# Generate unique parameters
./scripts/generate-params.sh

# Deploy AWS infrastructure
./scripts/deploy.sh us-east-1
```

**Save the output!** You'll see something like:
```
S3BucketName: file-sharing-app-bucket-123456789012
IAMAccessKeyId: AKIA...
IAMSecretAccessKey: ...
```

## Step 4: Configure the App

1. **Run the application**:
   - **Windows**: Double-click `file-sharing-app.exe`
   - **macOS**: `chmod +x file-sharing-app-darwin && ./file-sharing-app-darwin`
   - **Linux**: `chmod +x file-sharing-app-linux && ./file-sharing-app-linux`

2. **Click "Settings"** and enter:
   - AWS Region: `us-east-1` (or your chosen region)
   - S3 Bucket Name: (from deployment output)
   - AWS Access Key ID: (from deployment output)
   - AWS Secret Access Key: (from deployment output)

3. **Click "Save"**

## Step 5: Test It

1. Click "Upload File" and select a small test file
2. Choose expiration time (e.g., "1 day")
3. Click "Upload" and wait for completion
4. Click "Share" next to your file
5. Add your email address
6. Click "Generate Share Link"
7. Copy the link and test it in a browser

## That's It!

You now have a secure file sharing app running. Files are automatically deleted based on your expiration settings.

## Need Help?

- **Issues**: Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **Detailed Setup**: See [AWS_SETUP_GUIDE.md](AWS_SETUP_GUIDE.md)
- **Infrastructure**: See [CLOUDFORMATION_GUIDE.md](CLOUDFORMATION_GUIDE.md)

## Cost Estimate

For typical usage:
- **Free Tier**: First 12 months mostly free
- **After Free Tier**: ~$0.50-$2.00/month for light usage
- **Set billing alerts** in AWS Console to avoid surprises

## Security Notes

- Your AWS credentials are stored securely in your OS keychain
- All file transfers use HTTPS encryption
- Files are automatically deleted based on expiration settings
- Sharing links expire with the files