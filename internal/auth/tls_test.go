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
	"strings"
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
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:                  true,
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

// --- parseMinVersion tests ---

func TestParseMinVersion_Default(t *testing.T) {
	v, err := parseMinVersion("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3 default, got %d", v)
	}
}

func TestParseMinVersion_TLS12(t *testing.T) {
	for _, input := range []string{"TLS1.2", "1.2"} {
		v, err := parseMinVersion(input)
		if err != nil {
			t.Fatalf("parseMinVersion(%q) unexpected error: %v", input, err)
		}
		if v != tls.VersionTLS12 {
			t.Errorf("parseMinVersion(%q) = %d, want TLS 1.2", input, v)
		}
	}
}

func TestParseMinVersion_TLS13(t *testing.T) {
	for _, input := range []string{"TLS1.3", "1.3"} {
		v, err := parseMinVersion(input)
		if err != nil {
			t.Fatalf("parseMinVersion(%q) unexpected error: %v", input, err)
		}
		if v != tls.VersionTLS13 {
			t.Errorf("parseMinVersion(%q) = %d, want TLS 1.3", input, v)
		}
	}
}

func TestParseMinVersion_RejectTLS10(t *testing.T) {
	for _, input := range []string{"TLS1.0", "1.0"} {
		_, err := parseMinVersion(input)
		if err == nil {
			t.Errorf("parseMinVersion(%q) expected fatal error, got nil", input)
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("parseMinVersion(%q) error = %q, want 'not supported'", input, err.Error())
		}
	}
}

func TestParseMinVersion_RejectTLS11(t *testing.T) {
	for _, input := range []string{"TLS1.1", "1.1"} {
		_, err := parseMinVersion(input)
		if err == nil {
			t.Errorf("parseMinVersion(%q) expected fatal error, got nil", input)
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("parseMinVersion(%q) error = %q, want 'not supported'", input, err.Error())
		}
	}
}

func TestParseMinVersion_RejectUnknown(t *testing.T) {
	for _, input := range []string{"SSLv3", "TLS1.4", "bogus"} {
		_, err := parseMinVersion(input)
		if err == nil {
			t.Errorf("parseMinVersion(%q) expected fatal error, got nil", input)
		}
		if !strings.Contains(err.Error(), "unrecognised") {
			t.Errorf("parseMinVersion(%q) error = %q, want 'unrecognised'", input, err.Error())
		}
	}
}

// --- resolveCipherSuites tests ---

func TestResolveCipherSuites_AllSecure(t *testing.T) {
	// Use two known-good cipher suite names from tls.CipherSuites()
	suites := tls.CipherSuites()
	if len(suites) < 2 {
		t.Skip("not enough cipher suites available")
	}
	names := []string{suites[0].Name, suites[1].Name}
	ids, insecure, err := resolveCipherSuites(names)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(ids))
	}
	if len(insecure) != 0 {
		t.Errorf("expected 0 insecure, got %v", insecure)
	}
}

func TestResolveCipherSuites_UnknownName(t *testing.T) {
	_, _, err := resolveCipherSuites([]string{"TLS_FAKE_CIPHER_SUITE"})
	if err == nil {
		t.Error("expected error for unknown cipher suite")
	}
	if !strings.Contains(err.Error(), "unknown cipher suite") {
		t.Errorf("error = %q, want 'unknown cipher suite'", err.Error())
	}
}

func TestResolveCipherSuites_InsecureDetected(t *testing.T) {
	insecureSuites := tls.InsecureCipherSuites()
	if len(insecureSuites) == 0 {
		t.Skip("no insecure cipher suites available")
	}
	names := []string{insecureSuites[0].Name}
	ids, insecure, err := resolveCipherSuites(names)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 ID, got %d", len(ids))
	}
	if len(insecure) != 1 {
		t.Errorf("expected 1 insecure name, got %v", insecure)
	}
	if insecure[0] != insecureSuites[0].Name {
		t.Errorf("insecure[0] = %q, want %q", insecure[0], insecureSuites[0].Name)
	}
}

func TestResolveCipherSuites_MixedSecureAndInsecure(t *testing.T) {
	secureSuites := tls.CipherSuites()
	insecureSuites := tls.InsecureCipherSuites()
	if len(secureSuites) == 0 || len(insecureSuites) == 0 {
		t.Skip("need at least one secure and one insecure suite")
	}
	names := []string{secureSuites[0].Name, insecureSuites[0].Name}
	ids, insecure, err := resolveCipherSuites(names)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(ids))
	}
	// Only the insecure one should be reported
	if len(insecure) != 1 {
		t.Errorf("expected 1 insecure, got %v", insecure)
	}
}

