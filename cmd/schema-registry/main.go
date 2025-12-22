// Package main is the entry point for the schema registry.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/api"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
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

	// Setup logger
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

	logger.Info("starting schema registry",
		slog.String("version", version),
		slog.String("storage", cfg.Storage.Type),
		slog.String("address", cfg.Address()),
	)

	// Create storage backend
	store, err := createStorage(cfg, logger)
	if err != nil {
		logger.Error("failed to create storage backend", slog.String("error", err.Error()))
		os.Exit(1)
	}

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

	// Create the registry service
	reg := registry.New(store, schemaRegistry, compatChecker, cfg.Compatibility.DefaultLevel)

	// Create and start the HTTP server
	server := api.NewServer(cfg, reg, logger)

	// Handle shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-serverErr:
		if err != nil {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case sig := <-shutdown:
		logger.Info("shutting down", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", slog.String("error", err.Error()))
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
			Host:            cfg.Storage.PostgreSQL.Host,
			Port:            cfg.Storage.PostgreSQL.Port,
			Database:        cfg.Storage.PostgreSQL.Database,
			Username:        cfg.Storage.PostgreSQL.User,
			Password:        cfg.Storage.PostgreSQL.Password,
			SSLMode:         cfg.Storage.PostgreSQL.SSLMode,
			MaxOpenConns:    cfg.Storage.PostgreSQL.MaxOpenConns,
			MaxIdleConns:    cfg.Storage.PostgreSQL.MaxIdleConns,
			ConnMaxLifetime: time.Duration(cfg.Storage.PostgreSQL.ConnMaxLifetime) * time.Second,
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
			Host:            cfg.Storage.MySQL.Host,
			Port:            cfg.Storage.MySQL.Port,
			Database:        cfg.Storage.MySQL.Database,
			Username:        cfg.Storage.MySQL.User,
			Password:        cfg.Storage.MySQL.Password,
			TLS:             cfg.Storage.MySQL.TLS,
			MaxOpenConns:    cfg.Storage.MySQL.MaxOpenConns,
			MaxIdleConns:    cfg.Storage.MySQL.MaxIdleConns,
			ConnMaxLifetime: time.Duration(cfg.Storage.MySQL.ConnMaxLifetime) * time.Second,
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
		)
		cassCfg := cassandra.Config{
			Hosts:               cfg.Storage.Cassandra.Hosts,
			Keyspace:            cfg.Storage.Cassandra.Keyspace,
			Username:            cfg.Storage.Cassandra.Username,
			Password:            cfg.Storage.Cassandra.Password,
			Consistency:         cfg.Storage.Cassandra.Consistency,
			ReplicationStrategy: "SimpleStrategy",
			ReplicationFactor:   1,
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
		return cassandra.NewStore(cassCfg)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}
