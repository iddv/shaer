# GitHub Actions Setup for AWS Deployment and Integration Testing

This document describes how to configure GitHub Actions secrets and environment variables for automated AWS infrastructure deployment and integration testing.

## Required GitHub Secrets

To enable the deployment workflow, you need to configure the following secrets in your GitHub repository:

### AWS Credentials

1. **AWS_ACCESS_KEY_ID**
   - Description: AWS Access Key ID for deployment
   - Value: Your AWS access key ID (e.g., `AKIAIOSFODNN7EXAMPLE`)
   - Required for: CloudFormation deployment and integration testing

2. **AWS_SECRET_ACCESS_KEY**
   - Description: AWS Secret Access Key for deployment
   - Value: Your AWS secret access key
   - Required for: CloudFormation deployment and integration testing

### Setting Up GitHub Secrets

1. Navigate to your GitHub repository
2. Go to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add each secret with the name and value as specified above

## AWS IAM Setup

### Required IAM Permissions

The AWS credentials used for GitHub Actions need the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "cloudformation:CreateStack",
                "cloudformation:UpdateStack",
                "cloudformation:DeleteStack",
                "cloudformation:DescribeStacks",
                "cloudformation:DescribeStackEvents",
                "cloudformation:DescribeStackResources",
                "cloudformation:ValidateTemplate",
                "cloudformation:GetTemplate"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:CreateBucket",
                "s3:DeleteBucket",
                "s3:GetBucketLocation",
                "s3:GetBucketVersioning",
                "s3:PutBucketEncryption",
                "s3:PutBucketVersioning",
                "s3:PutBucketPublicAccessBlock",
                "s3:PutBucketLifecycleConfiguration",
                "s3:PutBucketNotification",
                "s3:PutBucketPolicy",
                "s3:GetBucketPolicy",
                "s3:DeleteBucketPolicy"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:PutObject",
                "s3:GetObject",
                "s3:DeleteObject",
                "s3:HeadObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::file-sharing-app-*",
                "arn:aws:s3:::file-sharing-app-*/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "iam:CreateUser",
                "iam:DeleteUser",
                "iam:CreateAccessKey",
                "iam:DeleteAccessKey",
                "iam:PutUserPolicy",
                "iam:DeleteUserPolicy",
                "iam:GetUser",
                "iam:ListAccessKeys",
                "iam:TagUser"
            ],
            "Resource": "arn:aws:iam::*:user/file-sharing-app-*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:DeleteLogGroup",
                "logs:PutRetentionPolicy",
                "logs:TagLogGroup"
            ],
            "Resource": "arn:aws:logs:*:*:log-group:/aws/s3/*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "cloudtrail:CreateTrail",
                "cloudtrail:DeleteTrail",
                "cloudtrail:PutEventSelectors",
                "cloudtrail:StartLogging",
                "cloudtrail:StopLogging",
                "cloudtrail:GetTrailStatus"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        }
    ]
}
```

### Creating IAM User for GitHub Actions

1. **Create IAM User:**
   ```bash
   aws iam create-user --user-name github-actions-file-sharing-app
   ```

2. **Create and attach policy:**
   ```bash
   # Save the above policy to a file named github-actions-policy.json
   aws iam put-user-policy \
     --user-name github-actions-file-sharing-app \
     --policy-name GitHubActionsDeploymentPolicy \
     --policy-document file://github-actions-policy.json
   ```

3. **Create access key:**
   ```bash
   aws iam create-access-key --user-name github-actions-file-sharing-app
   ```

4. **Save the access key ID and secret access key** to use as GitHub secrets.

## Workflow Configuration

### Deployment Workflow

The deployment workflow (`.github/workflows/deploy.yml`) supports:

- **Automatic deployment** on push to `main` branch when infrastructure files change
- **Manual deployment** via workflow dispatch with environment and region selection
- **Pull request validation** for infrastructure changes

### Workflow Inputs

When manually triggering the deployment workflow, you can specify:

- **Environment**: `dev` or `prod`
- **Region**: `us-west-2`, `us-east-1`, or `eu-central-1`

### Environment-Specific Configuration

The workflow automatically generates unique resource names based on:
- Environment (dev/prod)
- Timestamp
- Random number
- AWS region

Example generated names:
- Bucket: `file-sharing-app-dev-1640995200-1234-us-west-2`
- IAM User: `file-sharing-app-dev-user-1640995200-1234`

## Integration Testing

### Test Execution

Integration tests run automatically after successful infrastructure deployment and include:

1. **AWS Integration Tests** (`test/integration/`)
   - S3 service functionality
   - File upload/download workflows
   - Credential validation

2. **End-to-End Tests** (`test/e2e/`)
   - Complete user workflows
   - Application controller integration
   - Multi-file operations

3. **Security Tests** (`test/security/`)
   - Credential security validation
   - Presigned URL security
   - S3 bucket security configuration

### Test Environment Variables

The following environment variables are automatically set for integration tests:

- `S3_BUCKET`: Generated bucket name from deployment
- `AWS_REGION`: Deployment region
- `AWS_ACCESS_KEY_ID`: Generated access key from deployment
- `AWS_SECRET_ACCESS_KEY`: Generated secret key from deployment

## Cleanup

### Development Environment Cleanup

For development deployments, the workflow automatically cleans up resources if tests fail:

1. Deletes the CloudFormation stack
2. Removes all associated resources (S3 buckets, IAM users, etc.)

### Manual Cleanup

To manually clean up resources:

```bash
# Delete CloudFormation stack
aws cloudformation delete-stack --stack-name file-sharing-app-dev --region us-west-2

# Wait for deletion to complete
aws cloudformation wait stack-delete-complete --stack-name file-sharing-app-dev --region us-west-2
```

## Security Considerations

1. **Least Privilege**: The IAM policy provides minimal required permissions
2. **Resource Scoping**: Permissions are scoped to specific resource patterns
3. **Temporary Credentials**: Generated credentials are only used for testing
4. **Automatic Cleanup**: Development resources are automatically cleaned up
5. **Secret Management**: Sensitive values are stored as GitHub secrets

## Troubleshooting

### Common Issues

1. **Permission Denied Errors**
   - Verify IAM policy includes all required permissions
   - Check resource ARN patterns match your account

2. **Stack Creation Failures**
   - Check CloudFormation events in AWS Console
   - Verify unique resource names (bucket names must be globally unique)

3. **Integration Test Failures**
   - Verify AWS credentials are valid
   - Check S3 bucket permissions
   - Ensure network connectivity to AWS services

### Debug Information

The workflow provides detailed logging including:
- CloudFormation stack events
- AWS resource creation status
- Integration test results
- Security validation outcomes

### Getting Help

If you encounter issues:

1. Check the GitHub Actions logs for detailed error messages
2. Review AWS CloudFormation events in the AWS Console
3. Verify IAM permissions and resource limits
4. Check AWS service status for any ongoing issues