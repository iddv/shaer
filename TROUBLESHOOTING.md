# Troubleshooting Guide

This guide helps you resolve common issues with the File Sharing App.

## Quick Diagnostics

If you're experiencing issues, first check:

1. **Application Logs**: Check the application logs for error messages
   - **Windows**: `%APPDATA%\file-sharing-app\logs\`
   - **macOS**: `~/Library/Application Support/file-sharing-app/logs/`
   - **Linux**: `~/.local/share/file-sharing-app/logs/`

2. **Network Connectivity**: Ensure you have internet access and can reach AWS services
3. **AWS Credentials**: Verify your AWS credentials are valid and have the correct permissions
4. **AWS Region**: Confirm you're using the correct AWS region where your S3 bucket is located

## Common Issues and Solutions

### 1. Application Won't Start

#### Symptoms
- Application crashes immediately on startup
- Error message about missing dependencies
- "Permission denied" errors

#### Solutions

**Windows:**
```powershell
# Run as administrator if needed
Right-click file-sharing-app.exe → "Run as administrator"

# Check Windows Defender/antivirus
Add file-sharing-app.exe to antivirus exclusions

# Install Visual C++ Redistributable if needed
Download from Microsoft website
```

**macOS:**
```bash
# Make executable
chmod +x file-sharing-app-darwin

# Remove quarantine attribute
xattr -d com.apple.quarantine file-sharing-app-darwin

# Allow in Security & Privacy
System Preferences → Security & Privacy → General → "Open Anyway"
```

**Linux:**
```bash
# Make executable
chmod +x file-sharing-app-linux

# Install required libraries (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install libgl1-mesa-glx libxrandr2 libxss1 libxcursor1 libxcomposite1 libasound2

# Install required libraries (CentOS/RHEL/Fedora)
sudo yum install mesa-libGL libXrandr libXss libXcursor libXcomposite alsa-lib
```

### 2. AWS Credential Issues

#### Symptoms
- "Invalid AWS credentials" error
- "Access denied" when uploading files
- "Unable to locate credentials" message

#### Solutions

**Check Credential Configuration:**
```bash
# Test AWS CLI access
aws sts get-caller-identity

# List S3 buckets to verify permissions
aws s3 ls
```

**Reconfigure Credentials:**
1. Open app settings and re-enter AWS credentials
2. Ensure Access Key ID and Secret Access Key are correct
3. Verify the IAM user has the required permissions:
   - `s3:PutObject`, `s3:GetObject`, `s3:DeleteObject`
   - `s3:ListBucket`, `s3:HeadObject`
   - `s3:PutObjectTagging`, `s3:GetObjectTagging`

**Clear Stored Credentials:**
- **Windows**: Open Credential Manager → Windows Credentials → Remove "file-sharing-app" entries
- **macOS**: Open Keychain Access → Search "file-sharing-app" → Delete entries
- **Linux**: Use `secret-tool clear service file-sharing-app`

### 3. File Upload Failures

#### Symptoms
- Upload progress stops or fails
- "Network error" during upload
- Files don't appear in the file list after upload

#### Solutions

**Check File Size:**
- Maximum file size is 100MB
- For larger files, consider splitting or using a different service

**Network Issues:**
```bash
# Test S3 connectivity
aws s3 ls s3://your-bucket-name

# Check DNS resolution
nslookup s3.amazonaws.com
ping s3.amazonaws.com
```

**Retry Upload:**
1. Delete the failed upload from the file list
2. Try uploading again
3. Check your internet connection stability
4. Try uploading a smaller test file first

**Firewall/Proxy Issues:**
- Ensure ports 80 and 443 are open for outbound connections
- Configure proxy settings if behind a corporate firewall
- Add AWS S3 endpoints to firewall allowlist

### 4. S3 Bucket Issues

#### Symptoms
- "Bucket does not exist" error
- "Access denied" when accessing bucket
- Files not being deleted after expiration

#### Solutions

**Verify Bucket Configuration:**
```bash
# Check if bucket exists
aws s3 ls s3://your-bucket-name

# Check bucket region
aws s3api get-bucket-location --bucket your-bucket-name

# Verify lifecycle policies
aws s3api get-bucket-lifecycle-configuration --bucket your-bucket-name
```

**Common Fixes:**
1. **Wrong Region**: Ensure app is configured with the correct AWS region
2. **Bucket Name**: Verify the exact bucket name (case-sensitive)
3. **Permissions**: Check IAM user has access to the specific bucket
4. **Lifecycle Policies**: Redeploy CloudFormation stack if lifecycle policies are missing

### 5. Sharing Link Issues

#### Symptoms
- Generated links don't work
- "Access denied" when recipients click links
- Links expire immediately

#### Solutions

**Check Link Generation:**
1. Verify file was uploaded successfully
2. Ensure file hasn't expired
3. Check that presigned URL expiration is set correctly

**Test Links:**
```bash
# Test the presigned URL with curl
curl -I "your-presigned-url"

