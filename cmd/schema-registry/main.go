// Package main is the entry point for the schema registry.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/api"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/kms"
	openbaokms "github.com/axonops/axonops-schema-registry/internal/kms/openbao"
	vaultkms "github.com/axonops/axonops-schema-registry/internal/kms/vault"
	mcpkg "github.com/axonops/axonops-schema-registry/internal/mcp"
	"github.com/axonops/axonops-schema-registry/internal/metrics"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/cassandra"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
	"github.com/axonops/axonops-schema-registry/internal/storage/mysql"
	"github.com/axonops/axonops-schema-registry/internal/storage/postgres"
	"github.com/axonops/axonops-schema-registry/internal/storage/vault"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	// Command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("axonops-schema-registry %s (commit: %s, built: %s)\n", version, commit, buildDate)
		os.Exit(0)
	}

	// Bootstrap logger (JSON, used until config is loaded).
	logLevel := slog.LevelInfo
	if os.Getenv("SCHEMA_REGISTRY_LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Reconfigure logger from config (format + level).
	switch cfg.Logging.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	var logHandler slog.Handler
	if cfg.Logging.Format == "text" {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	}
	logger = slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("starting schema registry",
		slog.String("version", version),
		slog.String("storage", cfg.Storage.Type),
		slog.String("address", cfg.Address()),
	)

	// Create metrics early so we can wrap storage with instrumentation
	m := metrics.New()

	// Create storage backend
	store, err := createStorage(cfg, logger)
	if err != nil {
		logger.Error("failed to create storage backend", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Wrap storage with instrumentation to record operation metrics
	instrumentedStore := storage.NewInstrumentedStorage(store, cfg.Storage.Type, m)

	// Create schema parser registry
	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	// Create compatibility checker
	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	// Create the registry service (uses instrumented storage for metrics)
	reg := registry.New(instrumentedStore, schemaRegistry, compatChecker, cfg.Compatibility.DefaultLevel)

	// Wire KMS provider registry for server-side DEK encryption.
	// Providers are only registered when their connection env vars are present.
	kmsReg := initKMSRegistry(logger)
	if kmsReg != nil {
		reg.SetKMSRegistry(kmsReg)
	}

	// Create server options
	var serverOpts []api.ServerOption
	serverOpts = append(serverOpts, api.WithBuildInfo(version, commit))
	serverOpts = append(serverOpts, api.WithMetrics(m))

	// Create audit logger if enabled
	var auditLogger *auth.AuditLogger
	if cfg.Security.Audit.Enabled {
		var err error
		auditLogger, err = auth.NewAuditLogger(cfg.Security.Audit)
		if err != nil {
			logger.Error("failed to create audit logger", slog.String("error", err.Error()))
			os.Exit(1)
		}
		auditLogger.SetMetrics(m)
		serverOpts = append(serverOpts, api.WithAuditLogger(auditLogger))
		logger.Info("audit logging enabled",
			slog.Bool("stdout", cfg.Security.Audit.Outputs.Stdout.Enabled),
			slog.Bool("file", cfg.Security.Audit.Outputs.File.Enabled || cfg.Security.Audit.LogFile != ""),
			slog.Bool("syslog", cfg.Security.Audit.Outputs.Syslog.Enabled),
			slog.Bool("webhook", cfg.Security.Audit.Outputs.Webhook.Enabled),
			slog.Bool("include_body", cfg.Security.Audit.IncludeBody),
		)
	}

	var authService *auth.Service
	var vaultStore *vault.Store

	// Setup authentication if enabled
	if cfg.Security.Auth.Enabled {
		logger.Info("authentication enabled", slog.Any("methods", cfg.Security.Auth.Methods))

		// Create authenticator and authorizer
		authenticator := auth.NewAuthenticator(cfg.Security.Auth)
		authorizer := auth.NewAuthorizer(cfg.Security.Auth.RBAC)

		// Determine which auth storage backend to use
		var authStorage storage.AuthStorage
		authType := cfg.Storage.AuthType
		if authType == "" {
			// Default: use the same storage backend for auth
			authType = cfg.Storage.Type
		}

		switch authType {
		case "vault":
			logger.Info("using Vault for authentication storage",
				slog.String("address", cfg.Storage.Vault.Address),
				slog.String("mount_path", cfg.Storage.Vault.MountPath),
				slog.String("base_path", cfg.Storage.Vault.BasePath),
			)
			vaultCfg := vault.Config{
				Address:       cfg.Storage.Vault.Address,
				Token:         cfg.Storage.Vault.Token,
				Namespace:     cfg.Storage.Vault.Namespace,
				MountPath:     cfg.Storage.Vault.MountPath,
				BasePath:      cfg.Storage.Vault.BasePath,
				TLSCertFile:   cfg.Storage.Vault.TLSCertFile,
				TLSKeyFile:    cfg.Storage.Vault.TLSKeyFile,
				TLSCAFile:     cfg.Storage.Vault.TLSCAFile,
				TLSSkipVerify: cfg.Storage.Vault.TLSSkipVerify,
			}
			var err error
			vaultStore, err = vault.NewStore(vaultCfg)
			if err != nil {
				logger.Error("failed to connect to Vault", slog.String("error", err.Error()))
				os.Exit(1)
			}
			authStorage = vaultStore
		default:
			// Use the main storage backend for auth (database-backed)
			authStorage = store
		}

		// Create auth service with secure API key configuration.
		// UserCacheTTL defaults to 60s to reduce database load for frequently
		// authenticating users. CacheRefreshInterval ensures cluster consistency.
		authService = auth.NewServiceWithConfig(authStorage, auth.ServiceConfig{
			APIKeySecret:         cfg.Security.Auth.APIKey.Secret,
			APIKeyPrefix:         cfg.Security.Auth.APIKey.KeyPrefix,
			CacheRefreshInterval: time.Duration(cfg.Security.Auth.APIKey.CacheRefreshSeconds) * time.Second,
			UserCacheTTL:         auth.DefaultUserCacheTTL,
		})

		// Wire metrics to auth service for cache metrics
		authService.SetMetrics(m)

		// Wire the service to the authenticator for database-backed auth
		authenticator.SetService(authService)

		// Bootstrap initial admin user if enabled
		if cfg.Security.Auth.Bootstrap.Enabled {
			logger.Info("bootstrap enabled, checking for initial admin user")
			result, err := authService.BootstrapAdmin(
				context.Background(),
				cfg.Security.Auth.Bootstrap.Username,
				cfg.Security.Auth.Bootstrap.Password,
				cfg.Security.Auth.Bootstrap.Email,
			)
			if err != nil {
				logger.Error("failed to bootstrap admin user", slog.String("error", err.Error()))
				os.Exit(1)
			}
			if result.Created {
				logger.Info("bootstrap admin user created",
					slog.String("username", result.Username),
					slog.String("role", "super_admin"),
				)
			} else {
				logger.Info("bootstrap skipped", slog.String("reason", result.Message))
			}
		}

		// Setup LDAP provider if enabled
		if cfg.Security.Auth.LDAP.Enabled {
			logger.Info("LDAP authentication enabled",
				slog.String("url", cfg.Security.Auth.LDAP.URL),
				slog.String("user_search_base", cfg.Security.Auth.LDAP.UserSearchBase),
				slog.Bool("tls", strings.HasPrefix(cfg.Security.Auth.LDAP.URL, "ldaps://") || cfg.Security.Auth.LDAP.StartTLS),
				slog.Bool("mtls", cfg.Security.Auth.LDAP.ClientCertFile != ""),
			)
			ldapProvider, err := auth.NewLDAPProvider(cfg.Security.Auth.LDAP)
			if err != nil {
				logger.Error("failed to create LDAP provider", slog.String("error", err.Error()))
				os.Exit(1)
			}
			authenticator.SetLDAPProvider(ldapProvider)
			if auditLogger != nil {
				authenticator.SetAuditLogger(auditLogger)
			}

			// Warn if LDAP fallback to DB/htpasswd is enabled (default)
			if cfg.Security.Auth.LDAP.AllowFallback == nil || *cfg.Security.Auth.LDAP.AllowFallback {
				logger.Warn("LDAP allow_fallback is enabled — users not found in LDAP will be "+
					"tried against database/htpasswd users. Users that exist in LDAP but provide "+
					"wrong passwords are always rejected (no fallback). "+
					"Set allow_fallback: false for strict LDAP-only auth.",
					slog.String("setting", "security.auth.ldap.allow_fallback"),
					slog.Bool("current_value", true),
				)
			}

			// Emit audit event if LDAP is configured without TLS
			if auditLogger != nil && !ldapProvider.IsSecure() {
				auditLogger.Log(&auth.AuditEvent{
					EventType:  auth.AuditEventSecurityWarning,
					Timestamp:  time.Now(),
					Method:     "STARTUP",
					ActorID:    "system",
					ActorType:  "system",
					Outcome:    "warning",
					TargetType: "config",
					TargetID:   "security.auth.ldap",
					Reason:     "ldap_no_tls",
				})
			}
		}

		// Setup OIDC provider if enabled
		if cfg.Security.Auth.OIDC.Enabled {
			logger.Info("OIDC authentication enabled",
				slog.String("issuer_url", cfg.Security.Auth.OIDC.IssuerURL),
				slog.String("client_id", cfg.Security.Auth.OIDC.ClientID),
			)
			oidcProvider, err := auth.NewOIDCProvider(context.Background(), cfg.Security.Auth.OIDC)
			if err != nil {
				logger.Error("failed to create OIDC provider", slog.String("error", err.Error()))
				os.Exit(1)
			}
			authenticator.SetOIDCProvider(oidcProvider)
		}

		// Setup JWT provider if configured
		if cfg.Security.Auth.JWT.PublicKeyFile != "" || cfg.Security.Auth.JWT.JWKSURL != "" {
			logger.Info("JWT authentication enabled",
				slog.String("algorithm", cfg.Security.Auth.JWT.Algorithm),
				slog.String("issuer", cfg.Security.Auth.JWT.Issuer),
			)
			jwtProvider, err := auth.NewJWTProvider(cfg.Security.Auth.JWT)
			if err != nil {
				logger.Error("failed to create JWT provider", slog.String("error", err.Error()))
				os.Exit(1)
			}
			authenticator.SetJWTProvider(jwtProvider)
		}

		// Load config-defined API keys if storage_type is "memory"
		if strings.EqualFold(cfg.Security.Auth.APIKey.StorageType, "memory") {
			memAPIKeys, err := auth.NewMemoryAPIKeyStore(cfg.Security.Auth.APIKey.Keys)
			if err != nil {
				logger.Error("failed to load config-defined API keys", slog.String("error", err.Error()))
				os.Exit(1)
			}
			authenticator.SetMemoryAPIKeyStore(memAPIKeys)
			logger.Info("config-defined API keys loaded", slog.Int("keys", memAPIKeys.Count()))
		}

		// Load htpasswd file if configured
		if cfg.Security.Auth.Basic.HTPasswd != "" {
			htpasswdStore, err := auth.LoadHTPasswdFile(cfg.Security.Auth.Basic.HTPasswd)
			if err != nil {
				logger.Error("failed to load htpasswd file", slog.String("error", err.Error()))
				os.Exit(1)
			}
			authenticator.SetHTPasswdStore(htpasswdStore)
			logger.Info("htpasswd file loaded", slog.Int("entries", htpasswdStore.Count()))
		}

		// Add auth option
		serverOpts = append(serverOpts, api.WithAuth(authenticator, authorizer, authService))
	}

	// Create rate limiter if enabled
	var rateLimiter *auth.RateLimiter
	if cfg.Security.RateLimiting.Enabled {
		rateLimiter = auth.NewRateLimiter(cfg.Security.RateLimiting)
		serverOpts = append(serverOpts, api.WithRateLimiter(rateLimiter))
		logger.Info("rate limiting enabled",
			slog.Int("requests_per_second", cfg.Security.RateLimiting.RequestsPerSecond),
			slog.Int("burst_size", cfg.Security.RateLimiting.BurstSize),
		)
	}

	// Create and start the HTTP server
	server := api.NewServer(cfg, reg, logger, serverOpts...)

	// Enable per-principal metrics if configured (default: enabled).
	if cfg.Security.Metrics.PerPrincipalMetrics == nil || *cfg.Security.Metrics.PerPrincipalMetrics {
		server.Metrics().EnablePrincipalMetrics()
		logger.Info("per-principal metrics enabled")
	}

	// Start periodic gauge metrics refresh (schemas_total, subjects_total).
	gaugeRefreshInterval := time.Duration(cfg.Server.MetricsRefreshInterval) * time.Second
	if gaugeRefreshInterval <= 0 {
		gaugeRefreshInterval = 5 * time.Minute
	}
	gaugeStop := make(chan struct{})
	m.StartGaugeRefresh(reg, gaugeRefreshInterval, gaugeStop)
	logger.Info("gauge metrics refresh started", slog.Duration("interval", gaugeRefreshInterval))

	// Create and start the MCP server if enabled
	var mcpServer *mcpkg.Server
	if cfg.MCP.Enabled {
		var mcpOpts []mcpkg.Option
		if authService != nil {
			mcpOpts = append(mcpOpts, mcpkg.WithAuthService(authService))
		}
		mcpOpts = append(mcpOpts, mcpkg.WithBuildInfo(commit, buildDate))
		if cfg.Server.ClusterID != "" {
			mcpOpts = append(mcpOpts, mcpkg.WithClusterID(cfg.Server.ClusterID))
		}
		mcpOpts = append(mcpOpts, mcpkg.WithMetrics(server.Metrics()))
		if auditLogger != nil {
			mcpOpts = append(mcpOpts, mcpkg.WithAuditLogger(auditLogger))
		}
		mcpServer = mcpkg.New(&cfg.MCP, reg, logger, version, mcpOpts...)
	}

	// Handle shutdown and reload signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGHUP)

	// Handle SIGHUP for TLS certificate reload
	go func() {
		for range reload {
			logger.Info("received SIGHUP, reloading TLS certificates")
			if err := server.ReloadTLS(); err != nil {
				logger.Error("TLS certificate reload failed", slog.String("error", err.Error()))
			} else {
				logger.Info("TLS certificates reloaded successfully")
			}
		}
	}()

	// Start servers in goroutines — both feed into the same error channel.
	serverErr := make(chan error, 2)
	go func() {
		serverErr <- server.Start()
	}()
	if mcpServer != nil {
		go func() {
			if err := mcpServer.Start(); err != nil && err != http.ErrServerClosed {
				serverErr <- fmt.Errorf("MCP server: %w", err)
			}
		}()
	}

	// Wait for shutdown signal or error
	select {
	case err := <-serverErr:
		if err != nil {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case sig := <-shutdown:
		logger.Info("shutting down", slog.String("signal", sig.String()))

		shutdownTimeout := time.Duration(cfg.Server.ShutdownTimeout) * time.Second
		if shutdownTimeout <= 0 {
			shutdownTimeout = 30 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		close(gaugeStop)

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", slog.String("error", err.Error()))
		}

		// Stop MCP server
		if mcpServer != nil {
			if err := mcpServer.Shutdown(ctx); err != nil {
				logger.Error("MCP shutdown error", slog.String("error", err.Error()))
			}
		}

		// Close audit logger
		if auditLogger != nil {
			if err := auditLogger.Close(); err != nil {
				logger.Error("audit logger close error", slog.String("error", err.Error()))
			}
		}

		// Stop rate limiter cleanup goroutine
		if rateLimiter != nil {
			rateLimiter.Close()
		}

		// Stop auth service background goroutines
		if authService != nil {
			authService.Close()
		}

		// Close KMS providers
		if kmsReg != nil {
			if err := kmsReg.Close(); err != nil {
				logger.Error("kms registry close error", slog.String("error", err.Error()))
			}
		}

		// Close Vault store if used
		if vaultStore != nil {
			if err := vaultStore.Close(); err != nil {
				logger.Error("vault store close error", slog.String("error", err.Error()))
			}
		}

		if err := store.Close(); err != nil {
			logger.Error("storage close error", slog.String("error", err.Error()))
		}
	}

	logger.Info("shutdown complete")
}

// createStorage creates the appropriate storage backend based on configuration.
func createStorage(cfg *config.Config, logger *slog.Logger) (storage.Storage, error) {
	switch cfg.Storage.Type {
	case "memory":
		logger.Info("using in-memory storage")
		return memory.NewStore(), nil

	case "postgresql", "postgres":
		logger.Info("connecting to PostgreSQL",
			slog.String("host", cfg.Storage.PostgreSQL.Host),
			slog.Int("port", cfg.Storage.PostgreSQL.Port),
			slog.String("database", cfg.Storage.PostgreSQL.Database),
		)
		pgCfg := postgres.Config{
			Host:               cfg.Storage.PostgreSQL.Host,
			Port:               cfg.Storage.PostgreSQL.Port,
			Database:           cfg.Storage.PostgreSQL.Database,
			Username:           cfg.Storage.PostgreSQL.User,
			Password:           cfg.Storage.PostgreSQL.Password,
			SSLMode:            cfg.Storage.PostgreSQL.SSLMode,
			MaxOpenConns:       cfg.Storage.PostgreSQL.MaxOpenConns,
			MaxIdleConns:       cfg.Storage.PostgreSQL.MaxIdleConns,
			ConnMaxLifetime:    time.Duration(cfg.Storage.PostgreSQL.ConnMaxLifetime) * time.Second,
			ConnectTimeout:     time.Duration(cfg.Storage.PostgreSQL.ConnectTimeout) * time.Second,
			HealthCheckTimeout: time.Duration(cfg.Storage.PostgreSQL.HealthCheckTimeout) * time.Second,
			SchemaMaxRetries:   cfg.Storage.PostgreSQL.SchemaMaxRetries,
		}
		if pgCfg.Host == "" {
			pgCfg.Host = "localhost"
		}
		if pgCfg.Port == 0 {
			pgCfg.Port = 5432
		}
		if pgCfg.Database == "" {
			pgCfg.Database = "schema_registry"
		}
		if pgCfg.SSLMode == "" {
			pgCfg.SSLMode = "disable"
		}
		if pgCfg.MaxOpenConns == 0 {
			pgCfg.MaxOpenConns = 25
		}
		if pgCfg.MaxIdleConns == 0 {
			pgCfg.MaxIdleConns = 5
		}
		if pgCfg.ConnMaxLifetime == 0 {
			pgCfg.ConnMaxLifetime = 5 * time.Minute
		}
		return postgres.NewStore(pgCfg)

	case "mysql":
		logger.Info("connecting to MySQL",
			slog.String("host", cfg.Storage.MySQL.Host),
			slog.Int("port", cfg.Storage.MySQL.Port),
			slog.String("database", cfg.Storage.MySQL.Database),
		)
		mysqlCfg := mysql.Config{
			Host:               cfg.Storage.MySQL.Host,
			Port:               cfg.Storage.MySQL.Port,
			Database:           cfg.Storage.MySQL.Database,
			Username:           cfg.Storage.MySQL.User,
			Password:           cfg.Storage.MySQL.Password,
			TLS:                cfg.Storage.MySQL.TLS,
			MaxOpenConns:       cfg.Storage.MySQL.MaxOpenConns,
			MaxIdleConns:       cfg.Storage.MySQL.MaxIdleConns,
			ConnMaxLifetime:    time.Duration(cfg.Storage.MySQL.ConnMaxLifetime) * time.Second,
			ConnectTimeout:     time.Duration(cfg.Storage.MySQL.ConnectTimeout) * time.Second,
			HealthCheckTimeout: time.Duration(cfg.Storage.MySQL.HealthCheckTimeout) * time.Second,
			SchemaMaxRetries:   cfg.Storage.MySQL.SchemaMaxRetries,
		}
		if mysqlCfg.Host == "" {
			mysqlCfg.Host = "localhost"
		}
		if mysqlCfg.Port == 0 {
			mysqlCfg.Port = 3306
		}
		if mysqlCfg.Database == "" {
			mysqlCfg.Database = "schema_registry"
		}
		if mysqlCfg.TLS == "" {
			mysqlCfg.TLS = "false"
		}
		if mysqlCfg.MaxOpenConns == 0 {
			mysqlCfg.MaxOpenConns = 25
		}
		if mysqlCfg.MaxIdleConns == 0 {
			mysqlCfg.MaxIdleConns = 5
		}
		if mysqlCfg.ConnMaxLifetime == 0 {
			mysqlCfg.ConnMaxLifetime = 5 * time.Minute
		}
		return mysql.NewStore(mysqlCfg)

	case "cassandra":
		logger.Info("connecting to Cassandra",
			slog.Any("hosts", cfg.Storage.Cassandra.Hosts),
			slog.String("keyspace", cfg.Storage.Cassandra.Keyspace),
			slog.String("local_dc", cfg.Storage.Cassandra.LocalDC),
		)
		cassCfg := cassandra.Config{
			Hosts:             cfg.Storage.Cassandra.Hosts,
			Port:              cfg.Storage.Cassandra.Port,
			Keyspace:          cfg.Storage.Cassandra.Keyspace,
			LocalDC:           cfg.Storage.Cassandra.LocalDC,
			Username:          cfg.Storage.Cassandra.Username,
			Password:          cfg.Storage.Cassandra.Password,
			Consistency:       cfg.Storage.Cassandra.Consistency,
			ReadConsistency:   cfg.Storage.Cassandra.ReadConsistency,
			WriteConsistency:  cfg.Storage.Cassandra.WriteConsistency,
			SerialConsistency: cfg.Storage.Cassandra.SerialConsistency,
			MaxRetries:        cfg.Storage.Cassandra.MaxRetries,
			IDBlockSize:       cfg.Storage.Cassandra.IDBlockSize,
			Migrate:           true,
		}
		if cfg.Storage.Cassandra.Timeout != "" {
			d, err := time.ParseDuration(cfg.Storage.Cassandra.Timeout)
			if err != nil {
				return nil, fmt.Errorf("invalid cassandra timeout %q: %w", cfg.Storage.Cassandra.Timeout, err)
			}
			cassCfg.Timeout = d
		}
		if cfg.Storage.Cassandra.ConnectTimeout != "" {
			d, err := time.ParseDuration(cfg.Storage.Cassandra.ConnectTimeout)
			if err != nil {
				return nil, fmt.Errorf("invalid cassandra connect_timeout %q: %w", cfg.Storage.Cassandra.ConnectTimeout, err)
			}
			cassCfg.ConnectTimeout = d
		}
		if len(cassCfg.Hosts) == 0 {
			cassCfg.Hosts = []string{"localhost"}
		}
		if cassCfg.Keyspace == "" {
			cassCfg.Keyspace = "schema_registry"
		}
		if cassCfg.Consistency == "" {
			cassCfg.Consistency = "LOCAL_QUORUM"
		}
		return cassandra.NewStore(context.Background(), cassCfg)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}

// initKMSRegistry creates a KMS provider registry with available providers.
// Providers are only registered when their connection environment variables
// (e.g., VAULT_ADDR/VAULT_TOKEN, BAO_ADDR/BAO_TOKEN) are set.
// Returns nil if no providers are available.
func initKMSRegistry(logger *slog.Logger) *kms.Registry {
	reg := kms.NewRegistry()
	registered := 0

	// HashiCorp Vault Transit
	if os.Getenv("VAULT_ADDR") != "" {
		p, err := vaultkms.NewProvider(vaultkms.Config{})
		if err != nil {
			logger.Warn("failed to create Vault KMS provider", slog.String("error", err.Error()))
		} else {
			if err := reg.Register(p); err != nil {
				logger.Warn("failed to register Vault KMS provider", slog.String("error", err.Error()))
			} else {
				logger.Info("KMS provider registered", slog.String("type", "hcvault"), slog.String("address", os.Getenv("VAULT_ADDR")))
				registered++
			}
		}
	}

	// OpenBao Transit
	if os.Getenv("BAO_ADDR") != "" {
		p, err := openbaokms.NewProvider(vaultkms.Config{})
		if err != nil {
			logger.Warn("failed to create OpenBao KMS provider", slog.String("error", err.Error()))
		} else {
			if err := reg.Register(p); err != nil {
				logger.Warn("failed to register OpenBao KMS provider", slog.String("error", err.Error()))
			} else {
				logger.Info("KMS provider registered", slog.String("type", "openbao"), slog.String("address", os.Getenv("BAO_ADDR")))
				registered++
			}
		}
	}

	if registered == 0 {
		return nil
	}
	return reg
}
