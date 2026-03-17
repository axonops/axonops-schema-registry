package auth

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestNewSyslogOutput_EmptyAddress(t *testing.T) {
	_, err := NewSyslogOutput(config.AuditSyslogConfig{Enabled: true})
	if err == nil {
		t.Error("expected error for empty address")
	}
}

func TestSyslogOutput_Name(t *testing.T) {
	// Start a TCP listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	so, err := NewSyslogOutput(config.AuditSyslogConfig{
		Enabled: true,
		Address: ln.Addr().String(),
		Network: "tcp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer so.Close()

	if so.Name() != "syslog" {
		t.Errorf("expected name syslog, got %s", so.Name())
	}
}

func TestSyslogOutput_TCPWrite(t *testing.T) {
	// Start a TCP listener to receive syslog messages
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			received <- scanner.Text()
			return
		}
	}()

	so, err := NewSyslogOutput(config.AuditSyslogConfig{
		Enabled: true,
		Address: ln.Addr().String(),
		Network: "tcp",
		AppName: "test-registry",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer so.Close()

	testData := `{"event_type":"schema_register","outcome":"success"}`
	if err := so.Write([]byte(testData)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	select {
	case msg := <-received:
		if !strings.Contains(msg, "schema_register") {
			t.Errorf("expected message to contain schema_register, got: %s", msg)
		}
		if !strings.Contains(msg, "test-registry") {
			t.Errorf("expected message to contain app name, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for syslog message")
	}
}

func TestSyslogOutput_TLS(t *testing.T) {
	// Generate self-signed cert for testing
	certDir := t.TempDir()
	certFile, keyFile, caFile := generateTestCerts(t, certDir)

	// Load server TLS config
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	caCertPEM, _ := os.ReadFile(caFile)
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCertPEM)

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.NoClientCert,
		MinVersion:   tls.VersionTLS12,
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			received <- scanner.Text()
			return
		}
	}()

	so, err := NewSyslogOutput(config.AuditSyslogConfig{
		Enabled: true,
		Address: ln.Addr().String(),
		Network: "tcp+tls",
		TLSCA:   caFile,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer so.Close()

	testData := `{"event_type":"auth_failure","outcome":"failure"}`
	if err := so.Write([]byte(testData)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	select {
	case msg := <-received:
		if !strings.Contains(msg, "auth_failure") {
			t.Errorf("expected message to contain auth_failure, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for TLS syslog message")
	}
}

func TestParseFacility(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"local0", 128},
		{"local1", 136},
		{"auth", 32},
		{"daemon", 24},
		{"", 128},      // default
		{"bogus", 128}, // default
	}

	for _, tt := range tests {
		got := parseFacility(tt.name)
		if int(got) != tt.want {
			t.Errorf("parseFacility(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// generateTestCerts creates a self-signed CA + server cert for testing.
func generateTestCerts(t *testing.T, dir string) (certFile, keyFile, caFile string) {
	t.Helper()

	// Generate CA key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	// Generate server key
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	caCert, _ := x509.ParseCertificate(caCertDER)
	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	// Write CA cert
	caFile = filepath.Join(dir, "ca.pem")
	writePEM(t, caFile, "CERTIFICATE", caCertDER)

	// Write server cert
	certFile = filepath.Join(dir, "server.pem")
	writePEM(t, certFile, "CERTIFICATE", serverCertDER)

	// Write server key
	keyFile = filepath.Join(dir, "server-key.pem")
	keyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		t.Fatal(err)
	}
	writePEM(t, keyFile, "EC PRIVATE KEY", keyDER)

	return certFile, keyFile, caFile
}

func writePEM(t *testing.T, path, blockType string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: data}); err != nil {
		t.Fatal(err)
	}
}
