// Package server provides the HTTP server and SPA handler.
package server

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// SPAHandler serves the embedded React SPA.
// The SPA is built with base="/ui/" so all browser requests arrive as /ui/*.
// The embedded FS contains the dist output directly (index.html, assets/...).
// We strip the /ui/ prefix to map browser paths to FS paths.
func SPAHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip /ui/ prefix: /ui/assets/foo.js → assets/foo.js
		fsPath := strings.TrimPrefix(r.URL.Path, "/ui/")
		fsPath = strings.TrimPrefix(fsPath, "/ui")
		if fsPath == "" {
			fsPath = "index.html"
		}

		// Try to serve the file directly from the embedded FS
		if fsPath != "index.html" {
			// For directory paths (e.g. "api-docs/"), check for index.html inside
			if strings.HasSuffix(fsPath, "/") {
				idxPath := fsPath + "index.html"
				if _, err := fsys.Open(idxPath); err == nil {
					serveFileFromFS(w, r, fsys, idxPath)
					return
				}
			}

			if f, err := fsys.Open(fsPath); err == nil {
				f.Close()
				// Set cache headers for hashed static assets
				if strings.HasPrefix(fsPath, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				// Subdirectory index.html files (e.g. api-docs/index.html) must be
				// served directly — http.FileServer redirects them to "./" which
				// breaks when used inside an iframe (causes recursive SPA loading).
				if path.Base(fsPath) == "index.html" {
					serveFileFromFS(w, r, fsys, fsPath)
					return
				}
				// Rewrite the request path so http.FileServer finds the file
				r.URL.Path = "/" + fsPath
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// For SPA routes (and /ui root), serve index.html via "/"
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// serveFileFromFS reads a file from the FS and writes it to the response,
// bypassing http.FileServer's redirect behaviour for index.html files.
func serveFileFromFS(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) {
	f, err := fsys.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	// Detect content type from extension
	switch {
	case strings.HasSuffix(name, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(name, ".json"):
		w.Header().Set("Content-Type", "application/json")
	case strings.HasSuffix(name, ".yaml"), strings.HasSuffix(name, ".yml"):
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	}

	// If the file supports ReadSeeker, use http.ServeContent for proper caching
	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(w, r, name, time.Time{}, rs)
		return
	}

	// Fallback: stream the content
	io.Copy(w, f)
}
