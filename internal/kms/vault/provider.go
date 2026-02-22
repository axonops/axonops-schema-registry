// Package vault implements the KMS provider interface using HashiCorp Vault's
// Transit secrets engine for key wrapping and data key generation.
//
// Configuration is provided via kmsProps on the KEK record:
//
//	vault.address       — Vault server address (default: VAULT_ADDR env or http://127.0.0.1:8200)
//	vault.token         — Vault token (default: VAULT_TOKEN env)
//	vault.namespace     — Vault namespace (default: VAULT_NAMESPACE env, empty for root)
//	vault.transit.mount — Transit mount path (default: "transit")
package vault

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	vaultapi "github.com/hashicorp/vault/api"

	"github.com/axonops/axonops-schema-registry/internal/kms"
)

const (
	// ProviderType is the KMS type identifier for Vault Transit.
	ProviderType = "hcvault"

	defaultTransitMount = "transit"
)

// Provider implements kms.Provider using Vault Transit.
type Provider struct {
	client       *vaultapi.Client
	transitMount string
}

// ensure Provider implements kms.Provider at compile time.
var _ kms.Provider = (*Provider)(nil)

// Config holds the Vault Transit provider configuration.
type Config struct {
	Address      string `json:"address" yaml:"address"`
	Token        string `json:"token" yaml:"token"`
	Namespace    string `json:"namespace" yaml:"namespace"`
	TransitMount string `json:"transit_mount" yaml:"transit_mount"`
}

// NewProvider creates a new Vault Transit KMS provider.
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.Address == "" {
		cfg.Address = os.Getenv("VAULT_ADDR")
	}
	if cfg.Address == "" {
		cfg.Address = "http://127.0.0.1:8200"
	}
	if cfg.Token == "" {
		cfg.Token = os.Getenv("VAULT_TOKEN")
	}
	if cfg.Namespace == "" {
		cfg.Namespace = os.Getenv("VAULT_NAMESPACE")
	}
	if cfg.TransitMount == "" {
		cfg.TransitMount = defaultTransitMount
	}

	vaultCfg := vaultapi.DefaultConfig()
	vaultCfg.Address = cfg.Address

	client, err := vaultapi.NewClient(vaultCfg)
	if err != nil {
		return nil, fmt.Errorf("vault: create client: %w", err)
	}
	client.SetToken(cfg.Token)
	if cfg.Namespace != "" {
		client.SetNamespace(cfg.Namespace)
	}

	return &Provider{
		client:       client,
		transitMount: cfg.TransitMount,
	}, nil
}

// NewProviderFromProps creates a Vault Transit provider from KEK kmsProps.
func NewProviderFromProps(props map[string]string) (*Provider, error) {
	cfg := Config{
		Address:      props["vault.address"],
		Token:        props["vault.token"],
		Namespace:    props["vault.namespace"],
		TransitMount: props["vault.transit.mount"],
	}
	return NewProvider(cfg)
}

// Type returns the provider type identifier.
func (p *Provider) Type() string {
	return ProviderType
}

// Wrap encrypts plaintext using Vault Transit's encrypt endpoint.
func (p *Provider) Wrap(ctx context.Context, kmsKeyID string, plaintext []byte, props map[string]string) ([]byte, error) {
	path := fmt.Sprintf("%s/encrypt/%s", p.transitMount, kmsKeyID)

	secret, err := p.client.Logical().WriteWithContext(ctx, path, map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	})
	if err != nil {
		return nil, fmt.Errorf("vault transit encrypt: %w", err)
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return nil, fmt.Errorf("vault transit encrypt: missing ciphertext in response")
	}

	return []byte(ciphertext), nil
}

// Unwrap decrypts ciphertext using Vault Transit's decrypt endpoint.
func (p *Provider) Unwrap(ctx context.Context, kmsKeyID string, ciphertext []byte, props map[string]string) ([]byte, error) {
	path := fmt.Sprintf("%s/decrypt/%s", p.transitMount, kmsKeyID)

	secret, err := p.client.Logical().WriteWithContext(ctx, path, map[string]interface{}{
		"ciphertext": string(ciphertext),
	})
	if err != nil {
		return nil, fmt.Errorf("vault transit decrypt: %w", err)
	}

	b64Plaintext, ok := secret.Data["plaintext"].(string)
	if !ok {
		return nil, fmt.Errorf("vault transit decrypt: missing plaintext in response")
	}

	plaintext, err := base64.StdEncoding.DecodeString(b64Plaintext)
	if err != nil {
		return nil, fmt.Errorf("vault transit decrypt: decode plaintext: %w", err)
	}

	return plaintext, nil
}

// GenerateDataKey generates a new data encryption key.
// It generates random key material locally and wraps it using Vault Transit.
func (p *Provider) GenerateDataKey(ctx context.Context, kmsKeyID string, algorithm string, props map[string]string) (plaintext []byte, wrapped []byte, err error) {
	keySize := keySizeForAlgorithm(algorithm)

	// Generate random key material locally
	plaintext = make([]byte, keySize)
	if _, err := rand.Read(plaintext); err != nil {
		return nil, nil, fmt.Errorf("generate random key: %w", err)
	}

	// Wrap using Vault Transit
	wrapped, err = p.Wrap(ctx, kmsKeyID, plaintext, props)
	if err != nil {
		return nil, nil, err
	}

	return plaintext, wrapped, nil
}

// Close is a no-op for Vault — the HTTP client doesn't need explicit cleanup.
func (p *Provider) Close() error {
	return nil
}

// keySizeForAlgorithm returns the key size in bytes for the given algorithm.
func keySizeForAlgorithm(algorithm string) int {
	switch algorithm {
	case "AES128_GCM":
		return 16
	case "AES256_GCM", "AES256_SIV":
		return 32
	default:
		return 32 // default to 256-bit
	}
}
