// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// TLSManager manages TLS configuration with optional certificate reloading.
type TLSManager struct {
	config    config.TLSConfig
	mu        sync.RWMutex
	cert      *tls.Certificate
	clientCAs *x509.CertPool
}

// NewTLSManager creates a new TLS manager.
func NewTLSManager(cfg config.TLSConfig) (*TLSManager, error) {
	tm := &TLSManager{
		config: cfg,
	}

	if err := tm.loadCertificates(); err != nil {
		return nil, err
	}

	return tm, nil
}

// loadCertificates loads or reloads certificates from disk.
func (tm *TLSManager) loadCertificates() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(tm.config.CertFile, tm.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load server certificate: %w", err)
	}
	tm.cert = &cert

	// Load CA certificate for client verification if specified
	if tm.config.CAFile != "" {
		caCert, err := os.ReadFile(tm.config.CAFile)
		if err != nil {
			return fmt.Errorf("failed to load CA certificate: %w", err)
		}

		tm.clientCAs = x509.NewCertPool()
		if !tm.clientCAs.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA certificate")
		}
	}

	return nil
}

// Reload reloads certificates from disk.
func (tm *TLSManager) Reload() error {
	return tm.loadCertificates()
}

// GetCertificate returns the current certificate for TLS handshake.
func (tm *TLSManager) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.cert, nil
}

// TLSConfig returns the TLS configuration.
func (tm *TLSManager) TLSConfig() *tls.Config {
	// #nosec G402 -- MinVersion is configurable, defaults to TLS 1.2
	tlsConfig := &tls.Config{
		GetCertificate: tm.GetCertificate,
		MinVersion:     tm.getMinVersion(),
	}

	// Configure client authentication
	switch tm.config.ClientAuth {
	case "none", "":
		tlsConfig.ClientAuth = tls.NoClientCert
	case "request":
		tlsConfig.ClientAuth = tls.RequestClientCert
	case "require":
		tlsConfig.ClientAuth = tls.RequireAnyClientCert
	case "verify":
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tm.mu.RLock()
		tlsConfig.ClientCAs = tm.clientCAs
		tm.mu.RUnlock()
	}

	return tlsConfig
}

// getMinVersion returns the minimum TLS version.
func (tm *TLSManager) getMinVersion() uint16 {
	switch tm.config.MinVersion {
	case "TLS1.0", "1.0":
		return tls.VersionTLS10
	case "TLS1.1", "1.1":
		return tls.VersionTLS11
	case "TLS1.2", "1.2":
		return tls.VersionTLS12
	case "TLS1.3", "1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12 // Default to TLS 1.2
	}
}

// CreateServerTLSConfig creates a TLS config for an HTTPS server.
func CreateServerTLSConfig(cfg config.TLSConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	tm, err := NewTLSManager(cfg)
	if err != nil {
		return nil, err
	}

	return tm.TLSConfig(), nil
}

// CreateClientTLSConfig creates a TLS config for client connections.
func CreateClientTLSConfig(certFile, keyFile, caFile string, insecureSkipVerify bool) (*tls.Config, error) {
	// #nosec G402 -- InsecureSkipVerify is intentionally configurable for development/testing
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	// Load client certificate if specified
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate
	if caFile != "" {
		// #nosec G304 -- caFile is from trusted configuration
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
