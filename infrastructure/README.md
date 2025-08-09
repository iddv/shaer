# File Sharing App - AWS Infrastructure

This directory contains the AWS CloudFormation infrastructure for the File Sharing Application.

## Overview

The infrastructure includes:
- S3 bucket with versioning and server-side encryption
- IAM user with least-privilege policies for S3 operations
- S3 lifecycle policies for automatic cleanup of expired files
- CloudTrail for audit logging
- CloudWatch Log Group for S3 access logging

## Prerequisites

1. **AWS CLI**: Install and configure the AWS CLI
   ```bash
   # Install AWS CLI (if not already installed)
   curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
   unzip awscliv2.zip
   sudo ./aws/install
   
   # Configure AWS credentials
   aws configure
   ```

2. **AWS Permissions**: Your AWS user/role needs the following permissions:
   - CloudFormation: Full access
   - S3: Full access
   - IAM: Create users, policies, and access keys
   - CloudTrail: Create and manage trails
   - CloudWatch Logs: Create log groups

## Quick Start

1. **Generate Parameters File**:
   ```bash
   # This will automatically detect your AWS account ID
   ./infrastructure/scripts/generate-params.sh
   
   # Or specify account ID manually
   ./infrastructure/scripts/generate-params.sh 123456789012
   ```

2. **Deploy Infrastructure**:
   ```bash
   # Deploy to eu-central-1 (default)
   ./infrastructure/scripts/deploy.sh
   
   # Deploy to a specific region
   ./infrastructure/scripts/deploy.sh us-east-1
   ```

3. **Get Stack Outputs**:
   After deployment, the script will display the stack outputs including:
   - S3 bucket name
   - IAM user credentials (Access Key ID and Secret Access Key)
   - CloudTrail ARN
   - Audit logs bucket name

## Configuration

### Parameters File

The `infrastructure/cloudformation/parameters.json` file contains:

```json
[
  {
    "ParameterKey": "BucketName",
    "ParameterValue": "file-sharing-app-bucket-123456789012"
  },
  {
    "ParameterKey": "IAMUserName", 
    "ParameterValue": "file-sharing-app-user-123456789012"
  },
  {
    "ParameterKey": "Environment",
    "ParameterValue": "production"
  }
]
```

**Important**: The bucket name must be globally unique across all AWS accounts. The generate-params script includes your account ID to help ensure uniqueness.

### CloudFormation Template

The `infrastructure/cloudformation/file-sharing-app.yaml` template includes:

#### S3 Bucket Configuration
- **Encryption**: AES256 server-side encryption
- **Versioning**: Enabled for data protection
- **Public Access**: Completely blocked
- **Lifecycle Policies**: Automatic cleanup based on file tags:
  - `expiration=1hour`: Files deleted after 1 day
  - `expiration=1day`: Files deleted after 1 day  
  - `expiration=1week`: Files deleted after 7 days
  - `expiration=1month`: Files deleted after 30 days

#### IAM User Permissions
The IAM user has minimal required permissions:
- `s3:PutObject`, `s3:PutObjectAcl`, `s3:PutObjectTagging`
- `s3:GetObject`, `s3:GetObjectAcl`, `s3:GetObjectTagging`
- `s3:DeleteObject`, `s3:DeleteObjectTagging`
- `s3:ListBucket`, `s3:GetBucketLocation`, `s3:GetBucketVersioning`
- `s3:HeadObject`

#### Security Features
- **CloudTrail**: Audit logging for all S3 operations
- **CloudWatch Logs**: S3 access logging
- **Separate Audit Bucket**: CloudTrail logs stored in dedicated bucket
- **Lifecycle Management**: Audit logs retained for 90 days

## Usage in Application

After deployment, configure your application with the outputs:

```go
// Example Go configuration
config := &aws.Config{
    Region:      "eu-central-1", // or your chosen region
    Credentials: credentials.NewStaticCredentials(
        "AKIA...", // Access Key ID from stack output
        "...",     // Secret Access Key from stack output  
        "",
    ),
}

bucketName := "file-sharing-app-bucket-123456789012" // From stack output
```

## File Expiration

Files are automatically deleted based on their `expiration` tag:

```go
// Example: Upload file with 1 week expiration
_, err := s3Client.PutObject(&s3.PutObjectInput{
    Bucket: aws.String(bucketName),
    Key:    aws.String("shared-file.pdf"),
    Body:   fileReader,
    Tagging: aws.String("expiration=1week"),
})
```

Available expiration values:
- `1hour` - Deleted after 1 day (minimum S3 lifecycle period)
- `1day` - Deleted after 1 day
- `1week` - Deleted after 7 days  
- `1month` - Deleted after 30 days

## Security Best Practices

1. **Credential Management**:
   - Store the Secret Access Key securely (consider AWS Secrets Manager)
   - Rotate access keys regularly
   - Never commit credentials to version control

2. **Monitoring**:
   - Monitor CloudTrail logs for suspicious activity
   - Set up CloudWatch alarms for unusual S3 access patterns
   - Review S3 access logs regularly

3. **Network Security**:
   - Consider using VPC endpoints for S3 access
   - Implement IP restrictions if needed

## Troubleshooting

### Common Issues

1. **Bucket Name Already Exists**:
   - S3 bucket names must be globally unique
   - Modify the `BucketName` parameter in `parameters.json`
   - Use the generate-params script to include your account ID

2. **Insufficient Permissions**:
   - Ensure your AWS user has CloudFormation, S3, and IAM permissions
   - Check AWS CloudTrail logs for permission denied errors

3. **Stack Update Failures**:
   - Some resources (like S3 buckets) cannot be renamed
   - Delete and recreate the stack if major changes are needed
   - Check CloudFormation events in the AWS Console

### Cleanup

To delete all resources:

```bash
# Delete the CloudFormation stack
aws cloudformation delete-stack --stack-name file-sharing-app --region eu-central-1

# Wait for deletion to complete
aws cloudformation wait stack-delete-complete --stack-name file-sharing-app --region eu-central-1
```

**Note**: S3 buckets must be empty before deletion. The script does not automatically empty buckets.

## Cost Optimization

- **S3 Storage**: Costs depend on storage usage and requests
- **CloudTrail**: Data events incur additional charges
- **CloudWatch Logs**: Log retention costs
- **Data Transfer**: Outbound data transfer charges apply

Consider implementing S3 Intelligent Tiering for cost optimization on larger files.