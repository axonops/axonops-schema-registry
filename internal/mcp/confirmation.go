package mcp

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
)

// ConfirmationStore manages two-phase confirmation tokens for destructive
// MCP operations. Tokens are single-use, scoped to a specific tool+args
// combination, and expire after a configurable TTL.
type ConfirmationStore struct {
	mu        sync.Mutex
	tokens    map[string]*confirmationEntry
	ttl       time.Duration
	stopGC    chan struct{}
	closeOnce sync.Once
}

type confirmationEntry struct {
	toolName  string
	argsHash  string
	preview   map[string]any
	expiresAt time.Time
	used      bool
}

// NewConfirmationStore creates a confirmation store with the given TTL
// and starts a background garbage collection goroutine.
func NewConfirmationStore(ttl time.Duration) *ConfirmationStore {
	cs := &ConfirmationStore{
		tokens: make(map[string]*confirmationEntry),
		ttl:    ttl,
		stopGC: make(chan struct{}),
	}
	go cs.gcLoop()
	return cs
}

// Generate creates a new confirmation token for the given tool and args,
// returning the token ID (UUID v4).
func (cs *ConfirmationStore) Generate(toolName string, args map[string]any, preview map[string]any) string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	token := uuid.New().String()
	cs.tokens[token] = &confirmationEntry{
		toolName:  toolName,
		argsHash:  computeArgsHash(toolName, args),
		preview:   preview,
		expiresAt: time.Now().Add(cs.ttl),
	}
	return token
}

// Validate checks a confirmation token. Returns nil if the token is valid
// (exists, not expired, not used, scope matches) and marks it as used.
func (cs *ConfirmationStore) Validate(tokenID, toolName string, args map[string]any) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entry, ok := cs.tokens[tokenID]
	if !ok {
		return fmt.Errorf("invalid or expired confirmation token")
	}
	if time.Now().After(entry.expiresAt) {
		delete(cs.tokens, tokenID)
		return fmt.Errorf("confirmation token has expired")
	}
	if entry.used {
		return fmt.Errorf("confirmation token has already been used")
	}
	if entry.toolName != toolName {
		return fmt.Errorf("confirmation token was issued for tool %q, not %q", entry.toolName, toolName)
	}
	hash := computeArgsHash(toolName, args)
	if entry.argsHash != hash {
		return fmt.Errorf("confirmation token does not match the provided arguments")
	}

	entry.used = true
	return nil
}

// Close stops the background GC goroutine. Safe to call multiple times.
func (cs *ConfirmationStore) Close() {
	cs.closeOnce.Do(func() { close(cs.stopGC) })
}

func (cs *ConfirmationStore) gcLoop() {
	interval := cs.ttl / 2
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-cs.stopGC:
			return
		case <-ticker.C:
			cs.gc()
		}
	}
}

func (cs *ConfirmationStore) gc() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	now := time.Now()
	for id, entry := range cs.tokens {
		if now.After(entry.expiresAt) || entry.used {
			delete(cs.tokens, id)
		}
	}
}

// computeArgsHash produces a deterministic SHA-256 hash of toolName + canonical
// JSON args. The dry_run and confirm_token fields are removed before hashing so
// that the token from a dry_run call matches a subsequent confirm call.
func computeArgsHash(toolName string, args map[string]any) string {
	cleaned := make(map[string]any, len(args))
	for k, v := range args {
		if k == "dry_run" || k == "confirm_token" {
			continue
		}
		cleaned[k] = v
	}
	data, err := json.Marshal(cleaned) // Go sorts map keys deterministically
	if err != nil {
		// Fallback: hash toolName alone if marshal fails (should never happen with simple maps).
		data = []byte("{}")
	}
	h := sha256.Sum256([]byte(toolName + ":" + string(data)))
	return fmt.Sprintf("%x", h)
}

// confirmableTools maps tool names to predicate functions that determine
// whether a particular invocation requires confirmation based on its arguments.
var confirmableTools = map[string]func(args map[string]any) bool{
	"delete_subject":  func(a map[string]any) bool { return boolArg(a, "permanent") },
	"delete_version":  func(a map[string]any) bool { return boolArg(a, "permanent") },
	"import_schemas":  func(_ map[string]any) bool { return true },
	"set_mode":        func(a map[string]any) bool { return strings.EqualFold(stringArg(a, "mode"), "IMPORT") },
	"delete_config":   func(a map[string]any) bool { return stringArg(a, "subject") == "" },
	"delete_kek":      func(a map[string]any) bool { return boolArg(a, "permanent") },
	"delete_dek":      func(a map[string]any) bool { return boolArg(a, "permanent") },
	"delete_exporter": func(_ map[string]any) bool { return true },
}

