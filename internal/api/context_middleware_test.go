package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
)

func TestContextExtractionMiddleware_SetsContext(t *testing.T) {
	// The middleware should extract the context name from the URL parameter
	// and store it in the request context.
	var gotCtx string

	r := chi.NewRouter()
	r.Route("/contexts/{context}", func(r chi.Router) {
		r.Use(contextExtractionMiddleware)
		r.Get("/subjects", func(w http.ResponseWriter, r *http.Request) {
			gotCtx = registrycontext.RegistryContextFromRequest(r.Context())
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest("GET", "/contexts/.production/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if gotCtx != ".production" {
		t.Errorf("expected context %q, got %q", ".production", gotCtx)
	}
}

func TestContextExtractionMiddleware_NormalizesName(t *testing.T) {
	// Context names without a leading dot should get normalized with a dot prepended.
	// e.g., "TestCtx" becomes ".TestCtx"
	var gotCtx string

	r := chi.NewRouter()
	r.Route("/contexts/{context}", func(r chi.Router) {
		r.Use(contextExtractionMiddleware)
		r.Get("/subjects", func(w http.ResponseWriter, r *http.Request) {
			gotCtx = registrycontext.RegistryContextFromRequest(r.Context())
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest("GET", "/contexts/TestCtx/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if gotCtx != ".TestCtx" {
		t.Errorf("expected normalized context %q, got %q", ".TestCtx", gotCtx)
	}
}

func TestContextExtractionMiddleware_DefaultContext(t *testing.T) {
	// The special value ":.:" should map to the default context ".".
	var gotCtx string

	r := chi.NewRouter()
	r.Route("/contexts/{context}", func(r chi.Router) {
		r.Use(contextExtractionMiddleware)
		r.Get("/subjects", func(w http.ResponseWriter, r *http.Request) {
			gotCtx = registrycontext.RegistryContextFromRequest(r.Context())
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest("GET", "/contexts/:.:/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if gotCtx != registrycontext.DefaultContext {
		t.Errorf("expected default context %q, got %q", registrycontext.DefaultContext, gotCtx)
	}
}

func TestContextExtractionMiddleware_InvalidContext(t *testing.T) {
	// An invalid context name (containing disallowed characters) should return
	// an HTTP 422 error.
	r := chi.NewRouter()
	r.Route("/contexts/{context}", func(r chi.Router) {
		r.Use(contextExtractionMiddleware)
		r.Get("/subjects", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest("GET", "/contexts/invalid!name/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the error body contains the expected message
	var errResp struct {
		ErrorCode int    `json:"error_code"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.ErrorCode != 422 {
		t.Errorf("expected error_code 422, got %d", errResp.ErrorCode)
	}
	if errResp.Message != "Invalid context name" {
		t.Errorf("expected message %q, got %q", "Invalid context name", errResp.Message)
	}
}

func TestContextExtractionMiddleware_InvalidContextWithSpace(t *testing.T) {
	// A context name containing a space should be rejected.
	r := chi.NewRouter()
	r.Route("/contexts/{context}", func(r chi.Router) {
		r.Use(contextExtractionMiddleware)
		r.Get("/subjects", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// Spaces in URLs get encoded as %20; chi will decode the param
	req := httptest.NewRequest("GET", "/contexts/has%20space/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for context with space, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContextExtractionMiddleware_AlreadyDotPrefixed(t *testing.T) {
	// A context that already starts with "." should be passed through unchanged.
	var gotCtx string

	r := chi.NewRouter()
	r.Route("/contexts/{context}", func(r chi.Router) {
		r.Use(contextExtractionMiddleware)
		r.Get("/subjects", func(w http.ResponseWriter, r *http.Request) {
			gotCtx = registrycontext.RegistryContextFromRequest(r.Context())
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest("GET", "/contexts/.my-context/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if gotCtx != ".my-context" {
		t.Errorf("expected %q, got %q", ".my-context", gotCtx)
	}
}

func TestContextExtractionMiddleware_EmptyContext(t *testing.T) {
	// When the context URL param is empty, the middleware should pass through
	// without setting a context (default is used).
	var gotCtx string

	_ = chi.NewRouter()
	// Use a handler directly with the middleware but simulate empty param
	handler := contextExtractionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtx = registrycontext.RegistryContextFromRequest(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	// Set up chi route context with empty "context" param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("context", "")
	req = req.WithContext(req.Context())
	// No chi context param set at all, so chi.URLParam returns ""
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if gotCtx != registrycontext.DefaultContext {
		t.Errorf("expected default context %q when param is empty, got %q", registrycontext.DefaultContext, gotCtx)
	}
}

func TestContextExtractionMiddleware_ContextWithDashAndUnderscore(t *testing.T) {
	// Context names with dashes and underscores should be valid.
	var gotCtx string

	r := chi.NewRouter()
	r.Route("/contexts/{context}", func(r chi.Router) {
		r.Use(contextExtractionMiddleware)
		r.Get("/subjects", func(w http.ResponseWriter, r *http.Request) {
			gotCtx = registrycontext.RegistryContextFromRequest(r.Context())
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest("GET", "/contexts/my-ctx_v1/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// "my-ctx_v1" gets normalized to ".my-ctx_v1"
	if gotCtx != ".my-ctx_v1" {
		t.Errorf("expected %q, got %q", ".my-ctx_v1", gotCtx)
	}
}
