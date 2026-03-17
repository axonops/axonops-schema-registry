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
	config              config.TLSConfig
	mu                  sync.RWMutex
	cert                *tls.Certificate
	clientCAs           *x509.CertPool
	insecureCipherNames []string // populated during BuildTLSConfig(), read by caller for logging/audit
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

// InsecureCipherNames returns the names of insecure ciphers that were explicitly allowed.
// Empty if no insecure ciphers are configured. Populated after BuildTLSConfig() is called.
func (tm *TLSManager) InsecureCipherNames() []string {
	return tm.insecureCipherNames
}

// BuildTLSConfig builds and validates the TLS configuration.
// Returns an error if the configuration is invalid (unsupported TLS version,
// insecure ciphers without allow_insecure_ciphers, unknown cipher names).
func (tm *TLSManager) BuildTLSConfig() (*tls.Config, error) {
	minVersion, err := parseMinVersion(tm.config.MinVersion)
	if err != nil {
		return nil, err
	}

	// #nosec G402 -- MinVersion is validated above, minimum TLS 1.2
	tlsConfig := &tls.Config{
		GetCertificate: tm.GetCertificate,
		MinVersion:     minVersion,
	}

	// Cipher suites only matter for TLS 1.2 connections.
	// TLS 1.3 cipher suites are not configurable in Go — the runtime
	// always uses only AEAD suites (AES-GCM, ChaCha20-Poly1305).
	if minVersion <= tls.VersionTLS12 {
		if len(tm.config.CipherSuites) > 0 {
			ids, insecureNames, resolveErr := resolveCipherSuites(tm.config.CipherSuites)
			if resolveErr != nil {
				return nil, fmt.Errorf("invalid cipher_suites configuration: %w", resolveErr)
			}
			if len(insecureNames) > 0 && !tm.config.AllowInsecureCiphers {
				return nil, fmt.Errorf("refusing to start: cipher_suites contains insecure ciphers %v — set security.tls.allow_insecure_ciphers: true to override", insecureNames)
			}
			tm.insecureCipherNames = insecureNames
			tlsConfig.CipherSuites = ids
		} else {
			// Default: use Go's safe cipher suites (tls.CipherSuites())
			tlsConfig.CipherSuites = defaultSecureCipherSuites()
		}
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

	return tlsConfig, nil
}

// parseMinVersion validates and returns the minimum TLS version.
// Returns an error for TLS versions below 1.2 or unrecognized values.
func parseMinVersion(v string) (uint16, error) {
	switch v {
	case "TLS1.2", "1.2":
		return tls.VersionTLS12, nil
	case "TLS1.3", "1.3", "":
		return tls.VersionTLS13, nil // Default: TLS 1.3
	case "TLS1.0", "1.0", "TLS1.1", "1.1":
		return 0, fmt.Errorf("TLS versions below 1.2 are not supported (configured: %q)", v)
	default:
		return 0, fmt.Errorf("unrecognized min_version: %q (supported values: TLS1.2, TLS1.3)", v)
	}
}

// defaultSecureCipherSuites returns the IDs of all cipher suites that Go
// considers secure (i.e. tls.CipherSuites(), excluding tls.InsecureCipherSuites()).
func defaultSecureCipherSuites() []uint16 {
	suites := tls.CipherSuites()
	ids := make([]uint16, len(suites))
	for i, cs := range suites {
		ids[i] = cs.ID
	}
	return ids
}

// resolveCipherSuites resolves configured cipher suite names to IDs and
// identifies any that appear in tls.InsecureCipherSuites().
func resolveCipherSuites(names []string) ([]uint16, []string, error) {
	// Build lookup from both secure and insecure suites.
	lookup := make(map[string]uint16)
	for _, cs := range tls.CipherSuites() {
		lookup[cs.Name] = cs.ID
	}
	insecureSet := make(map[uint16]string)
	for _, cs := range tls.InsecureCipherSuites() {
		lookup[cs.Name] = cs.ID
		insecureSet[cs.ID] = cs.Name
	}

	var ids []uint16
	var insecureNames []string
	for _, name := range names {
		id, ok := lookup[name]
		if !ok {
			return nil, nil, fmt.Errorf("unknown cipher suite: %q", name)
		}
		ids = append(ids, id)
		if insecureName, isInsecure := insecureSet[id]; isInsecure {
			insecureNames = append(insecureNames, insecureName)
		}
	}
	return ids, insecureNames, nil
}

// CreateServerTLSConfig creates a TLS config for an HTTPS server.
// It returns both the tls.Config and the TLSManager so the caller can
// trigger certificate reloads (e.g., on SIGHUP).
func CreateServerTLSConfig(cfg config.TLSConfig) (*tls.Config, *TLSManager, error) {
	if !cfg.Enabled {
		return nil, nil, nil
	}

	tm, err := NewTLSManager(cfg)
	if err != nil {
		return nil, nil, err
	}

	tlsCfg, err := tm.BuildTLSConfig()
	if err != nil {
		return nil, nil, err
	}

	return tlsCfg, tm, nil
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
