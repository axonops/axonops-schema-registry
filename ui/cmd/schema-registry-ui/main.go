// Package main is the entry point for the Schema Registry UI server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/axonops/schema-registry-ui/internal/config"
	"github.com/axonops/schema-registry-ui/internal/server"
	"github.com/axonops/schema-registry-ui/web"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "schema-registry-ui",
		Short: "AxonOps Schema Registry UI Server",
		Long:  "A standalone web UI for AxonOps Schema Registry with htpasswd-based authentication.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), configPath)
		},
		SilenceUsage: true,
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file (YAML)")

	cmd.AddCommand(versionCmd())

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("schema-registry-ui %s (commit: %s, built: %s)\n", version, commit, buildDate)
		},
	}
}

func run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Configure structured logging
	logLevel := slog.LevelInfo
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	slog.Info("starting schema-registry-ui",
		"version", version,
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"registry_url", cfg.Registry.URL,
	)

	// Get embedded SPA filesystem (nil in dev mode)
	spaFS := web.DistFS()
	if spaFS != nil {
		slog.Info("serving embedded SPA")
	} else {
		slog.Warn("no embedded SPA found — use vite dev server for frontend")
	}

	// Create server
	srv, err := server.New(cfg, spaFS)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
