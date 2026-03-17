// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/RackSec/srslog"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// SyslogOutput writes audit events to a syslog server (RFC 5424) over
// TCP, UDP, or TCP+TLS.
type SyslogOutput struct {
	writer *srslog.Writer
}

// NewSyslogOutput creates a syslog audit output from config.
// Defaults: network=tcp, appName=schema-registry, facility=local0.
func NewSyslogOutput(cfg config.AuditSyslogConfig) (*SyslogOutput, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("audit syslog address is required")
	}

	network := cfg.Network
	if network == "" {
		network = "tcp"
	}

	appName := cfg.AppName
	if appName == "" {
		appName = "schema-registry"
	}

	facility := parseFacility(cfg.Facility)

	var w *srslog.Writer
	var err error

	if network == "tcp+tls" {
		tlsCfg, tlsErr := buildSyslogTLSConfig(cfg)
		if tlsErr != nil {
			return nil, fmt.Errorf("syslog TLS config: %w", tlsErr)
		}
		w, err = srslog.DialWithTLSConfig("tcp+tls", cfg.Address, facility|srslog.LOG_INFO, appName, tlsCfg)
	} else {
		w, err = srslog.Dial(network, cfg.Address, facility|srslog.LOG_INFO, appName)
	}
	if err != nil {
		return nil, fmt.Errorf("syslog dial %s://%s: %w", network, cfg.Address, err)
	}

	// Use RFC 5424 format
	w.SetFormatter(srslog.RFC5424Formatter)
	w.SetFramer(srslog.RFC5425MessageLengthFramer)

	return &SyslogOutput{writer: w}, nil
}

// Write writes data to the syslog server.
func (o *SyslogOutput) Write(data []byte) error {
	// srslog.Writer.Write is goroutine-safe
	_, err := o.writer.Write(data)
	return err
}

// Close closes the syslog connection.
func (o *SyslogOutput) Close() error {
	return o.writer.Close()
}

// Name returns "syslog".
func (o *SyslogOutput) Name() string { return "syslog" }

// parseFacility converts a facility name string to a srslog.Priority value.
func parseFacility(name string) srslog.Priority {
	switch name {
	case "kern":
		return srslog.LOG_KERN
	case "user":
		return srslog.LOG_USER
	case "mail":
		return srslog.LOG_MAIL
	case "daemon":
		return srslog.LOG_DAEMON
	case "auth":
		return srslog.LOG_AUTH
	case "syslog":
		return srslog.LOG_SYSLOG
	case "lpr":
		return srslog.LOG_LPR
	case "news":
		return srslog.LOG_NEWS
	case "uucp":
		return srslog.LOG_UUCP
	case "cron":
		return srslog.LOG_CRON
	case "authpriv":
		return srslog.LOG_AUTHPRIV
	case "ftp":
		return srslog.LOG_FTP
	case "local1":
		return srslog.LOG_LOCAL1
	case "local2":
		return srslog.LOG_LOCAL2
	case "local3":
		return srslog.LOG_LOCAL3
	case "local4":
		return srslog.LOG_LOCAL4
	case "local5":
		return srslog.LOG_LOCAL5
	case "local6":
		return srslog.LOG_LOCAL6
	case "local7":
		return srslog.LOG_LOCAL7
	default:
		return srslog.LOG_LOCAL0
	}
}

// buildSyslogTLSConfig creates a TLS configuration for syslog connections.
func buildSyslogTLSConfig(cfg config.AuditSyslogConfig) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load client certificate if provided
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("loading client certificate: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if cfg.TLSCA != "" {
		// #nosec G304 -- path from config, user-controlled is expected
		caCert, err := os.ReadFile(cfg.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("reading CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsCfg.RootCAs = caCertPool
	}

	return tlsCfg, nil
}
