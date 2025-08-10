#!/bin/bash

# Deployment Validation Script
# This script validates that the AWS infrastructure deployment and integration testing setup is working correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}File Sharing App - Deployment Validation${NC}"
echo "========================================"
echo ""

# Function to print status
print_status() {
    local status=$1
    local message=$2
    if [ "$status" = "OK" ]; then
        echo -e "${GREEN}✓${NC} $message"
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠${NC} $message"
    else
        echo -e "${RED}✗${NC} $message"
    fi
}

# Function to check command exists
check_command() {
    local cmd=$1
    local name=$2
    if command -v "$cmd" &> /dev/null; then
        print_status "OK" "$name is installed"
        return 0
    else
        print_status "ERROR" "$name is not installed"
        return 1
    fi
}

# Function to check environment variable
check_env_var() {
    local var=$1
    local required=$2
    if [ -n "${!var}" ]; then
        if [ "$var" = "AWS_SECRET_ACCESS_KEY" ]; then
            print_status "OK" "$var is set (hidden)"
        else
            print_status "OK" "$var is set: ${!var}"
        fi
        return 0
    else
        if [ "$required" = "true" ]; then
            print_status "ERROR" "$var is not set (required)"
            return 1
        else
            print_status "WARN" "$var is not set (optional)"
            return 0
        fi
    fi
}

# Function to validate file exists
check_file() {
    local file=$1
    local description=$2
    if [ -f "$file" ]; then
        print_status "OK" "$description exists: $file"
        return 0
    else
        print_status "ERROR" "$description not found: $file"
        return 1
    fi
}

# Function to validate directory exists
check_directory() {
    local dir=$1
    local description=$2
    if [ -d "$dir" ]; then
        print_status "OK" "$description exists: $dir"
        return 0
    else
        print_status "ERROR" "$description not found: $dir"
        return 1
    fi
}

VALIDATION_ERRORS=0

echo -e "${YELLOW}1. Checking Prerequisites${NC}"
echo "----------------------------"

# Check required commands
check_command "go" "Go" || ((VALIDATION_ERRORS++))
check_command "aws" "AWS CLI" || ((VALIDATION_ERRORS++))
check_command "git" "Git" || ((VALIDATION_ERRORS++))

# Check Go version
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    if [[ "$GO_VERSION" > "1.20" ]] || [[ "$GO_VERSION" == "1.20"* ]]; then
        print_status "OK" "Go version is compatible: $GO_VERSION"
    else
        print_status "WARN" "Go version may be too old: $GO_VERSION (recommended: 1.20+)"
    fi
fi

echo ""

echo -e "${YELLOW}2. Checking Project Structure${NC}"
echo "--------------------------------"

# Check key project files
check_file "go.mod" "Go module file" || ((VALIDATION_ERRORS++))
check_file "cmd/main.go" "Main application file" || ((VALIDATION_ERRORS++))
check_file "infrastructure/cloudformation/file-sharing-app.yaml" "CloudFormation template" || ((VALIDATION_ERRORS++))
check_file "infrastructure/scripts/deploy.sh" "Deployment script" || ((VALIDATION_ERRORS++))
check_file ".github/workflows/deploy.yml" "GitHub Actions deployment workflow" || ((VALIDATION_ERRORS++))
check_file "Makefile" "Makefile" || ((VALIDATION_ERRORS++))

# Check test directories
check_directory "test/integration" "Integration test directory" || ((VALIDATION_ERRORS++))
check_directory "test/e2e" "End-to-end test directory" || ((VALIDATION_ERRORS++))
check_directory "test/security" "Security test directory" || ((VALIDATION_ERRORS++))

# Check test files
check_file "test/integration/aws_integration_test.go" "AWS integration tests" || ((VALIDATION_ERRORS++))
check_file "test/e2e/end_to_end_test.go" "End-to-end tests" || ((VALIDATION_ERRORS++))
check_file "test/security/security_validation_test.go" "Security tests" || ((VALIDATION_ERRORS++))
check_file "test/run_integration_tests.sh" "Integration test runner" || ((VALIDATION_ERRORS++))

echo ""

echo -e "${YELLOW}3. Checking Environment Configuration${NC}"
echo "---------------------------------------"

# Check AWS credentials
if aws sts get-caller-identity &> /dev/null; then
    print_status "OK" "AWS credentials are valid"
    AWS_ACCOUNT=$(aws sts get-caller-identity --query Account --output text 2>/dev/null)
    AWS_USER=$(aws sts get-caller-identity --query Arn --output text 2>/dev/null | cut -d'/' -f2)
    print_status "OK" "AWS Account: $AWS_ACCOUNT"
    print_status "OK" "AWS User/Role: $AWS_USER"
