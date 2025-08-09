#!/bin/bash

# File Sharing App - CloudFormation Deployment Script
# Usage: ./deploy.sh [region]
# Example: ./deploy.sh eu-central-1

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
REGION=${1:-eu-central-1}
STACK_NAME="file-sharing-app"
TEMPLATE_FILE="infrastructure/cloudformation/file-sharing-app.yaml"
PARAMETERS_FILE="infrastructure/cloudformation/parameters.json"

echo -e "${GREEN}File Sharing App - CloudFormation Deployment${NC}"
echo "Region: ${REGION}"
echo "Stack Name: ${STACK_NAME}"
echo ""

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo -e "${RED}Error: AWS CLI is not installed${NC}"
    echo "Please install AWS CLI: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"
    exit 1
fi

# Check if AWS credentials are configured
if ! aws sts get-caller-identity &> /dev/null; then
    echo -e "${RED}Error: AWS credentials not configured${NC}"
    echo "Please configure AWS credentials using 'aws configure' or environment variables"
    exit 1
fi

# Check if template file exists
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo -e "${RED}Error: Template file not found: $TEMPLATE_FILE${NC}"
    exit 1
fi

# Check if parameters file exists
if [ ! -f "$PARAMETERS_FILE" ]; then
    echo -e "${RED}Error: Parameters file not found: $PARAMETERS_FILE${NC}"
    exit 1
fi

# Validate CloudFormation template
echo -e "${YELLOW}Validating CloudFormation template...${NC}"
aws cloudformation validate-template \
    --template-body file://$TEMPLATE_FILE \
    --region $REGION

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Template validation successful${NC}"
else
    echo -e "${RED}Template validation failed${NC}"
    exit 1
fi

# Check if stack exists
STACK_EXISTS=$(aws cloudformation describe-stacks \
    --stack-name $STACK_NAME \
    --region $REGION \
    --query 'Stacks[0].StackStatus' \
    --output text 2>/dev/null || echo "DOES_NOT_EXIST")

if [ "$STACK_EXISTS" = "DOES_NOT_EXIST" ]; then
    echo -e "${YELLOW}Creating new stack: $STACK_NAME${NC}"
    OPERATION="create-stack"
else
    echo -e "${YELLOW}Updating existing stack: $STACK_NAME${NC}"
    OPERATION="update-stack"
fi

# Deploy the stack
echo -e "${YELLOW}Deploying CloudFormation stack...${NC}"
aws cloudformation $OPERATION \
    --stack-name $STACK_NAME \
    --template-body file://$TEMPLATE_FILE \
    --parameters file://$PARAMETERS_FILE \
    --capabilities CAPABILITY_IAM \
    --region $REGION \
    --tags Key=Application,Value=file-sharing-app Key=Environment,Value=production

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Stack deployment initiated successfully${NC}"
else
    echo -e "${RED}Stack deployment failed${NC}"
    exit 1
fi

# Wait for stack operation to complete
echo -e "${YELLOW}Waiting for stack operation to complete...${NC}"
if [ "$OPERATION" = "create-stack" ]; then
    aws cloudformation wait stack-create-complete \
        --stack-name $STACK_NAME \
        --region $REGION
else
    aws cloudformation wait stack-update-complete \
        --stack-name $STACK_NAME \
        --region $REGION
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Stack operation completed successfully${NC}"
else
    echo -e "${RED}Stack operation failed or timed out${NC}"
    echo "Check the CloudFormation console for details"
    exit 1
fi

# Display stack outputs
echo -e "${GREEN}Stack Outputs:${NC}"
aws cloudformation describe-stacks \
    --stack-name $STACK_NAME \
    --region $REGION \
    --query 'Stacks[0].Outputs[*].[OutputKey,OutputValue]' \
    --output table

echo ""
echo -e "${GREEN}Deployment completed successfully!${NC}"
echo ""
echo -e "${YELLOW}Important Security Notes:${NC}"
echo "1. The Secret Access Key is displayed above - store it securely"
echo "2. Consider using AWS Secrets Manager or Parameter Store for production"
echo "3. Rotate access keys regularly"
echo "4. Monitor CloudTrail logs for security events"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "1. Configure the application with the AWS credentials from the outputs"
echo "2. Test file upload and sharing functionality"
echo "3. Monitor S3 lifecycle policies for automatic cleanup"