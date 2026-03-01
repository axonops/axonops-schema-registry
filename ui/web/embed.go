// Package web provides the embedded React SPA build output.
//
// During production builds, the Makefile/Dockerfile copies the dist/ directory
// into web/dist/ before running go build, so the embed works.
// During development, pass nil for the SPA FS and use the vite dev server.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded filesystem rooted at the dist directory.
// Returns nil if the embedded FS is empty (development mode).
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil
	}
	// Check if the FS has any content
	entries, err := fs.ReadDir(sub, ".")
	if err != nil || len(entries) == 0 {
		return nil
	}
	return sub
}
