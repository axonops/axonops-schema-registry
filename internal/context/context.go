// Package context provides multi-tenancy support via contexts.
package context

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Context represents a schema registry context for multi-tenancy.
type Context struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Config      *ContextConfig    `json:"config,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ContextConfig holds configuration for a context.
type ContextConfig struct {
	CompatibilityLevel string `json:"compatibilityLevel,omitempty"`
	Mode               string `json:"mode,omitempty"`
}

// ContextManager manages contexts for multi-tenancy.
type ContextManager struct {
	mu         sync.RWMutex
	contexts   map[string]*Context
	defaultCtx *Context
	storage    storage.Storage
}

// NewContextManager creates a new context manager.
func NewContextManager(store storage.Storage) *ContextManager {
	cm := &ContextManager{
		contexts: make(map[string]*Context),
		storage:  store,
		defaultCtx: &Context{
			Name:        ".",
			Description: "Default context",
			Config: &ContextConfig{
				CompatibilityLevel: "BACKWARD",
				Mode:               "READWRITE",
			},
		},
	}
	cm.contexts["."] = cm.defaultCtx
	return cm
}

// CreateContext creates a new context.
func (cm *ContextManager) CreateContext(ctx *Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if ctx.Name == "" {
		return fmt.Errorf("context name is required")
	}

	if !isValidContextName(ctx.Name) {
		return fmt.Errorf("invalid context name: %s", ctx.Name)
	}

	if _, exists := cm.contexts[ctx.Name]; exists {
		return fmt.Errorf("context already exists: %s", ctx.Name)
	}

	// Inherit from default if not specified
	if ctx.Config == nil {
		ctx.Config = &ContextConfig{
			CompatibilityLevel: cm.defaultCtx.Config.CompatibilityLevel,
			Mode:               cm.defaultCtx.Config.Mode,
		}
	}

	cm.contexts[ctx.Name] = ctx
	return nil
}

// GetContext retrieves a context by name.
func (cm *ContextManager) GetContext(name string) (*Context, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ctx, exists := cm.contexts[name]
	if !exists {
		return nil, fmt.Errorf("context not found: %s", name)
	}

	return ctx, nil
}

// ListContexts returns all contexts.
func (cm *ContextManager) ListContexts() []*Context {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]*Context, 0, len(cm.contexts))
	for _, ctx := range cm.contexts {
		result = append(result, ctx)
	}
	return result
}

// UpdateContext updates an existing context.
func (cm *ContextManager) UpdateContext(ctx *Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.contexts[ctx.Name]; !exists {
		return fmt.Errorf("context not found: %s", ctx.Name)
	}

	cm.contexts[ctx.Name] = ctx
	return nil
}

// DeleteContext deletes a context.
func (cm *ContextManager) DeleteContext(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if name == "." {
		return fmt.Errorf("cannot delete default context")
	}

	if _, exists := cm.contexts[name]; !exists {
		return fmt.Errorf("context not found: %s", name)
	}

	delete(cm.contexts, name)
	return nil
}

// GetDefaultContext returns the default context.
func (cm *ContextManager) GetDefaultContext() *Context {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.defaultCtx
}

// SetDefaultConfig sets the default context configuration.
func (cm *ContextManager) SetDefaultConfig(config *ContextConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.defaultCtx.Config = config
}

// ResolveSubject resolves a subject name with context prefix.
// Format: :.context.:subject or just subject (uses default context)
func (cm *ContextManager) ResolveSubject(subject string) (contextName, resolvedSubject string) {
	if strings.HasPrefix(subject, ":.") && strings.Contains(subject[2:], ".:") {
		// Format: :.context.:subject
		parts := strings.SplitN(subject[2:], ".:", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return ".", subject
}

// FormatSubject formats a subject with context prefix.
func (cm *ContextManager) FormatSubject(contextName, subject string) string {
	if contextName == "." || contextName == "" {
		return subject
	}
	return fmt.Sprintf(":.%s.:%s", contextName, subject)
}

// GetContextConfig gets the effective configuration for a context.
func (cm *ContextManager) GetContextConfig(name string) (*ContextConfig, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ctx, exists := cm.contexts[name]
	if !exists {
		return nil, fmt.Errorf("context not found: %s", name)
	}

	// Return context config or inherit from default
	if ctx.Config != nil {
		return ctx.Config, nil
	}
	return cm.defaultCtx.Config, nil
}

// isValidContextName validates a context name.
func isValidContextName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	// Allow alphanumeric, dash, underscore, dot
	for _, c := range name {
		if !isAlphaNumeric(c) && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

func isAlphaNumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// ContextKey is the context key for storing context info in request context.
type ContextKey string

const (
	// CurrentContextKey is the key for the current context in request context.
	CurrentContextKey ContextKey = "schema_registry_context"
)

// WithContext adds a context to the request context.
func WithContext(ctx context.Context, schemaCtx *Context) context.Context {
	return context.WithValue(ctx, CurrentContextKey, schemaCtx)
}

// FromContext retrieves the context from request context.
func FromContext(ctx context.Context) *Context {
	if v := ctx.Value(CurrentContextKey); v != nil {
		if c, ok := v.(*Context); ok {
			return c
		}
	}
	return nil
}
