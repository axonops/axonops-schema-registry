// Package api provides embedded API specification assets.
package api

import _ "embed"

// OpenAPISpec contains the embedded OpenAPI 3.0 specification.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
