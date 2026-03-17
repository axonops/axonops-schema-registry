// Package content provides embedded markdown content for MCP glossary resources and prompt templates.
package content

import "embed"

// GlossaryFS contains embedded glossary markdown files.
//
//go:embed glossary/*.md
var GlossaryFS embed.FS

// PromptsFS contains embedded prompt template markdown files.
//
//go:embed prompts/*.md
var PromptsFS embed.FS
