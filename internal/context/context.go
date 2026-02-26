// Package context provides multi-tenancy support via schema registry contexts.
// Contexts are Confluent-compatible namespaces that isolate subjects, schema IDs,
// compatibility config, and modes within a single Schema Registry instance.
package context

import (
	"context"
	"fmt"
	"strings"
)

// DefaultContext is the name of the default registry context.
const DefaultContext = "."

// GlobalContext is the special cross-context config/mode namespace.
// Confluent-compatible: only config and mode operations are allowed on this context.
// Schemas and subjects CANNOT be registered under __GLOBAL.
const GlobalContext = ".__GLOBAL"

// RegistryContextKey is the context key for storing the registry context name in request context.
type registryContextKeyType string

const registryContextKey registryContextKeyType = "schema_registry_context"

// WithRegistryContext adds a registry context name to the request context.
func WithRegistryContext(ctx context.Context, registryCtx string) context.Context {
	return context.WithValue(ctx, registryContextKey, registryCtx)
}

// RegistryContextFromRequest retrieves the registry context name from request context.
// Returns DefaultContext (".") if not set.
func RegistryContextFromRequest(ctx context.Context) string {
	if v := ctx.Value(registryContextKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return DefaultContext
}

// ResolveSubject resolves a subject name that may contain a context prefix.
// Confluent format: :.contextname:subject → (contextname, subject)
// Plain subject: mysubject → (DefaultContext, mysubject)
//
// Context names in the prefix include a leading dot: :.TestContext:mysubject
// The returned context name includes the dot: ".TestContext"
func ResolveSubject(subject string) (registryCtx, resolvedSubject string) {
	if strings.HasPrefix(subject, ":.") {
		// Find the second colon after the leading ":."
		rest := subject[2:] // everything after ":."
		idx := strings.Index(rest, ":")
		if idx > 0 {
			// :.contextname:subject (subject may be empty for context-level operations)
			ctxName := "." + rest[:idx] // prepend dot for display form
			subj := rest[idx+1:]
			return ctxName, subj
		}
	}
	return DefaultContext, subject
}

// FormatSubject formats a subject with a context prefix.
// Returns just the subject name for the default context.
// For non-default contexts: :.contextname:subject
func FormatSubject(registryCtx, subject string) string {
	if registryCtx == DefaultContext || registryCtx == "" {
		return subject
	}
	// registryCtx already includes the leading dot (e.g., ".TestContext")
	// Output: :.TestContext:subject
	return fmt.Sprintf(":%s:%s", registryCtx, subject)
}

// IsValidContextName validates a context name.
// Context names can contain alphanumeric characters, dashes, underscores, and dots.
// The leading dot is part of the display name (e.g., ".TestContext").
func IsValidContextName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	// The default context "." is always valid
	if name == DefaultContext {
		return true
	}
	// Allow alphanumeric, dash, underscore, dot
	for _, c := range name {
		if !isAlphaNumeric(c) && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

// NormalizeContextName ensures a context name is in the proper display form.
// If the name doesn't start with ".", it prepends one.
// The special value ":.:" maps to the default context ".".
func NormalizeContextName(name string) string {
	if name == ":.:" || name == "" {
		return DefaultContext
	}
	if !strings.HasPrefix(name, ".") {
		return "." + name
	}
	return name
}

// IsGlobalContext returns true if the context name is the special __GLOBAL context.
func IsGlobalContext(name string) bool {
	return name == GlobalContext
}

func isAlphaNumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
