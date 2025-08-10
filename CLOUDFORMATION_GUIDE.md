# CloudFormation Deployment Guide

This guide provides detailed information about the AWS CloudFormation infrastructure for the File Sharing App, including template structure, deployment options, and management procedures.

## Table of Contents

1. [Overview](#overview)
2. [Template Structure](#template-structure)
3. [Parameters](#parameters)
4. [Resources Created](#resources-created)
5. [Deployment Methods](#deployment-methods)
6. [Stack Management](#stack-management)
7. [Customization](#customization)
8. [Monitoring and Maintenance](#monitoring-and-maintenance)
9. [Troubleshooting](#troubleshooting)

## Overview

The CloudFormation template (`infrastructure/cloudformation/file-sharing-app.yaml`) creates a complete AWS infrastructure for secure file sharing, including:

- S3 bucket with security configurations
- IAM user with least-privilege permissions
- S3 lifecycle policies for automatic cleanup
- CloudTrail for audit logging
- CloudWatch Log Group for monitoring

### Architecture Diagram

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Desktop App   │───▶│   IAM User       │───▶│   S3 Bucket     │
│                 │    │   (Minimal       │    │   (Encrypted)   │
│                 │    │    Permissions)  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │   CloudTrail     │    │   Lifecycle     │
                       │   (Audit Logs)   │    │   Policies      │
                       └──────────────────┘    └─────────────────┘
```

## Template Structure

### File Organization

```
infrastructure/
├── cloudformation/
│   ├── file-sharing-app.yaml           # Main CloudFormation template
│   ├── parameters.json                 # Default parameters
│   ├── parameters-dev-us-west-2.json   # Environment-specific parameters
│   └── parameters-prod-us-west-2.json  # Production parameters
├── scripts/
│   ├── deploy.sh                       # Deployment script
│   └── generate-params.sh              # Parameter generation script
└── README.md                           # Infrastructure documentation
```

### Template Sections

The CloudFormation template includes:

1. **AWSTemplateFormatVersion**: `2010-09-09`
2. **Description**: Template purpose and version
3. **Parameters**: Configurable values
4. **Resources**: AWS resources to create
5. **Outputs**: Values returned after deployment

## Parameters

### Required Parameters

| Parameter | Description | Default | Example |
|-----------|-------------|---------|---------|
| `BucketName` | Unique S3 bucket name | None | `file-sharing-app-bucket-123456789012` |
| `IAMUserName` | IAM user name | None | `file-sharing-app-user-123456789012` |
| `Environment` | Environment tag | `production` | `development`, `staging`, `production` |

### Optional Parameters

| Parameter | Description | Default | Options |
|-----------|-------------|---------|---------|
| `EnableVersioning` | Enable S3 versioning | `true` | `true`, `false` |
| `EnableCloudTrail` | Enable audit logging | `true` | `true`, `false` |
| `LogRetentionDays` | CloudWatch log retention | `30` | `1`, `3`, `5`, `7`, `14`, `30`, `60`, `90`, `120`, `150`, `180`, `365`, `400`, `545`, `731`, `1827`, `3653` |

### Parameter File Format

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

## Resources Created

### 1. S3 Bucket (`FileShareBucket`)

**Purpose**: Stores uploaded files with security and lifecycle management.

**Configuration**:
```yaml
Type: AWS::S3::Bucket
Properties:
  BucketName: !Ref BucketName
  BucketEncryption:
    ServerSideEncryptionConfiguration:
      - ServerSideEncryptionByDefault:
          SSEAlgorithm: AES256
  VersioningConfiguration:
    Status: Enabled
  PublicAccessBlockConfiguration:
    BlockPublicAcls: true
    BlockPublicPolicy: true
    IgnorePublicAcls: true
    RestrictPublicBuckets: true
  LifecycleConfiguration:
    Rules:
      - Id: ExpireFiles1Hour
        Status: Enabled
        Filter:
          Tag:
            Key: expiration
            Value: 1hour
        ExpirationInDays: 1
      # Additional rules for 1day, 1week, 1month
```

**Features**:
- **Encryption**: AES-256 server-side encryption
- **Versioning**: Enabled for data protection
- **Public Access**: Completely blocked
- **Lifecycle Policies**: Automatic cleanup based on tags

### 2. S3 Bucket Policy (`FileShareBucketPolicy`)

**Purpose**: Enforces HTTPS-only access and additional security controls.

```yaml
Type: AWS::S3::BucketPolicy
Properties:
  Bucket: !Ref FileShareBucket
  PolicyDocument:
    Statement:
      - Sid: DenyInsecureConnections
        Effect: Deny
        Principal: "*"
        Action: "s3:*"
        Resource:
          - !Sub "${FileShareBucket}/*"
          - !Ref FileShareBucket
        Condition:
          Bool:
            "aws:SecureTransport": "false"
```

### 3. IAM User (`FileShareUser`)

**Purpose**: Provides programmatic access with minimal required permissions.

```yaml
Type: AWS::IAM::User
Properties:
  UserName: !Ref IAMUserName
  Policies:
    - PolicyName: S3FileShareAccess
      PolicyDocument:
        Statement:
          - Effect: Allow
            Action:
              - s3:PutObject
              - s3:PutObjectAcl
              - s3:PutObjectTagging
              - s3:GetObject
              - s3:GetObjectAcl
              - s3:GetObjectTagging
              - s3:DeleteObject
              - s3:DeleteObjectTagging
              - s3:ListBucket
              - s3:GetBucketLocation
              - s3:GetBucketVersioning
              - s3:HeadObject
            Resource:
              - !Sub "${FileShareBucket}/*"
              - !Ref FileShareBucket
```

### 4. IAM Access Key (`FileShareAccessKey`)

**Purpose**: Provides programmatic credentials for the application.

```yaml
Type: AWS::IAM::AccessKey
Properties:
  UserName: !Ref FileShareUser
```

### 5. CloudTrail (`FileShareCloudTrail`)

**Purpose**: Audit logging for all S3 operations.

```yaml
Type: AWS::CloudTrail::Trail
Properties:
  TrailName: !Sub "${AWS::StackName}-trail"
  S3BucketName: !Ref AuditLogsBucket
  IncludeGlobalServiceEvents: false
  IsMultiRegionTrail: false
  EnableLogFileValidation: true
  EventSelectors:
    - ReadWriteType: All
      IncludeManagementEvents: false
      DataResources:
        - Type: "AWS::S3::Object"
          Values:
            - !Sub "${FileShareBucket}/*"
        - Type: "AWS::S3::Bucket"
          Values:
            - !Ref FileShareBucket
```

### 6. Audit Logs Bucket (`AuditLogsBucket`)

**Purpose**: Stores CloudTrail audit logs separately from application files.

```yaml
Type: AWS::S3::Bucket
Properties:
  BucketName: !Sub "${BucketName}-audit-logs"
  BucketEncryption:
    ServerSideEncryptionConfiguration:
      - ServerSideEncryptionByDefault:
          SSEAlgorithm: AES256
  PublicAccessBlockConfiguration:
    BlockPublicAcls: true
    BlockPublicPolicy: true
    IgnorePublicAcls: true
    RestrictPublicBuckets: true
  LifecycleConfiguration:
    Rules:
      - Id: DeleteAuditLogsAfter90Days
        Status: Enabled
        ExpirationInDays: 90
```

### 7. CloudWatch Log Group (`FileShareLogGroup`)

**Purpose**: Centralized logging for application and AWS service logs.

```yaml
Type: AWS::Logs::LogGroup
Properties:
  LogGroupName: !Sub "/aws/fileshare/${AWS::StackName}"
  RetentionInDays: !Ref LogRetentionDays
```

## Deployment Methods

### Method 1: Using Deployment Script (Recommended)

```bash
# Navigate to infrastructure directory
cd infrastructure

# Generate parameters
./scripts/generate-params.sh

# Deploy to default region (eu-central-1)
./scripts/deploy.sh

# Deploy to specific region
./scripts/deploy.sh us-east-1
```

### Method 2: AWS CLI Direct

```bash
# Validate template
aws cloudformation validate-template \
  --template-body file://cloudformation/file-sharing-app.yaml

# Deploy stack
aws cloudformation create-stack \
  --stack-name file-sharing-app \
  --template-body file://cloudformation/file-sharing-app.yaml \
  --parameters file://cloudformation/parameters.json \
  --capabilities CAPABILITY_IAM \
  --region us-east-1

# Wait for completion
aws cloudformation wait stack-create-complete \
  --stack-name file-sharing-app \
  --region us-east-1
```

### Method 3: AWS Console

1. Go to AWS Console → CloudFormation
2. Click "Create stack" → "With new resources"
3. Upload `file-sharing-app.yaml` template
4. Enter parameters manually
5. Review and create stack

### Method 4: AWS CDK (Advanced)

For programmatic deployment, you can convert the template to CDK:

```typescript
import * as cdk from 'aws-cdk-lib';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as iam from 'aws-cdk-lib/aws-iam';

export class FileShareStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);
    
    // Create S3 bucket
    const bucket = new s3.Bucket(this, 'FileShareBucket', {
      encryption: s3.BucketEncryption.S3_MANAGED,
      versioned: true,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      lifecycleRules: [
        {
          id: 'ExpireFiles1Hour',
          enabled: true,
          tagFilters: { expiration: '1hour' },
          expiration: cdk.Duration.days(1),
        },
        // Additional lifecycle rules...
      ],
    });
    
    // Create IAM user
    const user = new iam.User(this, 'FileShareUser');
    
    // Grant permissions
    bucket.grantReadWrite(user);
  }
}
```

## Stack Management

### Updating the Stack

```bash
# Update with same parameters
aws cloudformation update-stack \
  --stack-name file-sharing-app \
  --template-body file://cloudformation/file-sharing-app.yaml \
  --parameters file://cloudformation/parameters.json \
  --capabilities CAPABILITY_IAM

# Update with new parameters
aws cloudformation update-stack \
  --stack-name file-sharing-app \
  --template-body file://cloudformation/file-sharing-app.yaml \
  --parameters ParameterKey=Environment,ParameterValue=staging \
  --capabilities CAPABILITY_IAM
```

### Viewing Stack Information

```bash
# Get stack status
aws cloudformation describe-stacks --stack-name file-sharing-app

# Get stack outputs
aws cloudformation describe-stacks \
  --stack-name file-sharing-app \
  --query 'Stacks[0].Outputs'

# List stack resources
aws cloudformation list-stack-resources --stack-name file-sharing-app

# Get stack events
aws cloudformation describe-stack-events --stack-name file-sharing-app
```

### Deleting the Stack

```bash
# Empty S3 buckets first (required)
aws s3 rm s3://your-bucket-name --recursive
aws s3 rm s3://your-audit-bucket-name --recursive

# Delete stack
aws cloudformation delete-stack --stack-name file-sharing-app

# Wait for deletion
aws cloudformation wait stack-delete-complete --stack-name file-sharing-app
```

## Customization

### Environment-Specific Deployments

Create separate parameter files for different environments:

**parameters-dev.json**:
```json
[
  {
    "ParameterKey": "BucketName",
    "ParameterValue": "file-sharing-app-dev-123456789012"
  },
  {
    "ParameterKey": "Environment",
    "ParameterValue": "development"
  },
  {
    "ParameterKey": "LogRetentionDays",
    "ParameterValue": "7"
  }
]
```

**parameters-prod.json**:
```json
[
  {
    "ParameterKey": "BucketName",
    "ParameterValue": "file-sharing-app-prod-123456789012"
  },
  {
    "ParameterKey": "Environment",
    "ParameterValue": "production"
  },
  {
    "ParameterKey": "LogRetentionDays",
    "ParameterValue": "90"
  }
]
```

Deploy with specific parameters:
```bash
aws cloudformation create-stack \
  --stack-name file-sharing-app-dev \
  --template-body file://cloudformation/file-sharing-app.yaml \
  --parameters file://cloudformation/parameters-dev.json \
  --capabilities CAPABILITY_IAM
```

### Adding Custom Resources

To extend the template, add resources to the `Resources` section:

```yaml
# Add SNS topic for notifications
FileShareNotifications:
  Type: AWS::SNS::Topic
  Properties:
    TopicName: !Sub "${AWS::StackName}-notifications"
    DisplayName: "File Share Notifications"

# Add Lambda function for processing
FileProcessorFunction:
  Type: AWS::Lambda::Function
  Properties:
    FunctionName: !Sub "${AWS::StackName}-processor"
    Runtime: python3.9
    Handler: index.handler
    Code:
      ZipFile: |
        def handler(event, context):
            print("Processing file event:", event)
            return {"statusCode": 200}
    Role: !GetAtt FileProcessorRole.Arn
```

### Modifying Lifecycle Policies

Customize file expiration rules:

```yaml
LifecycleConfiguration:
  Rules:
    - Id: ExpireFiles2Hours
      Status: Enabled
      Filter:
        Tag:
          Key: expiration
          Value: 2hours
      ExpirationInDays: 1
    - Id: ExpireFiles6Months
      Status: Enabled
      Filter:
        Tag:
          Key: expiration
          Value: 6months
      ExpirationInDays: 180
```

### Adding Cross-Region Replication

For disaster recovery:

```yaml
# Destination bucket in another region
ReplicationDestinationBucket:
  Type: AWS::S3::Bucket
  Properties:
    BucketName: !Sub "${BucketName}-replica"
    Region: us-west-2

# Replication configuration
FileShareBucket:
  Type: AWS::S3::Bucket
  Properties:
    ReplicationConfiguration:
      Role: !GetAtt ReplicationRole.Arn
      Rules:
        - Id: ReplicateAll
          Status: Enabled
          Prefix: ""
          Destination:
            Bucket: !Sub "arn:aws:s3:::${ReplicationDestinationBucket}"
            StorageClass: STANDARD_IA
```

## Monitoring and Maintenance

### CloudWatch Metrics

Monitor key metrics:

```bash
# S3 bucket size
aws cloudwatch get-metric-statistics \
  --namespace AWS/S3 \
  --metric-name BucketSizeBytes \
  --dimensions Name=BucketName,Value=your-bucket-name Name=StorageType,Value=StandardStorage \
  --start-time 2023-01-01T00:00:00Z \
  --end-time 2023-01-02T00:00:00Z \
  --period 86400 \
  --statistics Average

# Number of objects
aws cloudwatch get-metric-statistics \
  --namespace AWS/S3 \
  --metric-name NumberOfObjects \
  --dimensions Name=BucketName,Value=your-bucket-name Name=StorageType,Value=AllStorageTypes \
  --start-time 2023-01-01T00:00:00Z \
  --end-time 2023-01-02T00:00:00Z \
  --period 86400 \
  --statistics Average
```

### Cost Monitoring

Set up billing alerts:

```yaml
# Add to CloudFormation template
BillingAlarm:
  Type: AWS::CloudWatch::Alarm
  Properties:
    AlarmName: !Sub "${AWS::StackName}-billing-alarm"
    AlarmDescription: "Alert when estimated charges exceed threshold"
    MetricName: EstimatedCharges
    Namespace: AWS/Billing
    Statistic: Maximum
    Period: 86400
    EvaluationPeriods: 1
    Threshold: 10
    ComparisonOperator: GreaterThanThreshold
    Dimensions:
      - Name: Currency
        Value: USD
    AlarmActions:
      - !Ref FileShareNotifications
```

### Automated Backups

Create automated backup of configuration:

```bash
#!/bin/bash
# backup-stack.sh

STACK_NAME="file-sharing-app"
BACKUP_DIR="backups/$(date +%Y-%m-%d)"

mkdir -p "$BACKUP_DIR"

# Export template
aws cloudformation get-template \
  --stack-name "$STACK_NAME" \
  --query 'TemplateBody' \
  > "$BACKUP_DIR/template.json"

# Export parameters
aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --query 'Stacks[0].Parameters' \
  > "$BACKUP_DIR/parameters.json"

# Export outputs
aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --query 'Stacks[0].Outputs' \
  > "$BACKUP_DIR/outputs.json"

echo "Backup saved to $BACKUP_DIR"
```

## Troubleshooting

### Common Deployment Issues

#### 1. Template Validation Errors

```bash
# Validate template syntax
aws cloudformation validate-template \
  --template-body file://cloudformation/file-sharing-app.yaml

# Common issues:
# - YAML indentation errors
# - Invalid resource references
# - Missing required properties
```

#### 2. Parameter Validation Errors

```bash
# Check parameter constraints
# - BucketName must be globally unique
# - IAMUserName must be unique in your account
# - Environment must be valid string
```

#### 3. Resource Creation Failures

```bash
# Check stack events for detailed error messages
aws cloudformation describe-stack-events \
  --stack-name file-sharing-app \
  --query 'StackEvents[?ResourceStatus==`CREATE_FAILED`]'

# Common failures:
# - S3 bucket name already exists
# - IAM user name already exists
# - Insufficient permissions
```

#### 4. Stack Rollback

```bash
# If stack creation fails, it will rollback
# Check events to see what failed
aws cloudformation describe-stack-events --stack-name file-sharing-app

# Delete failed stack and retry
aws cloudformation delete-stack --stack-name file-sharing-app
aws cloudformation wait stack-delete-complete --stack-name file-sharing-app

# Fix issues and redeploy
```

### Debugging Stack Updates

```bash
# Create change set to preview changes
aws cloudformation create-change-set \
  --stack-name file-sharing-app \
  --change-set-name preview-changes \
  --template-body file://cloudformation/file-sharing-app.yaml \
  --parameters file://cloudformation/parameters.json \
  --capabilities CAPABILITY_IAM

# Review changes
aws cloudformation describe-change-set \
  --stack-name file-sharing-app \
  --change-set-name preview-changes

# Execute if changes look good
aws cloudformation execute-change-set \
  --stack-name file-sharing-app \
  --change-set-name preview-changes
```

### Resource-Specific Issues

#### S3 Bucket Issues

```bash
# Check bucket exists and is accessible
aws s3 ls s3://your-bucket-name

# Check bucket policy
aws s3api get-bucket-policy --bucket your-bucket-name

# Check lifecycle configuration
aws s3api get-bucket-lifecycle-configuration --bucket your-bucket-name
```

#### IAM Issues

```bash
# Check user exists
aws iam get-user --user-name your-iam-user-name

# Check user policies
aws iam list-user-policies --user-name your-iam-user-name

# Test user permissions
aws sts get-caller-identity
```

#### CloudTrail Issues

```bash
# Check trail status
aws cloudtrail get-trail-status --name your-trail-name

# Check trail configuration
aws cloudtrail describe-trails --trail-name-list your-trail-name
```

### Recovery Procedures

#### Restore from Backup

```bash
# Restore stack from backup
aws cloudformation create-stack \
  --stack-name file-sharing-app-restored \
  --template-body file://backups/2023-01-01/template.json \
  --parameters file://backups/2023-01-01/parameters.json \
  --capabilities CAPABILITY_IAM
```

#### Disaster Recovery

```bash
# If primary region fails, deploy to secondary region
./scripts/deploy.sh us-west-2

# Update application configuration to use new region
# Restore data from S3 cross-region replication if configured
```

This comprehensive guide should help you understand, deploy, and manage the CloudFormation infrastructure for the File Sharing App. For additional support, refer to the AWS CloudFormation documentation and the main project README.