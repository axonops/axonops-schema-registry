package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
)

// contextExtractionMiddleware extracts the registry context from the URL path
// parameter {context} and stores it in the request context.
// The URL format is /contexts/{context}/... where {context} is a context name
// like ".TestContext" or ":.:" for the default context.
func contextExtractionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxParam := chi.URLParam(r, "context")
		if ctxParam == "" {
			// No context in URL, use default
			next.ServeHTTP(w, r)
			return
		}

		// Normalize the context name
		registryCtx := registrycontext.NormalizeContextName(ctxParam)

		// Validate the context name
		if !registrycontext.IsValidContextName(registryCtx) {
			http.Error(w, `{"error_code":422,"message":"Invalid context name"}`, http.StatusUnprocessableEntity)
			return
		}

		// Store in request context
		ctx := registrycontext.WithRegistryContext(r.Context(), registryCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
