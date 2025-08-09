#!/bin/bash

# Generate parameter file for production deployment
# Usage: ./generate-params.sh [account-id]

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get AWS account ID automatically or use provided one
if [ -n "$1" ]; then
    ACCOUNT_ID="$1"
else
    ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null || echo "12345")
fi

FILENAME="infrastructure/cloudformation/parameters.json"

echo -e "${GREEN}Generating parameter file for File Sharing App${NC}"
echo "Account ID: ${ACCOUNT_ID}"
echo ""

echo -e "${YELLOW}Creating: ${FILENAME}${NC}"

cat > "$FILENAME" << EOF
[
  {
    "ParameterKey": "BucketName",
    "ParameterValue": "file-sharing-app-bucket-${ACCOUNT_ID}"
  },
  {
    "ParameterKey": "IAMUserName",
    "ParameterValue": "file-sharing-app-user-${ACCOUNT_ID}"
  },
  {
    "ParameterKey": "Environment",
    "ParameterValue": "production"
  }
]
EOF

echo ""
echo -e "${GREEN}Parameter file generated successfully!${NC}"
echo "Generated file: ${FILENAME}"