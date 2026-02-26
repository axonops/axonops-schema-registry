package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
)

// getRegistryContext returns the registry context name from the request.
// If a context was set by the context extraction middleware (URL prefix routing),
// it returns that. Otherwise returns the default context ".".
func getRegistryContext(r *http.Request) string {
	return registrycontext.RegistryContextFromRequest(r.Context())
}

// resolveSubjectAndContext extracts the registry context and subject from a request.
// It checks three sources in priority order:
//  1. Qualified subject name: :.TestContext:mysubject → (".TestContext", "mysubject")
//  2. URL path context (from middleware) + plain subject
//  3. Neither → default context (".") + plain subject
//
// When using qualified subjects at the root level, the response should include
// the qualified subject name. When using URL prefix routing, the response
// should include the plain subject name.
func resolveSubjectAndContext(r *http.Request) (registryCtx string, subject string) {
	rawSubject := chi.URLParam(r, "subject")

	// Check if the subject contains a context prefix
	resolvedCtx, resolvedSubject := registrycontext.ResolveSubject(rawSubject)
	if resolvedCtx != registrycontext.DefaultContext {
		// Qualified subject takes precedence
		return resolvedCtx, resolvedSubject
	}

	// Fall back to URL path context (from middleware) or default
	return getRegistryContext(r), rawSubject
}

// rejectGlobalContext returns true (and writes an error response) if the registry context
// is the __GLOBAL context. Schema and subject operations are not permitted on __GLOBAL;
// only config and mode operations are allowed (Confluent-compatible).
func rejectGlobalContext(w http.ResponseWriter, registryCtx string) bool {
	if registrycontext.IsGlobalContext(registryCtx) {
		writeError(w, http.StatusBadRequest, types.ErrorCodeOperationNotPermitted,
			fmt.Sprintf("Subject operations are not permitted on the %s context", registrycontext.GlobalContext))
		return true
	}
	return false
}
