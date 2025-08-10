#!/bin/bash

echo "🚀 Starting File Sharing App"
echo "============================"
echo ""
echo "✅ AWS Region: eu-west-1"
echo "✅ S3 Bucket: file-sharing-app-bucket-YOUR_ACCOUNT_ID"
echo "✅ Credentials: Configure with your values"
echo ""
echo "🎯 Opening app window..."

# Start the app with AWS credentials
# TODO: Replace with your actual AWS credentials from infrastructure deployment
AWS_ACCESS_KEY_ID="AKIA...YOUR_ACCESS_KEY_ID" \
AWS_SECRET_ACCESS_KEY="YOUR_SECRET_ACCESS_KEY_HERE" \
AWS_DEFAULT_REGION="eu-west-1" \
S3_BUCKET_NAME="file-sharing-app-bucket-YOUR_ACCOUNT_ID" \
./file-sharing-app