# Should return HTTP 200 OK
```

**Common Issues:**
- **Clock Skew**: Ensure your system clock is accurate
- **URL Encoding**: Some email clients may break long URLs
- **Expiration**: Links expire based on file expiration settings

### 6. Database Issues

#### Symptoms
- App crashes when viewing file list
- "Database locked" errors
- Missing file history

#### Solutions

**Reset Local Database:**
1. Close the application completely
2. Delete the local database file:
   - **Windows**: `%APPDATA%\file-sharing-app\database.db`
   - **macOS**: `~/Library/Application Support/file-sharing-app/database.db`
   - **Linux**: `~/.local/share/file-sharing-app/database.db`
3. Restart the application (database will be recreated)
4. Note: This will clear your local file history

**Fix Database Permissions:**
```bash
# Linux/macOS
chmod 600 ~/.local/share/file-sharing-app/database.db

# Windows (PowerShell)
icacls "%APPDATA%\file-sharing-app\database.db" /grant:r "%USERNAME%:F"
```

### 7. UI/Display Issues

#### Symptoms
- Blank or corrupted display
- UI elements not responding
- Application window too small/large

#### Solutions

**Reset UI Settings:**
1. Close the application
2. Delete UI configuration:
   - **Windows**: `%APPDATA%\file-sharing-app\ui-config.json`
   - **macOS**: `~/Library/Application Support/file-sharing-app/ui-config.json`
   - **Linux**: `~/.local/share/file-sharing-app/ui-config.json`
3. Restart the application

**Display Issues:**
- Try running on a different monitor
- Check display scaling settings (Windows: 100% recommended)
- Update graphics drivers

### 8. Performance Issues

#### Symptoms
- Slow file uploads
- Application freezes during operations
- High memory usage

#### Solutions

**Optimize Upload Performance:**
1. Close other applications using network bandwidth
2. Try uploading smaller files first
3. Check available disk space
4. Restart the application periodically

**Memory Issues:**
- Restart the application if it's been running for a long time
- Upload files one at a time instead of multiple simultaneously
- Check system memory usage

## Advanced Troubleshooting

### Enable Debug Logging

Set environment variable before starting the app:
```bash
# Enable debug logging
export LOG_LEVEL=debug

# Start the application
./file-sharing-app
```

### Check AWS CloudTrail

If you have CloudTrail enabled, check for API call logs:
1. Go to AWS Console → CloudTrail
2. Look for events related to your S3 bucket
3. Check for error codes and failure reasons

### Network Diagnostics

```bash
# Test AWS connectivity
curl -I https://s3.amazonaws.com

# Test specific region
curl -I https://s3.us-east-1.amazonaws.com

# Check DNS resolution
nslookup s3.amazonaws.com
```

### Reinstall Application

If all else fails:
1. Close the application completely
2. Delete application data directory:
   - **Windows**: `%APPDATA%\file-sharing-app\`
   - **macOS**: `~/Library/Application Support/file-sharing-app/`
   - **Linux**: `~/.local/share/file-sharing-app/`
3. Download and install the latest version
4. Reconfigure AWS credentials and settings

## Getting Help

If you're still experiencing issues:

1. **Check Application Logs**: Include relevant log entries when reporting issues
2. **Gather System Information**: OS version, application version, AWS region
3. **Document Steps**: What you were doing when the issue occurred
4. **Test Environment**: Try with a fresh AWS account/bucket if possible

### Log Locations

- **Windows**: `%APPDATA%\file-sharing-app\logs\app.log`
- **macOS**: `~/Library/Application Support/file-sharing-app/logs/app.log`
- **Linux**: `~/.local/share/file-sharing-app/logs/app.log`

### System Information

Include this information when reporting issues:
- Operating system and version
- Application version
- AWS region
- Error messages (exact text)
- Steps to reproduce the issue

## Prevention Tips

1. **Regular Updates**: Keep the application updated to the latest version
2. **Credential Rotation**: Rotate AWS access keys regularly
3. **Monitor Usage**: Check AWS billing for unexpected charges
4. **Backup Settings**: Note down your AWS configuration settings
5. **Test Regularly**: Periodically test file uploads and sharing to ensure everything works

## Security Considerations

- Never share your AWS credentials with others
- Use IAM users with minimal required permissions
- Monitor CloudTrail logs for suspicious activity
- Rotate access keys if you suspect they've been compromised
- Keep the application updated for security patches