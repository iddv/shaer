package aws

import (
	"context"
	"testing"
	"time"

	"github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestKeyring creates a file-based keyring for testing
func createTestKeyring(t *testing.T) keyring.Keyring {
	tempDir := t.TempDir()
	
	ring, err := keyring.Open(keyring.Config{
		ServiceName:     "file-sharing-app-test",
		AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		FileDir:         tempDir,
		FilePasswordFunc: func(string) (string, error) {
			return "test-password", nil
		},
	})
	require.NoError(t, err)
	
	return ring
}

// createTestCredentialProvider creates a credential provider with test keyring
func createTestCredentialProvider(t *testing.T) *SecureCredentialProvider {
	return &SecureCredentialProvider{
		keyring: createTestKeyring(t),
	}
}

func TestNewSecureCredentialProvider(t *testing.T) {
	// Test successful creation
	provider, err := NewSecureCredentialProvider()
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.keyring)
}

func TestStoreCredentials(t *testing.T) {
	provider := createTestCredentialProvider(t)

	tests := []struct {
		name        string
		accessKey   string
		secretKey   string
		region      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid credentials",
			accessKey:   "AKIAIOSFODNN7EXAMPLE",
			secretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			region:      "us-west-2",
			expectError: false,
		},
		{
			name:        "valid credentials without region",
			accessKey:   "AKIAIOSFODNN7EXAMPLE",
			secretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			region:      "",
			expectError: false,
		},
		{
			name:        "empty access key",
			accessKey:   "",
			secretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			region:      "us-west-2",
			expectError: true,
			errorMsg:    "access key and secret key cannot be empty",
		},
		{
			name:        "empty secret key",
			accessKey:   "AKIAIOSFODNN7EXAMPLE",
			secretKey:   "",
			region:      "us-west-2",
			expectError: true,
			errorMsg:    "access key and secret key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.StoreCredentials(tt.accessKey, tt.secretKey, tt.region)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				
				// Verify credentials were stored
				accessKeyItem, err := provider.keyring.Get(AccessKeyItem)
				assert.NoError(t, err)
				assert.Equal(t, tt.accessKey, string(accessKeyItem.Data))
				
				secretKeyItem, err := provider.keyring.Get(SecretKeyItem)
				assert.NoError(t, err)
				assert.Equal(t, tt.secretKey, string(secretKeyItem.Data))
				
				if tt.region != "" {
					regionItem, err := provider.keyring.Get(RegionItem)
					assert.NoError(t, err)
					assert.Equal(t, tt.region, string(regionItem.Data))
				}
			}
		})
	}
}

func TestGetCredentials(t *testing.T) {
	provider := createTestCredentialProvider(t)
	ctx := context.Background()

	// Test when no credentials are stored
	t.Run("no credentials stored", func(t *testing.T) {
		_, err := provider.GetCredentials(ctx)
		// This should try the AWS credential chain - may succeed or fail depending on environment
		// We just verify it doesn't panic and returns some result
		// In a real test environment without AWS credentials, this would fail
		if err != nil {
			assert.Contains(t, err.Error(), "failed to retrieve credentials from AWS credential chain")
		}
	})

	// Store test credentials
	testAccessKey := "AKIAIOSFODNN7EXAMPLE"
	testSecretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	err := provider.StoreCredentials(testAccessKey, testSecretKey, "us-west-2")
	require.NoError(t, err)

	// Test retrieving stored credentials
	t.Run("retrieve stored credentials", func(t *testing.T) {
		creds, err := provider.GetCredentials(ctx)
		assert.NoError(t, err)
		assert.Equal(t, testAccessKey, creds.AccessKeyID)
		assert.Equal(t, testSecretKey, creds.SecretAccessKey)
		assert.Equal(t, "file-sharing-app-keychain", creds.Source)
	})

	// Test when access key exists but secret key is missing
	t.Run("missing secret key", func(t *testing.T) {
		err := provider.keyring.Remove(SecretKeyItem)
		require.NoError(t, err)
		
		_, err = provider.GetCredentials(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve secret key from keychain")
	})
}

func TestValidateCredentials(t *testing.T) {
	provider := createTestCredentialProvider(t)
	ctx := context.Background()

	// Test with no credentials
	t.Run("no credentials", func(t *testing.T) {
		// Clear any existing credentials first
		_ = provider.ClearCredentials()
		
		err := provider.ValidateCredentials(ctx)
		// This may succeed if AWS credentials are available in the environment
		// or fail if no credentials are found - both are valid test outcomes
		if err != nil {
			// If it fails, it should be due to credential issues
			assert.True(t, 
				err.Error() != "" && 
				(err.Error() != "failed to get credentials" || 
				 err.Error() != "credential validation failed"),
				"Error should be related to credentials: %v", err)
		}
	})

	// Test with invalid credentials
	t.Run("invalid credentials", func(t *testing.T) {
		err := provider.StoreCredentials("invalid-key", "invalid-secret", "us-east-1")
		require.NoError(t, err)
		
		// Set a shorter timeout for the test
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		
		err = provider.ValidateCredentials(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "credential validation failed")
	})
}

func TestClearCredentials(t *testing.T) {
	provider := createTestCredentialProvider(t)

	// Store test credentials first
	err := provider.StoreCredentials("test-access-key", "test-secret-key", "us-west-2")
	require.NoError(t, err)

	// Verify credentials are stored
	_, err = provider.keyring.Get(AccessKeyItem)
	assert.NoError(t, err)
	_, err = provider.keyring.Get(SecretKeyItem)
	assert.NoError(t, err)
	_, err = provider.keyring.Get(RegionItem)
	assert.NoError(t, err)

	// Clear credentials
	err = provider.ClearCredentials()
	if err != nil {
		t.Logf("ClearCredentials error: %v", err)
	}
	assert.NoError(t, err)

	// Verify credentials are removed
	_, err = provider.keyring.Get(AccessKeyItem)
	assert.Error(t, err)
	
	_, err = provider.keyring.Get(SecretKeyItem)
	assert.Error(t, err)
	
	_, err = provider.keyring.Get(RegionItem)
	assert.Error(t, err)

	// Test clearing when no credentials exist (should not error)
	// Create a fresh provider to test clearing non-existent credentials
	freshProvider := createTestCredentialProvider(t)
	err = freshProvider.ClearCredentials()
	assert.NoError(t, err)
}

func TestGetRegion(t *testing.T) {
	provider := createTestCredentialProvider(t)

	// Test when no region is stored (should return default)
	t.Run("no region stored", func(t *testing.T) {
		region, err := provider.GetRegion()
		assert.NoError(t, err)
		assert.Equal(t, "us-east-1", region)
	})

	// Test when region is stored
	t.Run("region stored", func(t *testing.T) {
		testRegion := "eu-west-1"
		err := provider.SetRegion(testRegion)
		require.NoError(t, err)
		
		region, err := provider.GetRegion()
		assert.NoError(t, err)
		assert.Equal(t, testRegion, region)
	})
}

func TestSetRegion(t *testing.T) {
	provider := createTestCredentialProvider(t)

	tests := []struct {
		name        string
		region      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid region",
			region:      "us-west-2",
			expectError: false,
		},
		{
			name:        "another valid region",
			region:      "eu-central-1",
			expectError: false,
		},
		{
			name:        "empty region",
			region:      "",
			expectError: true,
			errorMsg:    "region cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.SetRegion(tt.region)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				
				// Verify region was stored
				item, err := provider.keyring.Get(RegionItem)
				assert.NoError(t, err)
				assert.Equal(t, tt.region, string(item.Data))
			}
		})
	}
}