func boolArg(args map[string]any, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return strings.EqualFold(b, "true")
	default:
		return false
	}
}

func stringArg(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// confirmationCheck evaluates whether a tool call requires two-phase
// confirmation and returns an appropriate result. Returns nil if the call
// should proceed normally.
func (s *Server) confirmationCheck(toolName string, dryRun bool, confirmToken string,
	args map[string]any, preview map[string]any) *gomcp.CallToolResult {

	if s.confirmStore == nil {
		return nil
	}

	predicate, ok := confirmableTools[toolName]
	if !ok {
		return nil
	}
	if !predicate(args) {
		return nil
	}

	if dryRun {
		token := s.confirmStore.Generate(toolName, args, preview)
		if s.metrics != nil {
			s.metrics.RecordMCPConfirmation("token_issued")
		}
		if s.auditLogger != nil {
			actorID, actorType, authMethod := s.mcpActor()
			s.auditLogger.LogMCPConfirmationEvent(auth.AuditEventMCPConfirmIssued, actorID, actorType, authMethod, toolName, nil)
		}
		data, err := json.Marshal(map[string]any{
			"confirmation_required": true,
			"confirm_token":         token,
			"preview":               preview,
			"message":               fmt.Sprintf("This operation requires confirmation. To proceed, call %s again with confirm_token set to the token above and dry_run omitted or false.", toolName),
		})
		if err != nil {
			return &gomcp.CallToolResult{
				Content: []gomcp.Content{
					&gomcp.TextContent{Text: fmt.Sprintf("confirmation_required: token=%s (marshal error: %v)", token, err)},
				},
			}
		}
		return &gomcp.CallToolResult{
			Content: []gomcp.Content{
				&gomcp.TextContent{Text: string(data)},
			},
		}
	}

	if confirmToken != "" {
		if err := s.confirmStore.Validate(confirmToken, toolName, args); err != nil {
			if s.metrics != nil {
				s.metrics.RecordMCPConfirmation("token_rejected")
			}
			if s.auditLogger != nil {
				actorID, actorType, authMethod := s.mcpActor()
				s.auditLogger.LogMCPConfirmationEvent(auth.AuditEventMCPConfirmRejected, actorID, actorType, authMethod, toolName, nil)
			}
			return &gomcp.CallToolResult{
				Content: []gomcp.Content{
					&gomcp.TextContent{Text: fmt.Sprintf("error: confirmation failed: %v", err)},
				},
				IsError: true,
			}
		}
		if s.metrics != nil {
			s.metrics.RecordMCPConfirmation("confirmed")
		}
		if s.auditLogger != nil {
			actorID, actorType, authMethod := s.mcpActor()
			s.auditLogger.LogMCPConfirmationEvent(auth.AuditEventMCPConfirmed, actorID, actorType, authMethod, toolName, nil)
		}
		return nil // proceed with the operation
	}

	// Neither dry_run nor confirm_token provided
	if s.metrics != nil {
		s.metrics.RecordMCPPolicyDenial("confirmation_required")
	}
	msg := fmt.Sprintf("This destructive operation requires confirmation. Call %s with dry_run=true first to get a confirmation token.", toolName)
	data, err := json.Marshal(map[string]any{
		"error":                 "confirmation_required",
		"confirmation_required": true,
		"message":               msg,
	})
	if err != nil {
		return &gomcp.CallToolResult{
			Content: []gomcp.Content{
				&gomcp.TextContent{Text: msg},
			},
			IsError: true,
		}
	}
	// IsError is false because a confirmation prompt is not a failure — the tool
	// was called correctly and is returning an instructional response directing
	// the caller through the two-phase confirmation flow.  The structured JSON
	// body already contains "confirmation_required": true for MCP clients to parse.
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{
			&gomcp.TextContent{Text: string(data)},
		},
		IsError: false,
	}
}
