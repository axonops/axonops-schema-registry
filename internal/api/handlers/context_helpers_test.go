package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
)

// --- getRegistryContext ---

func TestGetRegistryContext_Default(t *testing.T) {
	// When no context is set in the request, getRegistryContext should return
	// the default context ".".
	req := httptest.NewRequest("GET", "/subjects", nil)

	got := getRegistryContext(req)
	if got != registrycontext.DefaultContext {
		t.Errorf("expected default context %q, got %q", registrycontext.DefaultContext, got)
	}
}

func TestGetRegistryContext_FromMiddleware(t *testing.T) {
	// When context is set via WithRegistryContext (simulating the middleware),
	// getRegistryContext should return that context.
	req := httptest.NewRequest("GET", "/subjects", nil)
	ctx := registrycontext.WithRegistryContext(req.Context(), ".production")
	req = req.WithContext(ctx)

	got := getRegistryContext(req)
	if got != ".production" {
		t.Errorf("expected %q, got %q", ".production", got)
	}
}

// --- resolveSubjectAndContext ---

func TestResolveSubjectAndContext_PlainSubject(t *testing.T) {
	// A plain subject "my-subject" with no context in the request should return
	// the default context and the subject unchanged.
	r := chi.NewRouter()
	var gotCtx, gotSubject string
	r.Get("/subjects/{subject}/versions", func(w http.ResponseWriter, r *http.Request) {
		gotCtx, gotSubject = resolveSubjectAndContext(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/subjects/my-subject/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if gotCtx != registrycontext.DefaultContext {
		t.Errorf("expected default context %q, got %q", registrycontext.DefaultContext, gotCtx)
	}
	if gotSubject != "my-subject" {
		t.Errorf("expected subject %q, got %q", "my-subject", gotSubject)
	}
}

func TestResolveSubjectAndContext_QualifiedSubject(t *testing.T) {
	// A qualified subject ":.myctx:my-subject" should extract the context
	// ".myctx" and subject "my-subject" from the subject string itself.
	r := chi.NewRouter()
	var gotCtx, gotSubject string
	r.Get("/subjects/{subject}/versions", func(w http.ResponseWriter, r *http.Request) {
		gotCtx, gotSubject = resolveSubjectAndContext(r)
		w.WriteHeader(http.StatusOK)
	})

	// URL-encode the qualified subject: ":.myctx:my-subject"
	req := httptest.NewRequest("GET", "/subjects/:.myctx:my-subject/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if gotCtx != ".myctx" {
		t.Errorf("expected context %q, got %q", ".myctx", gotCtx)
	}
	if gotSubject != "my-subject" {
		t.Errorf("expected subject %q, got %q", "my-subject", gotSubject)
	}
}

func TestResolveSubjectAndContext_QualifiedOverridesURL(t *testing.T) {
	// When the URL path has a context set via middleware AND the subject is
	// qualified, the qualified subject's context should win.
	r := chi.NewRouter()
	var gotCtx, gotSubject string

	// Simulate middleware that sets a URL context
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := registrycontext.WithRegistryContext(r.Context(), ".url-context")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	r.Get("/subjects/{subject}/versions", func(w http.ResponseWriter, r *http.Request) {
		gotCtx, gotSubject = resolveSubjectAndContext(r)
		w.WriteHeader(http.StatusOK)
	})

	// Qualified subject has its own context ".qualified-ctx"
	req := httptest.NewRequest("GET", "/subjects/:.qualified-ctx:my-subject/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if gotCtx != ".qualified-ctx" {
		t.Errorf("expected qualified context %q to override URL context, got %q", ".qualified-ctx", gotCtx)
	}
	if gotSubject != "my-subject" {
		t.Errorf("expected subject %q, got %q", "my-subject", gotSubject)
	}
}

func TestResolveSubjectAndContext_URLContext(t *testing.T) {
	// When the URL sets a context via middleware but the subject is plain,
	// the URL context should be used.
	r := chi.NewRouter()
	var gotCtx, gotSubject string

	// Simulate middleware that sets a URL context
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := registrycontext.WithRegistryContext(r.Context(), ".staging")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	r.Get("/subjects/{subject}/versions", func(w http.ResponseWriter, r *http.Request) {
		gotCtx, gotSubject = resolveSubjectAndContext(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/subjects/plain-subject/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if gotCtx != ".staging" {
		t.Errorf("expected URL context %q, got %q", ".staging", gotCtx)
	}
	if gotSubject != "plain-subject" {
		t.Errorf("expected subject %q, got %q", "plain-subject", gotSubject)
	}
}

func TestResolveSubjectAndContext_DefaultContextSubjectPrefix(t *testing.T) {
	// A subject with ":.:" prefix (default context) should resolve to the
	// default context. The subject starts with ":." but "." is the default.
	// ResolveSubject(":.::my-subject") -> rest is ":" => idx=0, which is not > 0
	// so it falls through to default.
	r := chi.NewRouter()
	var gotCtx, gotSubject string
	r.Get("/subjects/{subject}/versions", func(w http.ResponseWriter, r *http.Request) {
		gotCtx, gotSubject = resolveSubjectAndContext(r)
		w.WriteHeader(http.StatusOK)
	})

	// ":.:" with no further content treats the whole thing as a plain subject
	req := httptest.NewRequest("GET", "/subjects/:.:my-subject/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// ":." followed by ":my-subject" -> rest = ":my-subject", idx at first ":"
	// is 0, so idx > 0 is false. Falls back to default context.
	if gotCtx != registrycontext.DefaultContext {
		t.Errorf("expected default context %q, got %q", registrycontext.DefaultContext, gotCtx)
	}
	if gotSubject != ":.:my-subject" {
		t.Errorf("expected raw subject %q, got %q", ":.:my-subject", gotSubject)
	}
}

func TestResolveSubjectAndContext_EmptySubject(t *testing.T) {
	// When chi provides an empty subject URL param, we should get default context
	// and empty subject. This validates the edge case.
	req := httptest.NewRequest("GET", "/subjects//versions", nil)

	// Manually set up chi route context with empty subject param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("subject", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	gotCtx, gotSubject := resolveSubjectAndContext(req)
	if gotCtx != registrycontext.DefaultContext {
		t.Errorf("expected default context %q, got %q", registrycontext.DefaultContext, gotCtx)
	}
	if gotSubject != "" {
		t.Errorf("expected empty subject, got %q", gotSubject)
	}
}