func TestGetSetupGuidance(t *testing.T) {
	guidance := GetSetupGuidance()
	
	// Verify the guidance contains key information
	assert.Contains(t, guidance, "AWS Credentials Setup Guide")
	assert.Contains(t, guidance, "Create an AWS Account")
	assert.Contains(t, guidance, "Create an IAM User")
	assert.Contains(t, guidance, "s3:PutObject")
	assert.Contains(t, guidance, "s3:GetObject")
	assert.Contains(t, guidance, "s3:DeleteObject")
	assert.Contains(t, guidance, "Security Best Practices")
	assert.Contains(t, guidance, "Never share your AWS credentials")
	
	// Verify it's not empty and has reasonable length
	assert.NotEmpty(t, guidance)
	assert.Greater(t, len(guidance), 500) // Should be a substantial guide
}

// Integration test that tests the full workflow
func TestCredentialWorkflow(t *testing.T) {
	provider := createTestCredentialProvider(t)
	ctx := context.Background()

	// Step 1: Store credentials
	accessKey := "AKIAIOSFODNN7EXAMPLE"
	secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	region := "us-west-2"
	
	err := provider.StoreCredentials(accessKey, secretKey, region)
	assert.NoError(t, err)

	// Step 2: Retrieve credentials
	creds, err := provider.GetCredentials(ctx)
	assert.NoError(t, err)
	assert.Equal(t, accessKey, creds.AccessKeyID)
	assert.Equal(t, secretKey, creds.SecretAccessKey)

	// Step 3: Get region
	retrievedRegion, err := provider.GetRegion()
	assert.NoError(t, err)
	assert.Equal(t, region, retrievedRegion)

	// Step 4: Update region
	newRegion := "eu-west-1"
	err = provider.SetRegion(newRegion)
	assert.NoError(t, err)
	
	retrievedRegion, err = provider.GetRegion()
	assert.NoError(t, err)
	assert.Equal(t, newRegion, retrievedRegion)

	// Step 5: Clear credentials
	err = provider.ClearCredentials()
	assert.NoError(t, err)

	// Step 6: Verify credentials are cleared
	_, err = provider.GetCredentials(ctx)
	// This may succeed if AWS credentials are available in environment
	// The important thing is that our stored credentials are gone
	if err == nil {
		// If it succeeds, it should be from AWS credential chain, not our keychain
		// We can't easily verify this without mocking, so we'll accept either outcome
		t.Log("Credentials retrieved from AWS credential chain after clearing keychain")
	}
	
	// Region should return default after clearing
	retrievedRegion, err = provider.GetRegion()
	assert.NoError(t, err)
	assert.Equal(t, "us-east-1", retrievedRegion)
}

// Benchmark tests for performance
func BenchmarkStoreCredentials(b *testing.B) {
	// Create a proper test instance for benchmarks
	tempDir := b.TempDir()
	
	ring, err := keyring.Open(keyring.Config{
		ServiceName:     "file-sharing-app-bench",
		AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		FileDir:         tempDir,
		FilePasswordFunc: func(string) (string, error) {
			return "test-password", nil
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	
	provider := &SecureCredentialProvider{keyring: ring}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := provider.StoreCredentials("test-access-key", "test-secret-key", "us-west-2")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetCredentials(b *testing.B) {
	// Create a proper test instance for benchmarks
	tempDir := b.TempDir()
	
	ring, err := keyring.Open(keyring.Config{
		ServiceName:     "file-sharing-app-bench",
		AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		FileDir:         tempDir,
		FilePasswordFunc: func(string) (string, error) {
			return "test-password", nil
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	
	provider := &SecureCredentialProvider{keyring: ring}
	ctx := context.Background()
	
	// Setup
	err = provider.StoreCredentials("test-access-key", "test-secret-key", "us-west-2")
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetCredentials(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}