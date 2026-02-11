package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// generateTestCert creates a self-signed cert and key in the given directory.
func generateTestCert(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:         true,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")

	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal key: %v", err)
	}

	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	keyOut.Close()

	return certFile, keyFile
}

func TestNewTLSManager(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, err := NewTLSManager(config.TLSConfig{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  keyFile,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm == nil {
		t.Fatal("expected non-nil TLSManager")
	}
	if tm.cert == nil {
		t.Error("expected certificate to be loaded")
	}
}

func TestNewTLSManager_InvalidCert(t *testing.T) {
	_, err := NewTLSManager(config.TLSConfig{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	})
	if err == nil {
		t.Error("expected error for invalid cert files")
	}
}

func TestNewTLSManager_WithCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, err := NewTLSManager(config.TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   certFile, // Self-signed, use as CA too
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm.clientCAs == nil {
		t.Error("expected clientCAs to be loaded")
	}
}

func TestNewTLSManager_InvalidCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	_, err := NewTLSManager(config.TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   "/nonexistent/ca.pem",
	})
	if err == nil {
		t.Error("expected error for invalid CA file")
	}
}

func TestNewTLSManager_InvalidCAPEM(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	// Write a file that's not valid PEM
	badCA := filepath.Join(dir, "bad-ca.pem")
	os.WriteFile(badCA, []byte("not a certificate"), 0600)

	_, err := NewTLSManager(config.TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   badCA,
	})
	if err == nil {
		t.Error("expected error for invalid CA PEM")
	}
}

func TestTLSManager_GetCertificate(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
	})

	cert, err := tm.GetCertificate(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert == nil {
		t.Error("expected certificate")
	}
}

func TestTLSManager_Reload(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
	})

	// Reload should succeed with same files
	if err := tm.Reload(); err != nil {
		t.Fatalf("unexpected error on reload: %v", err)
	}
}

func TestTLSManager_GetMinVersion(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tests := []struct {
		version  string
		expected uint16
	}{
		{"TLS1.0", tls.VersionTLS10},
		{"1.0", tls.VersionTLS10},
		{"TLS1.1", tls.VersionTLS11},
		{"1.1", tls.VersionTLS11},
		{"TLS1.2", tls.VersionTLS12},
		{"1.2", tls.VersionTLS12},
		{"TLS1.3", tls.VersionTLS13},
		{"1.3", tls.VersionTLS13},
		{"", tls.VersionTLS12},       // default
		{"invalid", tls.VersionTLS12}, // default
	}

	for _, tt := range tests {
		tm, _ := NewTLSManager(config.TLSConfig{
			CertFile:   certFile,
			KeyFile:    keyFile,
			MinVersion: tt.version,
		})
		got := tm.getMinVersion()
		if got != tt.expected {
			t.Errorf("getMinVersion(%q) = %d, want %d", tt.version, got, tt.expected)
		}
	}
}

func TestTLSManager_TLSConfig_ClientAuth(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tests := []struct {
		clientAuth string
		expected   tls.ClientAuthType
	}{
		{"none", tls.NoClientCert},
		{"", tls.NoClientCert},
		{"request", tls.RequestClientCert},
		{"require", tls.RequireAnyClientCert},
		{"verify", tls.RequireAndVerifyClientCert},
	}

	for _, tt := range tests {
		tm, _ := NewTLSManager(config.TLSConfig{
			CertFile:   certFile,
			KeyFile:    keyFile,
			CAFile:     certFile, // Needed for "verify" mode
			ClientAuth: tt.clientAuth,
		})
		tlsCfg := tm.TLSConfig()
		if tlsCfg.ClientAuth != tt.expected {
			t.Errorf("ClientAuth(%q) = %v, want %v", tt.clientAuth, tlsCfg.ClientAuth, tt.expected)
		}
	}
}

func TestTLSManager_TLSConfig_VerifyHasClientCAs(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     certFile,
		ClientAuth: "verify",
	})
	tlsCfg := tm.TLSConfig()
	if tlsCfg.ClientCAs == nil {
		t.Error("expected ClientCAs to be set for verify mode")
	}
}

func TestCreateServerTLSConfig_Disabled(t *testing.T) {
	cfg, err := CreateServerTLSConfig(config.TLSConfig{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config when TLS disabled")
	}
}

func TestCreateServerTLSConfig_Enabled(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	cfg, err := CreateServerTLSConfig(config.TLSConfig{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  keyFile,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Error("expected non-nil TLS config")
	}
}

func TestCreateServerTLSConfig_InvalidCert(t *testing.T) {
	_, err := CreateServerTLSConfig(config.TLSConfig{
		Enabled:  true,
		CertFile: "/bad/cert.pem",
		KeyFile:  "/bad/key.pem",
	})
	if err == nil {
		t.Error("expected error for invalid certs")
	}
}

func TestCreateClientTLSConfig_Minimal(t *testing.T) {
	cfg, err := CreateClientTLSConfig("", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify=false")
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 min, got %d", cfg.MinVersion)
	}
}

func TestCreateClientTLSConfig_InsecureSkipVerify(t *testing.T) {
	cfg, err := CreateClientTLSConfig("", "", "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify=true")
	}
}

func TestCreateClientTLSConfig_WithClientCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	cfg, err := CreateClientTLSConfig(certFile, keyFile, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 client cert, got %d", len(cfg.Certificates))
	}
}

func TestCreateClientTLSConfig_InvalidClientCert(t *testing.T) {
	_, err := CreateClientTLSConfig("/bad/cert.pem", "/bad/key.pem", "", false)
	if err == nil {
		t.Error("expected error for invalid client cert")
	}
}

func TestCreateClientTLSConfig_WithCA(t *testing.T) {
	dir := t.TempDir()
	certFile, _ := generateTestCert(t, dir)

	cfg, err := CreateClientTLSConfig("", "", certFile, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
}

func TestCreateClientTLSConfig_InvalidCA(t *testing.T) {
	_, err := CreateClientTLSConfig("", "", "/bad/ca.pem", false)
	if err == nil {
		t.Error("expected error for invalid CA file")
	}
}

func TestCreateClientTLSConfig_InvalidCAPEM(t *testing.T) {
	dir := t.TempDir()
	badCA := filepath.Join(dir, "bad.pem")
	os.WriteFile(badCA, []byte("not a certificate"), 0600)

	_, err := CreateClientTLSConfig("", "", badCA, false)
	if err == nil {
		t.Error("expected error for invalid CA PEM")
	}
}
