//go:build bdd

// Package bdd provides BDD tests using godog (Cucumber for Go).
// Run with: go test -tags bdd -v ./tests/bdd/...
package bdd

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

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
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
	"github.com/axonops/axonops-schema-registry/tests/bdd/steps"
)

// newTestServer creates a fresh in-process schema registry backed by memory storage.
func newTestServer() (*httptest.Server, storage.Storage) {
	store := memory.NewStore()

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaRegistry, compatChecker, "BACKWARD")

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := api.NewServer(cfg, reg, logger)

	return httptest.NewServer(server), store
}

func TestFeatures(t *testing.T) {
	// Default to excluding @operational scenarios (require Docker infrastructure).
	// Override with BDD_TAGS env var for Docker-based runs.
	tags := "~@operational"
	if envTags := os.Getenv("BDD_TAGS"); envTags != "" {
		tags = envTags
	}

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     tags,
		TestingT: t,
	}

	// Docker-based mode: use external registry URL instead of in-process server
	registryURL := os.Getenv("BDD_REGISTRY_URL")
	webhookURL := os.Getenv("BDD_WEBHOOK_URL")
	backend := os.Getenv("BDD_BACKEND")

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			var tc *steps.TestContext

			if registryURL != "" {
				// Docker-based: use external registry
				tc = steps.NewTestContext(registryURL)
				tc.WebhookURL = webhookURL
				tc.RegistryContainer = "bdd-schema-registry-1"
				if backend != "" {
					tc.BackendContainer = "bdd-" + backend + "-1"
				}
			} else {
				// In-process: create fresh server per scenario
				ts, store := newTestServer()
				tc = steps.NewTestContext(ts.URL)
				ctx.After(func(gctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
					ts.Close()
					store.Close()
					return gctx, nil
				})
			}

			// Register step definitions
			steps.RegisterSchemaSteps(ctx, tc)
			steps.RegisterImportSteps(ctx, tc)
			steps.RegisterModeSteps(ctx, tc)
			steps.RegisterReferenceSteps(ctx, tc)
			steps.RegisterInfraSteps(ctx, tc)
		},
		Options: &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("BDD tests failed")
	}
}
