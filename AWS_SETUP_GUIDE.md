# AWS Setup Guide

This guide walks you through setting up AWS infrastructure for the File Sharing App, from creating an AWS account to configuring the application.

## Table of Contents

1. [AWS Account Setup](#aws-account-setup)
2. [AWS CLI Installation and Configuration](#aws-cli-installation-and-configuration)
3. [Infrastructure Deployment](#infrastructure-deployment)
4. [Application Configuration](#application-configuration)
5. [Security Best Practices](#security-best-practices)
6. [Cost Management](#cost-management)
7. [Troubleshooting](#troubleshooting)

## AWS Account Setup

### 1. Create AWS Account

If you don't have an AWS account:

1. Go to [aws.amazon.com](https://aws.amazon.com)
2. Click "Create an AWS Account"
3. Follow the registration process
4. Provide payment information (required even for free tier)
5. Verify your phone number and email
6. Choose the Basic support plan (free)

### 2. Understand AWS Free Tier

The File Sharing App uses services covered by AWS Free Tier:

- **S3**: 5 GB storage, 20,000 GET requests, 2,000 PUT requests per month
- **CloudTrail**: One trail with management events (free)
- **CloudWatch Logs**: 5 GB ingestion, 5 GB storage, 5 GB data scanned per month

**Important**: Charges may apply if you exceed free tier limits or use the service beyond 12 months.

### 3. Set Up Billing Alerts

To avoid unexpected charges:

1. Go to AWS Console → Billing Dashboard
2. Click "Billing preferences"
3. Enable "Receive Billing Alerts"
4. Go to CloudWatch → Alarms
5. Create a billing alarm for your desired threshold (e.g., $5)

## AWS CLI Installation and Configuration

### Installation

#### Windows
```powershell
# Download and install MSI
Invoke-WebRequest -Uri "https://awscli.amazonaws.com/AWSCLIV2.msi" -OutFile "AWSCLIV2.msi"
Start-Process msiexec.exe -ArgumentList "/i AWSCLIV2.msi /quiet" -Wait

# Verify installation
aws --version
```

#### macOS
```bash
# Using Homebrew (recommended)
brew install awscli

# Or download installer
curl "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o "AWSCLIV2.pkg"
sudo installer -pkg AWSCLIV2.pkg -target /

# Verify installation
aws --version
```

#### Linux
```bash
# Ubuntu/Debian
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# CentOS/RHEL/Amazon Linux
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# Verify installation
aws --version
```

### Configuration

#### Option 1: Create IAM User (Recommended for Personal Use)

1. **Create IAM User**:
   - Go to AWS Console → IAM → Users
   - Click "Add users"
   - Username: `file-sharing-admin`
   - Access type: "Programmatic access"
   - Click "Next: Permissions"

2. **Attach Policies**:
   - Click "Attach existing policies directly"
   - Search and select these policies:
     - `AmazonS3FullAccess`
     - `CloudFormationFullAccess`
     - `IAMFullAccess`
     - `CloudTrailFullAccess`
     - `CloudWatchLogsFullAccess`
   - Click "Next" through remaining steps
   - Click "Create user"

3. **Save Credentials**:
   - **Important**: Download the CSV file or copy the Access Key ID and Secret Access Key
   - You won't be able to see the Secret Access Key again

4. **Configure AWS CLI**:
   ```bash
   aws configure
   ```
   Enter:
   - AWS Access Key ID: [Your Access Key ID]
   - AWS Secret Access Key: [Your Secret Access Key]
   - Default region name: `us-east-1` (or your preferred region)
   - Default output format: `json`

#### Option 2: Use Root Account (Not Recommended)

If you must use root account credentials:

1. Go to AWS Console → Security Credentials
2. Create Access Key under "Access keys"
3. Configure AWS CLI as above

**Warning**: Root account has full access to everything. Create IAM users instead.

### Verify Configuration

```bash
# Test AWS CLI configuration
aws sts get-caller-identity

# Should return your account ID and user ARN
```

## Infrastructure Deployment

### 1. Download Application Code

```bash
# Clone the repository
git clone <repository-url>
cd file-sharing-app

# Or download and extract ZIP file
```

### 2. Choose AWS Region

Select a region close to your users for better performance:

- **US East (N. Virginia)**: `us-east-1` - Lowest cost, most services
- **US West (Oregon)**: `us-west-2` - Good for US West Coast
- **Europe (Ireland)**: `eu-west-1` - Good for Europe
- **Asia Pacific (Singapore)**: `ap-southeast-1` - Good for Asia

### 3. Generate Parameters

The deployment script needs unique names for AWS resources:

```bash
# Navigate to infrastructure directory
cd infrastructure

# Generate parameters file (automatically includes your AWS account ID)
./scripts/generate-params.sh

# Or specify account ID manually if the script can't detect it
./scripts/generate-params.sh 123456789012
```

This creates `cloudformation/parameters.json` with unique resource names.

### 4. Deploy Infrastructure

```bash
# Deploy to default region (eu-central-1)
./scripts/deploy.sh

# Or deploy to specific region
./scripts/deploy.sh us-east-1

# The script will:
# 1. Validate the CloudFormation template
# 2. Deploy the stack
# 3. Wait for completion
# 4. Display stack outputs
```

### 5. Save Stack Outputs

After successful deployment, you'll see output like:
```
Stack Outputs:
S3BucketName: file-sharing-app-bucket-123456789012
IAMAccessKeyId: AKIA...
IAMSecretAccessKey: ...
CloudTrailArn: arn:aws:cloudtrail:us-east-1:123456789012:trail/file-sharing-app-trail
AuditLogsBucket: file-sharing-app-audit-logs-123456789012
```

**Important**: Save these values securely - you'll need them to configure the application.

## Application Configuration

### 1. Launch the Application

Download and run the File Sharing App for your platform.

### 2. Configure AWS Settings

1. Click the "Settings" button in the application
2. Enter the following information from your stack outputs:
   - **AWS Region**: The region where you deployed (e.g., `us-east-1`)
   - **S3 Bucket Name**: From `S3BucketName` output
   - **AWS Access Key ID**: From `IAMAccessKeyId` output
   - **AWS Secret Access Key**: From `IAMSecretAccessKey` output
3. Click "Save"

The application will securely store these credentials in your OS keychain.

### 3. Test the Configuration

1. Try uploading a small test file
2. Verify it appears in your file list
3. Generate a sharing link and test it in a browser
4. Check that the file appears in your S3 bucket (AWS Console → S3)

## Security Best Practices

### 1. Credential Management

**Do:**
- Use IAM users with minimal required permissions
- Store credentials securely (OS keychain, not plain text files)
- Rotate access keys regularly (every 90 days)
- Enable MFA on your AWS account

**Don't:**
- Share AWS credentials with others
- Commit credentials to version control
- Use root account credentials for applications
- Store credentials in plain text files

### 2. Access Control

The deployed IAM user has minimal permissions:
- S3 operations only on your specific bucket
- No access to other AWS services
- No ability to create/modify IAM policies

### 3. Monitoring

**Enable CloudTrail** (included in deployment):
- Logs all API calls to your S3 bucket
- Helps detect unauthorized access
- Stored in separate audit bucket

**Monitor Costs**:
- Set up billing alerts
- Review AWS Cost Explorer monthly
- Monitor S3 storage usage

### 4. Network Security

**S3 Bucket Security**:
- Public access is completely blocked
- HTTPS-only access enforced
- Server-side encryption enabled

**Presigned URLs**:
- Limited expiration time (max 7 days)
- Unique, non-guessable URLs
- Automatic expiration

### 5. Data Protection

**Encryption**:
- S3 server-side encryption (AES-256)
- Local database encryption
- Secure credential storage

**Backup**:
- S3 versioning enabled
- CloudTrail logs for audit trail
- Local file metadata backup

## Cost Management

### Understanding Costs

**S3 Storage**:
- $0.023 per GB per month (first 50 TB)
- $0.0004 per 1,000 PUT requests
- $0.0004 per 10,000 GET requests

**Data Transfer**:
- First 1 GB out to internet per month: Free
- Next 9.999 TB: $0.09 per GB

**CloudTrail**:
- Management events: Free
- Data events: $0.10 per 100,000 events

### Cost Optimization

1. **File Expiration**: Use shorter expiration times to reduce storage costs
2. **Monitor Usage**: Check AWS Cost Explorer monthly
3. **Clean Up**: Delete old CloudFormation stacks you're not using
4. **Free Tier**: Stay within free tier limits for first 12 months

### Estimated Monthly Costs

For typical usage (assuming you exceed free tier):
- **Light usage** (1 GB storage, 100 files): ~$0.50/month
- **Medium usage** (5 GB storage, 500 files): ~$2.00/month
- **Heavy usage** (20 GB storage, 2000 files): ~$8.00/month

## Troubleshooting

### Common Deployment Issues

#### 1. "Bucket name already exists"
```bash
# Edit parameters file with a different bucket name
nano cloudformation/parameters.json

# Change BucketName to something unique
"ParameterValue": "file-sharing-app-bucket-your-unique-suffix"
```

#### 2. "Insufficient permissions"
Ensure your AWS user has these policies:
- `CloudFormationFullAccess`
- `AmazonS3FullAccess`
- `IAMFullAccess`
- `CloudTrailFullAccess`

#### 3. "Stack rollback"
```bash
# Check what failed
aws cloudformation describe-stack-events --stack-name file-sharing-app

# Delete failed stack and retry
aws cloudformation delete-stack --stack-name file-sharing-app
aws cloudformation wait stack-delete-complete --stack-name file-sharing-app

# Then redeploy
./scripts/deploy.sh
```

### Verification Commands

```bash
# Check if stack deployed successfully
aws cloudformation describe-stacks --stack-name file-sharing-app

# List stack resources
aws cloudformation list-stack-resources --stack-name file-sharing-app

# Test S3 bucket access
aws s3 ls s3://your-bucket-name

# Test IAM user permissions
aws sts get-caller-identity
```

### Clean Up Resources

To delete all AWS resources and stop charges:

```bash
# Empty S3 buckets first (required before deletion)
aws s3 rm s3://your-bucket-name --recursive
aws s3 rm s3://your-audit-bucket-name --recursive

# Delete CloudFormation stack
aws cloudformation delete-stack --stack-name file-sharing-app

# Wait for deletion to complete
aws cloudformation wait stack-delete-complete --stack-name file-sharing-app

# Verify deletion
aws cloudformation describe-stacks --stack-name file-sharing-app
# Should return "does not exist" error
```

## Advanced Configuration

### Multi-Region Deployment

To deploy in multiple regions:

```bash
# Deploy to multiple regions
./scripts/deploy.sh us-east-1
./scripts/deploy.sh eu-west-1
./scripts/deploy.sh ap-southeast-1

# Configure application to use the closest region
```

### Custom Domain Names

To use custom domain for S3 bucket:

1. Register domain in Route 53
2. Create CloudFront distribution
3. Configure custom SSL certificate
4. Update application configuration

### Monitoring and Alerting

Set up additional monitoring:

```bash
# Create CloudWatch alarm for high S3 costs
aws cloudwatch put-metric-alarm \
  --alarm-name "S3-High-Costs" \
  --alarm-description "Alert when S3 costs exceed $10" \
  --metric-name EstimatedCharges \
  --namespace AWS/Billing \
  --statistic Maximum \
  --period 86400 \
  --threshold 10 \
  --comparison-operator GreaterThanThreshold
```

## Next Steps

After successful setup:

1. **Test thoroughly**: Upload various file types and sizes
2. **Share with others**: Test the sharing functionality
3. **Monitor costs**: Check AWS billing dashboard weekly
4. **Plan for growth**: Consider upgrading to paid support if needed
5. **Stay updated**: Keep the application updated to the latest version

For ongoing support and updates, refer to the main [README.md](README.md) and [TROUBLESHOOTING.md](TROUBLESHOOTING.md) guides.