// --- BuildTLSConfig tests ---

func TestBuildTLSConfig_DefaultCiphers_TLS13(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "TLS1.3",
	})
	tlsCfg, err := tm.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// TLS 1.3 cipher suites are not configurable — CipherSuites should be nil
	if tlsCfg.CipherSuites != nil {
		t.Error("expected nil CipherSuites for TLS 1.3 minimum")
	}
	if tlsCfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3, got %d", tlsCfg.MinVersion)
	}
}

func TestBuildTLSConfig_DefaultCiphers_TLS12(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "TLS1.2",
	})
	tlsCfg, err := tm.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should use Go's secure defaults
	expected := defaultSecureCipherSuites()
	if len(tlsCfg.CipherSuites) != len(expected) {
		t.Errorf("expected %d cipher suites, got %d", len(expected), len(tlsCfg.CipherSuites))
	}
}

func TestBuildTLSConfig_CustomSecureCiphers(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	suites := tls.CipherSuites()
	if len(suites) == 0 {
		t.Skip("no secure cipher suites available")
	}

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:     certFile,
		KeyFile:      keyFile,
		MinVersion:   "TLS1.2",
		CipherSuites: []string{suites[0].Name},
	})
	tlsCfg, err := tm.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tlsCfg.CipherSuites) != 1 {
		t.Errorf("expected 1 cipher suite, got %d", len(tlsCfg.CipherSuites))
	}
	if len(tm.InsecureCipherNames()) != 0 {
		t.Error("expected no insecure cipher names")
	}
}

func TestBuildTLSConfig_InsecureCiphers_Rejected(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	insecureSuites := tls.InsecureCipherSuites()
	if len(insecureSuites) == 0 {
		t.Skip("no insecure cipher suites available")
	}

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:             certFile,
		KeyFile:              keyFile,
		MinVersion:           "TLS1.2",
		CipherSuites:         []string{insecureSuites[0].Name},
		AllowInsecureCiphers: false,
	})
	_, err := tm.BuildTLSConfig()
	if err == nil {
		t.Fatal("expected fatal error for insecure cipher without allow_insecure_ciphers")
	}
	if !strings.Contains(err.Error(), "refusing to start") {
		t.Errorf("error = %q, want 'refusing to start'", err.Error())
	}
	if !strings.Contains(err.Error(), insecureSuites[0].Name) {
		t.Errorf("error should list insecure cipher name %q, got: %q", insecureSuites[0].Name, err.Error())
	}
}

func TestBuildTLSConfig_InsecureCiphers_Allowed(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	insecureSuites := tls.InsecureCipherSuites()
	secureSuites := tls.CipherSuites()
	if len(insecureSuites) == 0 || len(secureSuites) == 0 {
		t.Skip("need at least one secure and one insecure suite")
	}

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:             certFile,
		KeyFile:              keyFile,
		MinVersion:           "TLS1.2",
		CipherSuites:         []string{secureSuites[0].Name, insecureSuites[0].Name},
		AllowInsecureCiphers: true,
	})
	tlsCfg, err := tm.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tlsCfg.CipherSuites) != 2 {
		t.Errorf("expected 2 cipher suites, got %d", len(tlsCfg.CipherSuites))
	}
	insecureNames := tm.InsecureCipherNames()
	if len(insecureNames) != 1 {
		t.Errorf("expected 1 insecure cipher name, got %v", insecureNames)
	}
	if insecureNames[0] != insecureSuites[0].Name {
		t.Errorf("insecure name = %q, want %q", insecureNames[0], insecureSuites[0].Name)
	}
}

func TestBuildTLSConfig_UnknownCipherName(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:     certFile,
		KeyFile:      keyFile,
		MinVersion:   "TLS1.2",
		CipherSuites: []string{"TLS_NONEXISTENT_CIPHER"},
	})
	_, err := tm.BuildTLSConfig()
	if err == nil {
		t.Fatal("expected error for unknown cipher suite name")
	}
	if !strings.Contains(err.Error(), "unknown cipher suite") {
		t.Errorf("error = %q, want 'unknown cipher suite'", err.Error())
	}
}

func TestBuildTLSConfig_RejectTLS10(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "TLS1.0",
	})
	_, err := tm.BuildTLSConfig()
	if err == nil {
		t.Fatal("expected fatal error for TLS 1.0")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error = %q, want 'not supported'", err.Error())
	}
}

