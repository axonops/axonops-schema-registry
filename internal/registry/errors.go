package registry

import "errors"

// Sentinel errors for the registry layer.
// These allow handlers to check error types with errors.Is() instead of string matching.
var (
	ErrInvalidSchema           = errors.New("invalid schema")
	ErrUnsupportedSchemaType   = errors.New("unsupported schema type")
	ErrInvalidRuleSet          = errors.New("invalid ruleSet")
	ErrFailedResolveReferences = errors.New("failed to resolve references")
	ErrReferenceExists         = errors.New("schema is referenced by other schemas")
	ErrInvalidCompatibility    = errors.New("invalid compatibility level")
	ErrInvalidMode             = errors.New("invalid mode")
)
