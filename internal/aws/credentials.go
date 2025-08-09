package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
)

// CredentialProvider interface defines credential management operations
type CredentialProvider interface {
	GetCredentials() (aws.Credentials, error)
	RefreshCredentials() error
	ValidateCredentials() error
	StoreCredentials(accessKey, secretKey string) error
}

// SecureCredentialProvider implements CredentialProvider using OS keychain
type SecureCredentialProvider struct {
	// Will be implemented in task 3
}

// NewSecureCredentialProvider creates a new secure credential provider
func NewSecureCredentialProvider() (*SecureCredentialProvider, error) {
	// Will be implemented in task 3
	return &SecureCredentialProvider{}, nil
}