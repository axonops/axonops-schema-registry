// Package openbao implements the KMS provider interface using OpenBao's
// Transit secrets engine. OpenBao is an API-compatible fork of HashiCorp Vault,
// so this provider reuses the Vault Transit implementation with different
// default environment variables and provider type identifier.
//
// Configuration is provided via kmsProps on the KEK record:
//
//	openbao.address       — OpenBao server address (default: BAO_ADDR env or http://127.0.0.1:8200)
//	openbao.token         — OpenBao token (default: BAO_TOKEN env)
//	openbao.namespace     — OpenBao namespace (default: BAO_NAMESPACE env, empty for root)
//	openbao.transit.mount — Transit mount path (default: "transit")
package openbao

import (
	"context"
	"os"

	"github.com/axonops/axonops-schema-registry/internal/kms"
	vaultprovider "github.com/axonops/axonops-schema-registry/internal/kms/vault"
)

const (
	// ProviderType is the KMS type identifier for OpenBao.
	ProviderType = "openbao"
)

// Provider wraps the Vault Transit provider with OpenBao-specific defaults.
type Provider struct {
	inner *vaultprovider.Provider
}

// ensure Provider implements kms.Provider at compile time.
var _ kms.Provider = (*Provider)(nil)

// NewProvider creates a new OpenBao Transit KMS provider.
func NewProvider(cfg vaultprovider.Config) (*Provider, error) {
	// Apply OpenBao environment variable defaults
	if cfg.Address == "" {
		cfg.Address = os.Getenv("BAO_ADDR")
	}
	if cfg.Token == "" {
		cfg.Token = os.Getenv("BAO_TOKEN")
	}
	if cfg.Namespace == "" {
		cfg.Namespace = os.Getenv("BAO_NAMESPACE")
	}

	inner, err := vaultprovider.NewProvider(cfg)
	if err != nil {
		return nil, err
	}
	return &Provider{inner: inner}, nil
}

// NewProviderFromProps creates an OpenBao provider from KEK kmsProps.
func NewProviderFromProps(props map[string]string) (*Provider, error) {
	cfg := vaultprovider.Config{
		Address:      props["openbao.address"],
		Token:        props["openbao.token"],
		Namespace:    props["openbao.namespace"],
		TransitMount: props["openbao.transit.mount"],
	}
	return NewProvider(cfg)
}

func (p *Provider) Type() string { return ProviderType }
func (p *Provider) Close() error { return p.inner.Close() }

func (p *Provider) Wrap(ctx context.Context, kmsKeyID string, plaintext []byte, props map[string]string) ([]byte, error) {
	return p.inner.Wrap(ctx, kmsKeyID, plaintext, props)
}

func (p *Provider) Unwrap(ctx context.Context, kmsKeyID string, ciphertext []byte, props map[string]string) ([]byte, error) {
	return p.inner.Unwrap(ctx, kmsKeyID, ciphertext, props)
}

func (p *Provider) GenerateDataKey(ctx context.Context, kmsKeyID string, algorithm string, props map[string]string) (plaintext []byte, wrapped []byte, err error) {
	return p.inner.GenerateDataKey(ctx, kmsKeyID, algorithm, props)
}
