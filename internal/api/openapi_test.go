package api

import (
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	openapispec "github.com/axonops/axonops-schema-registry/api"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

// openAPIDocument is a minimal representation of the OpenAPI spec for path extraction.
type openAPIDocument struct {
	Paths map[string]map[string]interface{} `yaml:"paths"`
}

// setupFullServer creates a server with all routes registered (including auth-conditional
// routes like /admin/* and /me/*) and docs enabled.
func setupFullServer(t *testing.T) *Server {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Server.DocsEnabled = true

	store := memory.NewStore()

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())

	reg := registry.New(store, schemaRegistry, compatChecker, cfg.Compatibility.DefaultLevel)

	// Create auth service and authorizer so admin/account routes are registered.
	authService := auth.NewService(store)
	t.Cleanup(func() { authService.Close() })
	authorizer := auth.NewAuthorizer(config.RBACConfig{Enabled: true, DefaultRole: "readonly"})

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewServer(cfg, reg, logger, WithAuth(nil, authorizer, authService))
}

// normalizeRoute removes trailing slashes from routes (except root "/").
func normalizeRoute(route string) string {
	if route == "/" {
		return route
	}
	return strings.TrimRight(route, "/")
}

// routeKey creates a comparable key from method and path.
func routeKey(method, path string) string {
	return method + " " + path
}

// getRouterRoutes walks the chi router and returns all registered method+path pairs.
func getRouterRoutes(t *testing.T, router chi.Routes) map[string]bool {
	t.Helper()

	routes := make(map[string]bool)
	err := chi.Walk(router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		normalized := normalizeRoute(route)
		routes[routeKey(method, normalized)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk failed: %v", err)
	}
	return routes
}

// getSpecRoutes parses the embedded OpenAPI spec and returns all method+path pairs.
func getSpecRoutes(t *testing.T) map[string]bool {
	t.Helper()

	var doc openAPIDocument
	if err := yaml.Unmarshal(openapispec.OpenAPISpec, &doc); err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	routes := make(map[string]bool)
	for path, methods := range doc.Paths {
		for method := range methods {
			upper := strings.ToUpper(method)
			// Skip non-HTTP-method keys (e.g. "parameters", "summary")
			switch upper {
			case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
				routes[routeKey(upper, path)] = true
			}
		}
	}
	return routes
}

// routesToSortedSlice converts a route set to a sorted slice for readable output.
func routesToSortedSlice(routes map[string]bool) []string {
	result := make([]string, 0, len(routes))
	for r := range routes {
		result = append(result, r)
	}
	sort.Strings(result)
	return result
}

// TestOpenAPISpecMatchesRoutes validates that every chi route exists in the OpenAPI spec
// and every OpenAPI spec path exists as a chi route. This prevents spec drift in either
// direction.
func TestOpenAPISpecMatchesRoutes(t *testing.T) {
	server := setupFullServer(t)
	router := server.router

	routerRoutes := getRouterRoutes(t, router)
	specRoutes := getSpecRoutes(t)

	// Routes that are intentionally excluded from the spec (internal/infrastructure).
	// Context-scoped routes (/contexts/{context}/...) mirror the root-level registry
	// routes exactly and are excluded from the sync test. They will be documented
	// separately in the OpenAPI spec when context support documentation is added.
	specExclusions := map[string]bool{}
	for route := range routerRoutes {
		if strings.Contains(route, "/contexts/{context}/") {
			specExclusions[route] = true
		}
	}

	// Routes that exist in the spec but may not exist in the router due to
	// conditional registration (e.g. docs routes when DocsEnabled=false).
	// Since we create the server with DocsEnabled=true and auth enabled,
	// all routes should be present.
	routerExclusions := map[string]bool{}

	t.Run("every router route exists in OpenAPI spec", func(t *testing.T) {
		var missing []string
		for route := range routerRoutes {
			if specExclusions[route] {
				continue
			}
			if !specRoutes[route] {
				missing = append(missing, route)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("Routes registered in router but missing from OpenAPI spec:\n  %s",
				strings.Join(missing, "\n  "))
		}
	})

	t.Run("every OpenAPI spec route exists in router", func(t *testing.T) {
		var missing []string
		for route := range specRoutes {
			if routerExclusions[route] {
				continue
			}
			if !routerRoutes[route] {
				missing = append(missing, route)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("Routes in OpenAPI spec but not registered in router:\n  %s",
				strings.Join(missing, "\n  "))
		}
	})

	t.Run("route counts match", func(t *testing.T) {
		routerFiltered := 0
		for r := range routerRoutes {
			if !specExclusions[r] {
				routerFiltered++
			}
		}
		specFiltered := 0
		for r := range specRoutes {
			if !routerExclusions[r] {
				specFiltered++
			}
		}
		t.Logf("Router routes: %d, Spec routes: %d", routerFiltered, specFiltered)

		if routerFiltered != specFiltered {
			t.Logf("Router routes:\n  %s", strings.Join(routesToSortedSlice(routerRoutes), "\n  "))
			t.Logf("Spec routes:\n  %s", strings.Join(routesToSortedSlice(specRoutes), "\n  "))
		}
	})
}

// TestOpenAPISpecIsValidYAML ensures the embedded spec is valid YAML.
func TestOpenAPISpecIsValidYAML(t *testing.T) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(openapispec.OpenAPISpec, &doc); err != nil {
		t.Fatalf("OpenAPI spec is not valid YAML: %v", err)
	}

	if doc["openapi"] == nil {
		t.Error("OpenAPI spec missing 'openapi' version field")
	}
	if doc["info"] == nil {
		t.Error("OpenAPI spec missing 'info' field")
	}
	if doc["paths"] == nil {
		t.Error("OpenAPI spec missing 'paths' field")
	}
}

// TestOpenAPISpecHasSecuritySchemes validates that all expected security schemes are defined.
func TestOpenAPISpecHasSecuritySchemes(t *testing.T) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(openapispec.OpenAPISpec, &doc); err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	components, ok := doc["components"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI spec missing 'components' section")
	}

	schemes, ok := components["securitySchemes"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI spec missing 'securitySchemes' section")
	}

	expected := []string{"basicAuth", "apiKey", "bearerAuth"}
	for _, name := range expected {
		if schemes[name] == nil {
			t.Errorf("Missing security scheme: %s", name)
		}
	}
}