func TestBuildTLSConfig_RejectTLS11(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "TLS1.1",
	})
	_, err := tm.BuildTLSConfig()
	if err == nil {
		t.Fatal("expected fatal error for TLS 1.1")
	}
}

func TestBuildTLSConfig_AllowInsecureTrue_NoInsecureCiphers_NoWarning(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	suites := tls.CipherSuites()
	if len(suites) == 0 {
		t.Skip("no secure cipher suites available")
	}

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:             certFile,
		KeyFile:              keyFile,
		MinVersion:           "TLS1.2",
		CipherSuites:         []string{suites[0].Name},
		AllowInsecureCiphers: true, // Flag set but no insecure ciphers in list
	})
	_, err := tm.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tm.InsecureCipherNames()) != 0 {
		t.Error("expected no insecure cipher names when list is all safe")
	}
}

func TestBuildTLSConfig_ClientAuth(t *testing.T) {
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
		tlsCfg, err := tm.BuildTLSConfig()
		if err != nil {
			t.Fatalf("BuildTLSConfig(%q) unexpected error: %v", tt.clientAuth, err)
		}
		if tlsCfg.ClientAuth != tt.expected {
			t.Errorf("ClientAuth(%q) = %v, want %v", tt.clientAuth, tlsCfg.ClientAuth, tt.expected)
		}
	}
}

func TestBuildTLSConfig_VerifyHasClientCAs(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	tm, _ := NewTLSManager(config.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     certFile,
		ClientAuth: "verify",
	})
	tlsCfg, err := tm.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tlsCfg.ClientCAs == nil {
		t.Error("expected ClientCAs to be set for verify mode")
	}
}

// --- defaultSecureCipherSuites tests ---

func TestDefaultSecureCipherSuites(t *testing.T) {
	ids := defaultSecureCipherSuites()
	expected := tls.CipherSuites()
	if len(ids) != len(expected) {
		t.Errorf("expected %d suites, got %d", len(expected), len(ids))
	}
	for i, cs := range expected {
		if ids[i] != cs.ID {
			t.Errorf("suite[%d] = %d, want %d (%s)", i, ids[i], cs.ID, cs.Name)
		}
	}
}

// --- CreateServerTLSConfig tests ---

func TestCreateServerTLSConfig_Disabled(t *testing.T) {
	cfg, tm, err := CreateServerTLSConfig(config.TLSConfig{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config when TLS disabled")
	}
	if tm != nil {
		t.Error("expected nil TLSManager when TLS disabled")
	}
}

func TestCreateServerTLSConfig_Enabled(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	cfg, tm, err := CreateServerTLSConfig(config.TLSConfig{
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
	if tm == nil {
		t.Error("expected non-nil TLSManager")
	}
	// Default min version should be TLS 1.3
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3 default, got %d", cfg.MinVersion)
	}
}

func TestCreateServerTLSConfig_InvalidCert(t *testing.T) {
	_, _, err := CreateServerTLSConfig(config.TLSConfig{
		Enabled:  true,
		CertFile: "/bad/cert.pem",
		KeyFile:  "/bad/key.pem",
	})
	if err == nil {
		t.Error("expected error for invalid certs")
	}
}

func TestCreateServerTLSConfig_InvalidMinVersion(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	_, _, err := CreateServerTLSConfig(config.TLSConfig{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "TLS1.0",
	})
	if err == nil {
		t.Error("expected error for TLS 1.0")
	}
}

func TestCreateServerTLSConfig_InsecureCiphersRejected(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	insecureSuites := tls.InsecureCipherSuites()
	if len(insecureSuites) == 0 {
		t.Skip("no insecure cipher suites")
	}

	_, _, err := CreateServerTLSConfig(config.TLSConfig{
		Enabled:      true,
		CertFile:     certFile,
		KeyFile:      keyFile,
		MinVersion:   "TLS1.2",
		CipherSuites: []string{insecureSuites[0].Name},
	})
	if err == nil {
		t.Error("expected error for insecure ciphers without allow_insecure_ciphers")
	}
}

func TestCreateServerTLSConfig_ReloadViaTLSManager(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCert(t, dir)

	_, tm, err := CreateServerTLSConfig(config.TLSConfig{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  keyFile,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reload should succeed with the same cert files
	if err := tm.Reload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// Verify GetCertificate still works after reload
	cert, err := tm.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate failed after reload: %v", err)
	}
	if cert == nil {
		t.Error("expected non-nil certificate after reload")
	}
}

// --- CreateClientTLSConfig tests ---

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
