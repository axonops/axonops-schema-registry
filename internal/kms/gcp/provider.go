// Package gcp implements the KMS provider interface using Google Cloud KMS.
//
// Configuration is provided via kmsProps on the KEK record:
//
//	gcp.project.id       — GCP project ID (default: GOOGLE_CLOUD_PROJECT env)
//	gcp.location         — KMS location (default: "global")
//	gcp.key.ring         — Key ring name
//	gcp.credentials.path — Path to service account JSON (default: GOOGLE_APPLICATION_CREDENTIALS env)
package gcp

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"

	kmsapi "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/option"

	kmsintf "github.com/axonops/axonops-schema-registry/internal/kms"
)

const (
	// ProviderType is the KMS type identifier for GCP Cloud KMS.
	ProviderType = "gcp-kms"
)

// Provider implements kms.Provider using GCP Cloud KMS.
type Provider struct {
	client    *kmsapi.KeyManagementClient
	projectID string
	location  string
	keyRing   string
}

// ensure Provider implements kms.Provider at compile time.
var _ kmsintf.Provider = (*Provider)(nil)

// Config holds the GCP Cloud KMS provider configuration.
type Config struct {
	ProjectID       string `json:"project_id" yaml:"project_id"`
	Location        string `json:"location" yaml:"location"`
	KeyRing         string `json:"key_ring" yaml:"key_ring"`
	CredentialsPath string `json:"credentials_path" yaml:"credentials_path"`
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
}

// NewProvider creates a new GCP Cloud KMS provider.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	if cfg.ProjectID == "" {
		cfg.ProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if cfg.ProjectID == "" {
		cfg.ProjectID = os.Getenv("GCLOUD_PROJECT")
	}
	if cfg.Location == "" {
		cfg.Location = "global"
	}
	if cfg.CredentialsPath == "" {
		cfg.CredentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	var opts []option.ClientOption
	if cfg.CredentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsPath))
	}
	if cfg.Endpoint != "" {
		opts = append(opts, option.WithEndpoint(cfg.Endpoint))
		opts = append(opts, option.WithoutAuthentication())
	}

	client, err := kmsapi.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("gcp kms: create client: %w", err)
	}

	return &Provider{
		client:    client,
		projectID: cfg.ProjectID,
		location:  cfg.Location,
		keyRing:   cfg.KeyRing,
	}, nil
}

// NewProviderFromProps creates a GCP Cloud KMS provider from KEK kmsProps.
func NewProviderFromProps(ctx context.Context, props map[string]string) (*Provider, error) {
	cfg := Config{
		ProjectID:       props["gcp.project.id"],
		Location:        props["gcp.location"],
		KeyRing:         props["gcp.key.ring"],
		CredentialsPath: props["gcp.credentials.path"],
		Endpoint:        props["gcp.endpoint"],
	}
	return NewProvider(ctx, cfg)
}

// Type returns the provider type identifier.
func (p *Provider) Type() string {
	return ProviderType
}

// cryptoKeyName builds the full GCP Cloud KMS key resource name.
// The kmsKeyID is the crypto key name within the key ring.
func (p *Provider) cryptoKeyName(kmsKeyID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		p.projectID, p.location, p.keyRing, kmsKeyID)
}

// Wrap encrypts plaintext using GCP Cloud KMS Encrypt.
func (p *Provider) Wrap(ctx context.Context, kmsKeyID string, plaintext []byte, props map[string]string) ([]byte, error) {
	resp, err := p.client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      p.cryptoKeyName(kmsKeyID),
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, fmt.Errorf("gcp kms encrypt: %w", err)
	}

	return resp.Ciphertext, nil
}

// Unwrap decrypts ciphertext using GCP Cloud KMS Decrypt.
func (p *Provider) Unwrap(ctx context.Context, kmsKeyID string, ciphertext []byte, props map[string]string) ([]byte, error) {
	resp, err := p.client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       p.cryptoKeyName(kmsKeyID),
		Ciphertext: ciphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("gcp kms decrypt: %w", err)
	}

	return resp.Plaintext, nil
}

// GenerateDataKey generates a new data encryption key.
// GCP Cloud KMS doesn't have a native GenerateDataKey API,
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

// Close closes the GCP KMS client connection.
func (p *Provider) Close() error {
	return p.client.Close()
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
