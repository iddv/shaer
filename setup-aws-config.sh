#!/bin/bash

# File Sharing App - AWS Configuration Setup
# This script creates AWS credential files for persistent configuration

echo "🔧 Setting up AWS Configuration Files"
echo "====================================="

# Create AWS directory if it doesn't exist
mkdir -p ~/.aws

# Create credentials file
cat > ~/.aws/credentials << EOF
[default]
aws_access_key_id = AKIA...YOUR_ACCESS_KEY_ID
aws_secret_access_key = YOUR_SECRET_ACCESS_KEY_HERE
EOF

# Create config file
cat > ~/.aws/config << EOF
[default]
region = eu-west-1
output = json
EOF

# Set proper permissions
chmod 600 ~/.aws/credentials
chmod 600 ~/.aws/config

echo "✅ AWS credentials file created: ~/.aws/credentials"
echo "✅ AWS config file created: ~/.aws/config"
echo "✅ Permissions set to 600 (secure)"
echo ""
echo "🎯 You can now start the app with: ./file-sharing-app"
echo "   Or use the run-app.sh script"