else
    print_status "ERROR" "AWS credentials are not configured or invalid"
    ((VALIDATION_ERRORS++))
fi

# Check environment variables for integration testing
echo ""
echo "Integration Test Environment Variables:"
check_env_var "S3_BUCKET" "false"
check_env_var "AWS_REGION" "false"
check_env_var "AWS_ACCESS_KEY_ID" "false"
check_env_var "AWS_SECRET_ACCESS_KEY" "false"

echo ""

echo -e "${YELLOW}4. Validating CloudFormation Template${NC}"
echo "----------------------------------------"

if command -v aws &> /dev/null && aws sts get-caller-identity &> /dev/null; then
    if aws cloudformation validate-template --template-body file://infrastructure/cloudformation/file-sharing-app.yaml &> /dev/null; then
        print_status "OK" "CloudFormation template is valid"
    else
        print_status "ERROR" "CloudFormation template validation failed"
        ((VALIDATION_ERRORS++))
    fi
else
    print_status "WARN" "Cannot validate CloudFormation template (AWS CLI not configured)"
fi

echo ""

echo -e "${YELLOW}5. Checking Build System${NC}"
echo "----------------------------"

# Check if project builds
if go build -o /tmp/file-sharing-app-test ./cmd/main.go &> /dev/null; then
    print_status "OK" "Project builds successfully"
    rm -f /tmp/file-sharing-app-test
else
    print_status "ERROR" "Project build failed"
    ((VALIDATION_ERRORS++))
fi

# Check if tests compile
if go test -c -tags=integration ./test/integration/... &> /dev/null; then
    print_status "OK" "Integration tests compile successfully"
    rm -f integration.test
else
    print_status "ERROR" "Integration tests compilation failed"
    ((VALIDATION_ERRORS++))
fi

if go test -c -tags=e2e ./test/e2e/... &> /dev/null; then
    print_status "OK" "E2E tests compile successfully"
    rm -f e2e.test
else
    print_status "ERROR" "E2E tests compilation failed"
    ((VALIDATION_ERRORS++))
fi

if go test -c -tags=security ./test/security/... &> /dev/null; then
    print_status "OK" "Security tests compile successfully"
    rm -f security.test
else
    print_status "ERROR" "Security tests compilation failed"
    ((VALIDATION_ERRORS++))
fi

echo ""

echo -e "${YELLOW}6. Checking File Permissions${NC}"
echo "--------------------------------"

# Check script permissions
if [ -x "infrastructure/scripts/deploy.sh" ]; then
    print_status "OK" "Deployment script is executable"
else
    print_status "WARN" "Deployment script is not executable (run: chmod +x infrastructure/scripts/deploy.sh)"
fi

if [ -x "test/run_integration_tests.sh" ]; then
    print_status "OK" "Integration test runner is executable"
else
    print_status "WARN" "Integration test runner is not executable (run: chmod +x test/run_integration_tests.sh)"
fi

echo ""

echo -e "${YELLOW}7. GitHub Actions Validation${NC}"
echo "--------------------------------"

# Check GitHub Actions workflow syntax
if command -v yamllint &> /dev/null; then
    if yamllint .github/workflows/deploy.yml &> /dev/null; then
        print_status "OK" "GitHub Actions workflow syntax is valid"
    else
        print_status "WARN" "GitHub Actions workflow has syntax issues"
    fi
else
    print_status "WARN" "yamllint not available, cannot validate workflow syntax"
fi

# Check for required GitHub secrets documentation
if [ -f "docs/github-actions-setup.md" ]; then
    print_status "OK" "GitHub Actions setup documentation exists"
else
    print_status "WARN" "GitHub Actions setup documentation not found"
fi

echo ""

echo -e "${YELLOW}Summary${NC}"
echo "-------"

if [ $VALIDATION_ERRORS -eq 0 ]; then
    echo -e "${GREEN}✓ All validations passed!${NC}"
    echo ""
    echo -e "${BLUE}Next Steps:${NC}"
    echo "1. Set up GitHub repository secrets (see docs/github-actions-setup.md)"
    echo "2. Deploy AWS infrastructure: make deploy-dev"
    echo "3. Run integration tests: make test-integration"
    echo "4. Push to GitHub to trigger automated deployment"
    echo ""
    echo -e "${GREEN}Your deployment setup is ready!${NC}"
    exit 0
else
    echo -e "${RED}✗ $VALIDATION_ERRORS validation error(s) found${NC}"
    echo ""
    echo -e "${YELLOW}Please fix the errors above before proceeding with deployment.${NC}"
    exit 1
fi