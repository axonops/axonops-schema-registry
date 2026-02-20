package gcp

import (
	"testing"
)

func TestProviderType_Constant(t *testing.T) {
	if ProviderType != "gcp-kms" {
		t.Errorf("ProviderType = %q, want %q", ProviderType, "gcp-kms")
	}
}

func TestCryptoKeyName(t *testing.T) {
	p := &Provider{
		projectID: "my-project",
		location:  "us-east1",
		keyRing:   "my-ring",
	}

	got := p.cryptoKeyName("my-key")
	want := "projects/my-project/locations/us-east1/keyRings/my-ring/cryptoKeys/my-key"
	if got != want {
		t.Errorf("cryptoKeyName() = %q, want %q", got, want)
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
