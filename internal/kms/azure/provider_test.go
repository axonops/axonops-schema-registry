package azure

import (
	"testing"
)

func TestProviderType_Constant(t *testing.T) {
	if ProviderType != "azure-kms" {
		t.Errorf("ProviderType = %q, want %q", ProviderType, "azure-kms")
	}
}

func TestKeySizeForAlgorithm(t *testing.T) {
	tests := []struct {
		algo string
		want int
	}{
		{"AES128_GCM", 16},
		{"AES256_GCM", 32},
		{"AES256_SIV", 32},
		{"UNKNOWN", 32},
	}
	for _, tt := range tests {
		if got := keySizeForAlgorithm(tt.algo); got != tt.want {
			t.Errorf("keySizeForAlgorithm(%q) = %d, want %d", tt.algo, got, tt.want)
		}
	}
}

func TestNewProvider_MissingVaultURL(t *testing.T) {
	_, err := NewProvider(Config{
		TenantID:     "test-tenant",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
	if err == nil {
		t.Fatal("expected error for missing vault URL")
	}
}

func TestNewProvider_MissingCredentials(t *testing.T) {
	_, err := NewProvider(Config{
		VaultURL: "https://test.vault.azure.net",
	})
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
}
