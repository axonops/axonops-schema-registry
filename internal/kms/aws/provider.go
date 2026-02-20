// Package aws implements the KMS provider interface using AWS Key Management Service.
//
// Configuration is provided via kmsProps on the KEK record:
//
//	aws.region          — AWS region (default: AWS_REGION env or us-east-1)
//	aws.access.key.id   — AWS access key ID (default: AWS_ACCESS_KEY_ID env, or IAM role)
//	aws.secret.access.key — AWS secret access key (default: AWS_SECRET_ACCESS_KEY env, or IAM role)
//	aws.endpoint        — Custom endpoint URL (for testing with LocalStack etc.)
package aws

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"

	kmsintf "github.com/axonops/axonops-schema-registry/internal/kms"
)

const (
	// ProviderType is the KMS type identifier for AWS KMS.
	ProviderType = "aws-kms"
)

// Provider implements kms.Provider using AWS KMS.
type Provider struct {
	client *kms.Client
	region string
}

// ensure Provider implements kms.Provider at compile time.
var _ kmsintf.Provider = (*Provider)(nil)

// Config holds the AWS KMS provider configuration.
type Config struct {
	Region          string `json:"region" yaml:"region"`
	AccessKeyID     string `json:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key" yaml:"secret_access_key"`
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
}

// NewProvider creates a new AWS KMS provider.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	if cfg.Region == "" {
		cfg.Region = os.Getenv("AWS_REGION")
	}
	if cfg.Region == "" {
		cfg.Region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	opts := []func(*awscfg.LoadOptions) error{
		awscfg.WithRegion(cfg.Region),
	}

	// Use explicit credentials if provided, otherwise fall back to default chain (env, IAM role, etc.)
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awscfg.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awscfg.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws kms: load config: %w", err)
	}

	kmsOpts := []func(*kms.Options){}
	if cfg.Endpoint != "" {
		kmsOpts = append(kmsOpts, func(o *kms.Options) {
			o.BaseEndpoint = &cfg.Endpoint
		})
	}

	client := kms.NewFromConfig(awsCfg, kmsOpts...)

	return &Provider{
		client: client,
		region: cfg.Region,
	}, nil
}

// NewProviderFromProps creates an AWS KMS provider from KEK kmsProps.
func NewProviderFromProps(ctx context.Context, props map[string]string) (*Provider, error) {
	cfg := Config{
		Region:          props["aws.region"],
		AccessKeyID:     props["aws.access.key.id"],
		SecretAccessKey: props["aws.secret.access.key"],
		Endpoint:        props["aws.endpoint"],
	}
	return NewProvider(ctx, cfg)
}

// Type returns the provider type identifier.
func (p *Provider) Type() string {
	return ProviderType
}

// Wrap encrypts plaintext using AWS KMS Encrypt.
func (p *Provider) Wrap(ctx context.Context, kmsKeyID string, plaintext []byte, props map[string]string) ([]byte, error) {
	output, err := p.client.Encrypt(ctx, &kms.EncryptInput{
		KeyId:               &kmsKeyID,
		Plaintext:           plaintext,
		EncryptionAlgorithm: kmstypes.EncryptionAlgorithmSpecSymmetricDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("aws kms encrypt: %w", err)
	}

	return output.CiphertextBlob, nil
}

// Unwrap decrypts ciphertext using AWS KMS Decrypt.
func (p *Provider) Unwrap(ctx context.Context, kmsKeyID string, ciphertext []byte, props map[string]string) ([]byte, error) {
	output, err := p.client.Decrypt(ctx, &kms.DecryptInput{
		KeyId:               &kmsKeyID,
		CiphertextBlob:      ciphertext,
		EncryptionAlgorithm: kmstypes.EncryptionAlgorithmSpecSymmetricDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("aws kms decrypt: %w", err)
	}

	return output.Plaintext, nil
}

// GenerateDataKey generates a new data encryption key using AWS KMS GenerateDataKey.
func (p *Provider) GenerateDataKey(ctx context.Context, kmsKeyID string, algorithm string, props map[string]string) (plaintext []byte, wrapped []byte, err error) {
	keySpec := dataKeySpecForAlgorithm(algorithm)

	output, err := p.client.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:   &kmsKeyID,
		KeySpec: keySpec,
	})
	if err != nil {
		// Fall back to local generation + Wrap if GenerateDataKey is not supported
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

	return output.Plaintext, output.CiphertextBlob, nil
}

// Close is a no-op for AWS KMS — the SDK client doesn't need explicit cleanup.
func (p *Provider) Close() error {
	return nil
}

func dataKeySpecForAlgorithm(algorithm string) kmstypes.DataKeySpec {
	switch algorithm {
	case "AES128_GCM":
		return kmstypes.DataKeySpecAes128
	case "AES256_GCM", "AES256_SIV":
		return kmstypes.DataKeySpecAes256
	default:
		return kmstypes.DataKeySpecAes256
	}
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
