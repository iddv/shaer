package aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	KeyringServiceName = "file-sharing-app"
	AccessKeyItem      = "aws-access-key"
	SecretKeyItem      = "aws-secret-key"
	RegionItem         = "aws-region"
)

// CredentialProvider defines the interface for managing AWS credentials
type CredentialProvider interface {
	GetCredentials(ctx context.Context) (aws.Credentials, error)
	StoreCredentials(accessKey, secretKey, region string) error
	ValidateCredentials(ctx context.Context) error
	ClearCredentials() error
	GetRegion() (string, error)
	SetRegion(region string) error
}

// SecureCredentialProvider implements CredentialProvider using OS keychain
type SecureCredentialProvider struct {
	keyring keyring.Keyring
}

// NewSecureCredentialProvider creates a new SecureCredentialProvider
func NewSecureCredentialProvider() (*SecureCredentialProvider, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: KeyringServiceName,
		// Allow fallback to file backend for testing
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.SecretServiceBackend,
			keyring.WinCredBackend,
			keyring.FileBackend,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	return &SecureCredentialProvider{
		keyring: ring,
	}, nil
}

// StoreCredentials stores AWS credentials securely in the OS keychain
func (p *SecureCredentialProvider) StoreCredentials(accessKey, secretKey, region string) error {
	if accessKey == "" || secretKey == "" {
		return errors.New("access key and secret key cannot be empty")
	}

	// Store access key
	if err := p.keyring.Set(keyring.Item{
		Key:  AccessKeyItem,
		Data: []byte(accessKey),
	}); err != nil {
		return fmt.Errorf("failed to store access key: %w", err)
	}

	// Store secret key
	if err := p.keyring.Set(keyring.Item{
		Key:  SecretKeyItem,
		Data: []byte(secretKey),
	}); err != nil {
		return fmt.Errorf("failed to store secret key: %w", err)
	}

	// Store region if provided
	if region != "" {
		if err := p.SetRegion(region); err != nil {
			return fmt.Errorf("failed to store region: %w", err)
		}
	}

	return nil
}

// GetCredentials retrieves AWS credentials from the keychain
func (p *SecureCredentialProvider) GetCredentials(ctx context.Context) (aws.Credentials, error) {
	// Try to get credentials from keychain first
	accessKeyItem, err := p.keyring.Get(AccessKeyItem)
	if err != nil {
		// If not found in keychain, try AWS credential chain
		return p.getCredentialsFromChain(ctx)
	}

	secretKeyItem, err := p.keyring.Get(SecretKeyItem)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to retrieve secret key from keychain: %w", err)
	}

	return aws.Credentials{
		AccessKeyID:     string(accessKeyItem.Data),
		SecretAccessKey: string(secretKeyItem.Data),
		Source:          "file-sharing-app-keychain",
	}, nil
}

// getCredentialsFromChain attempts to get credentials using AWS credential chain
func (p *SecureCredentialProvider) getCredentialsFromChain(ctx context.Context) (aws.Credentials, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to retrieve credentials from AWS credential chain: %w", err)
	}

	return creds, nil
}

// ValidateCredentials validates the stored credentials by making a test AWS API call
func (p *SecureCredentialProvider) ValidateCredentials(ctx context.Context) error {
	creds, err := p.GetCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Get region for the STS client
	region, err := p.GetRegion()
	if err != nil {
		region = "us-east-1" // Default region
	}

	// Create AWS config with the credentials
	cfg := aws.Config{
		Credentials: credentials.StaticCredentialsProvider{
			Value: creds,
		},
		Region: region,
	}

	// Create STS client and make a test call
	stsClient := sts.NewFromConfig(cfg)
	
	// Set a timeout for the validation call
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	return nil
}

// ClearCredentials removes all stored credentials from the keychain
func (p *SecureCredentialProvider) ClearCredentials() error {
	// Remove access key - ignore if not found
	_ = p.keyring.Remove(AccessKeyItem)

	// Remove secret key - ignore if not found
	_ = p.keyring.Remove(SecretKeyItem)

	// Remove region - ignore if not found
	_ = p.keyring.Remove(RegionItem)

	return nil
}

// GetRegion retrieves the stored AWS region
func (p *SecureCredentialProvider) GetRegion() (string, error) {
	item, err := p.keyring.Get(RegionItem)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "us-east-1", nil // Default region
		}
		return "", fmt.Errorf("failed to retrieve region: %w", err)
	}

	return string(item.Data), nil
}

// SetRegion stores the AWS region
func (p *SecureCredentialProvider) SetRegion(region string) error {
	if region == "" {
		return errors.New("region cannot be empty")
	}

	if err := p.keyring.Set(keyring.Item{
		Key:  RegionItem,
		Data: []byte(region),
	}); err != nil {
		return fmt.Errorf("failed to store region: %w", err)
	}

	return nil
}

// GetSetupGuidance provides user-friendly guidance for setting up AWS credentials
func GetSetupGuidance() string {
	return `AWS Credentials Setup Guide:

1. Create an AWS Account:
   - Visit https://aws.amazon.com and create an account if you don't have one

2. Create an IAM User:
   - Go to AWS Console > IAM > Users > Create User
   - Choose "Programmatic access" for access type
   - Attach the following policy (or create a custom policy):
     {
       "Version": "2012-10-17",
       "Statement": [
         {
           "Effect": "Allow",
           "Action": [
             "s3:PutObject",
             "s3:GetObject",
             "s3:DeleteObject"
           ],
           "Resource": "arn:aws:s3:::your-bucket-name/*"
         }
       ]
     }

3. Get Your Credentials:
   - After creating the user, download the CSV file with your credentials
   - You'll need the Access Key ID and Secret Access Key

4. Set Up S3 Bucket:
   - Create an S3 bucket in your preferred region
   - Configure bucket policies for secure access
   - Set up lifecycle policies for automatic file cleanup

5. Configure the Application:
   - Use the settings dialog to enter your credentials
   - The app will securely store them in your OS keychain
   - Test the connection to ensure everything works

Security Best Practices:
- Never share your AWS credentials
- Use IAM policies with minimal required permissions
- Regularly rotate your access keys
- Monitor your AWS usage and costs`
}