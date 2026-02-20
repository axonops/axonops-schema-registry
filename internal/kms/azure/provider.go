// Package azure implements the KMS provider interface using Azure Key Vault.
//
// Configuration is provided via kmsProps on the KEK record:
//
//	azure.tenant.id      — Azure AD tenant ID (default: AZURE_TENANT_ID env)
//	azure.client.id      — Azure AD client ID (default: AZURE_CLIENT_ID env)
//	azure.client.secret  — Azure AD client secret (default: AZURE_CLIENT_SECRET env)
//	azure.keyvault.url   — Key Vault URL (e.g. https://myvault.vault.azure.net)
//	azure.key.name       — Key name in Key Vault (overrides kmsKeyID if set)
//	azure.key.version    — Key version (optional, defaults to latest)
package azure

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"

	kmsintf "github.com/axonops/axonops-schema-registry/internal/kms"
)

const (
	// ProviderType is the KMS type identifier for Azure Key Vault.
	ProviderType = "azure-kms"
)

// Provider implements kms.Provider using Azure Key Vault.
type Provider struct {
	client     *azkeys.Client
	vaultURL   string
	keyVersion string
}

// ensure Provider implements kms.Provider at compile time.
var _ kmsintf.Provider = (*Provider)(nil)

// Config holds the Azure Key Vault provider configuration.
type Config struct {
	TenantID     string `json:"tenant_id" yaml:"tenant_id"`
	ClientID     string `json:"client_id" yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
	VaultURL     string `json:"vault_url" yaml:"vault_url"`
	KeyVersion   string `json:"key_version" yaml:"key_version"`
}

// NewProvider creates a new Azure Key Vault KMS provider.
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.TenantID == "" {
		cfg.TenantID = os.Getenv("AZURE_TENANT_ID")
	}
	if cfg.ClientID == "" {
		cfg.ClientID = os.Getenv("AZURE_CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = os.Getenv("AZURE_CLIENT_SECRET")
	}
	if cfg.VaultURL == "" {
		return nil, fmt.Errorf("azure key vault URL is required")
	}

	var cred *azidentity.ClientSecretCredential
	var err error

	if cfg.TenantID != "" && cfg.ClientID != "" && cfg.ClientSecret != "" {
		cred, err = azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("azure: create credential: %w", err)
		}
	} else {
		return nil, fmt.Errorf("azure: tenant_id, client_id, and client_secret are required")
	}

	client, err := azkeys.NewClient(cfg.VaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: create key vault client: %w", err)
	}

	return &Provider{
		client:     client,
		vaultURL:   cfg.VaultURL,
		keyVersion: cfg.KeyVersion,
	}, nil
}

// NewProviderFromProps creates an Azure Key Vault provider from KEK kmsProps.
func NewProviderFromProps(props map[string]string) (*Provider, error) {
	cfg := Config{
		TenantID:     props["azure.tenant.id"],
		ClientID:     props["azure.client.id"],
		ClientSecret: props["azure.client.secret"],
		VaultURL:     props["azure.keyvault.url"],
		KeyVersion:   props["azure.key.version"],
	}
	return NewProvider(cfg)
}

// Type returns the provider type identifier.
func (p *Provider) Type() string {
	return ProviderType
}

// Wrap encrypts plaintext using Azure Key Vault wrapKey.
func (p *Provider) Wrap(ctx context.Context, kmsKeyID string, plaintext []byte, props map[string]string) ([]byte, error) {
	version := p.keyVersion
	if v, ok := props["azure.key.version"]; ok && v != "" {
		version = v
	}

	algo := azkeys.EncryptionAlgorithmRSAOAEP256
	params := azkeys.KeyOperationParameters{
		Algorithm: &algo,
		Value:     plaintext,
	}

	resp, err := p.client.WrapKey(ctx, kmsKeyID, version, params, nil)
	if err != nil {
		return nil, fmt.Errorf("azure key vault wrapKey: %w", err)
	}

	return resp.Result, nil
}

// Unwrap decrypts ciphertext using Azure Key Vault unwrapKey.
func (p *Provider) Unwrap(ctx context.Context, kmsKeyID string, ciphertext []byte, props map[string]string) ([]byte, error) {
	version := p.keyVersion
	if v, ok := props["azure.key.version"]; ok && v != "" {
		version = v
	}

	algo := azkeys.EncryptionAlgorithmRSAOAEP256
	params := azkeys.KeyOperationParameters{
		Algorithm: &algo,
		Value:     ciphertext,
	}

	resp, err := p.client.UnwrapKey(ctx, kmsKeyID, version, params, nil)
	if err != nil {
		return nil, fmt.Errorf("azure key vault unwrapKey: %w", err)
	}

	return resp.Result, nil
}

// GenerateDataKey generates a new data encryption key.
// Azure Key Vault doesn't have a native GenerateDataKey API,
// so we generate random key material locally and wrap it.
func (p *Provider) GenerateDataKey(ctx context.Context, kmsKeyID string, algorithm string, props map[string]string) (plaintext []byte, wrapped []byte, err error) {
	keySize := keySizeForAlgorithm(algorithm)

	plaintext = make([]byte, keySize)
	if _, err := rand.Read(plaintext); err != nil {
		return nil, nil, fmt.Errorf("generate random key: %w", err)
	}

	wrapped, err = p.Wrap(ctx, kmsKeyID, plaintext, props)
	if err != nil {
		return nil, nil, err
	}

	return plaintext, wrapped, nil
}

// Close is a no-op for Azure Key Vault.
func (p *Provider) Close() error {
	return nil
}

func keySizeForAlgorithm(algorithm string) int {
	switch algorithm {
	case "AES128_GCM":
		return 16
	case "AES256_GCM", "AES256_SIV":
		return 32
	default:
		return 32
	}
}
