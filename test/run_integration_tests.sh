#!/bin/bash

# Integration Test Runner for File Sharing App
# This script runs all integration tests with proper environment setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}File Sharing App - Integration Test Runner${NC}"
echo ""

# Check required environment variables
REQUIRED_VARS=("S3_BUCKET" "AWS_REGION" "AWS_ACCESS_KEY_ID" "AWS_SECRET_ACCESS_KEY")
MISSING_VARS=()

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        MISSING_VARS+=("$var")
    fi
done

if [ ${#MISSING_VARS[@]} -ne 0 ]; then
    echo -e "${RED}Error: Missing required environment variables:${NC}"
    for var in "${MISSING_VARS[@]}"; do
        echo "  - $var"
    done
    echo ""
    echo "Please set these environment variables before running integration tests."
    echo "Example:"
    echo "  export S3_BUCKET=your-test-bucket"
    echo "  export AWS_REGION=us-west-2"
    echo "  export AWS_ACCESS_KEY_ID=your-access-key"
    echo "  export AWS_SECRET_ACCESS_KEY=your-secret-key"
    exit 1
fi

echo -e "${YELLOW}Environment Configuration:${NC}"
echo "  S3_BUCKET: $S3_BUCKET"
echo "  AWS_REGION: $AWS_REGION"
echo "  AWS_ACCESS_KEY_ID: ${AWS_ACCESS_KEY_ID:0:8}..."
echo "  AWS_SECRET_ACCESS_KEY: [HIDDEN]"
echo ""

# Verify AWS credentials work
echo -e "${YELLOW}Verifying AWS credentials...${NC}"
if ! aws sts get-caller-identity >/dev/null 2>&1; then
    echo -e "${RED}Error: AWS credentials are not valid or AWS CLI is not configured${NC}"
    exit 1
fi
echo -e "${GREEN}AWS credentials verified${NC}"
echo ""

# Build the application first
echo -e "${YELLOW}Building application...${NC}"
go build -o file-sharing-app ./cmd/main.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Application built successfully${NC}"
else
    echo -e "${RED}Application build failed${NC}"
    exit 1
fi
echo ""

# Run different test suites
TEST_SUITES=("integration" "e2e" "security")
FAILED_SUITES=()

for suite in "${TEST_SUITES[@]}"; do
    echo -e "${YELLOW}Running $suite tests...${NC}"
    
    if go test -v -tags="$suite" "./test/$suite/..."; then
        echo -e "${GREEN}$suite tests passed${NC}"
    else
        echo -e "${RED}$suite tests failed${NC}"
        FAILED_SUITES+=("$suite")
    fi
    echo ""
done

# Summary
echo -e "${YELLOW}Test Summary:${NC}"
if [ ${#FAILED_SUITES[@]} -eq 0 ]; then
    echo -e "${GREEN}All test suites passed!${NC}"
    exit 0
else
    echo -e "${RED}Failed test suites:${NC}"
    for suite in "${FAILED_SUITES[@]}"; do
        echo "  - $suite"
    done
    exit 1
fi