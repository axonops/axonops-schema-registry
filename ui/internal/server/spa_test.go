package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFS() fs.FS {
	return fstest.MapFS{
		"index.html":              {Data: []byte("<html>SPA</html>")},
		"assets/app-abc123.js":    {Data: []byte("console.log('app')")},
		"assets/style-def456.css": {Data: []byte("body{}")},
	}
}

func TestSPAServesStaticFile(t *testing.T) {
	handler := SPAHandler(testFS())

	req := httptest.NewRequest("GET", "/ui/assets/app-abc123.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "console.log")
}

func TestSPACachesAssets(t *testing.T) {
	handler := SPAHandler(testFS())

	req := httptest.NewRequest("GET", "/ui/assets/app-abc123.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "public, max-age=31536000, immutable", rec.Header().Get("Cache-Control"))
}

func TestSPAFallbackToIndex(t *testing.T) {
	handler := SPAHandler(testFS())

	req := httptest.NewRequest("GET", "/ui/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "<html>SPA</html>")
}

func TestSPAServesUIRoot(t *testing.T) {
	handler := SPAHandler(testFS())

	// /ui (no trailing slash) falls through to index.html
	req := httptest.NewRequest("GET", "/ui", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "<html>SPA</html>")
}

func TestSPAFallbackForDeepRoutes(t *testing.T) {
	handler := SPAHandler(testFS())

	paths := []string{
		"/ui/subjects/my-topic",
		"/ui/subjects/my-topic/versions/1",
		"/ui/admin/users",
		"/ui/tools/compatibility",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Contains(t, rec.Body.String(), "<html>SPA</html>")
		})
	}
}

func TestSPANoCacheForHTML(t *testing.T) {
	handler := SPAHandler(testFS())

	req := httptest.NewRequest("GET", "/ui/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should NOT have immutable cache header for HTML fallback
	cc := rec.Header().Get("Cache-Control")
	assert.NotContains(t, cc, "immutable")
}

func TestSPAServesSubdirIndexHTML(t *testing.T) {
	// Subdirectory index.html files (e.g. api-docs/index.html) must be served
	// directly — http.FileServer redirects them to "./" which breaks iframes.
	fsys := fstest.MapFS{
		"index.html":              {Data: []byte("<html>SPA</html>")},
		"api-docs/index.html":     {Data: []byte("<html>API Docs</html>")},
		"assets/app-abc123.js":    {Data: []byte("console.log('app')")},
		"assets/style-def456.css": {Data: []byte("body{}")},
	}
	handler := SPAHandler(fsys)

	t.Run("direct path serves subdir index.html not SPA", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ui/api-docs/index.html", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "API Docs")
		assert.NotContains(t, rec.Body.String(), "SPA")
	})

	t.Run("trailing slash serves subdir index.html", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ui/api-docs/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "API Docs")
		assert.NotContains(t, rec.Body.String(), "SPA")
	})

	t.Run("no trailing slash redirects to dir", func(t *testing.T) {
		// /ui/api-docs (no trailing slash) — http.FileServer sees the directory
		// and issues a 301 to api-docs/ which will then serve the subdir index.html
		req := httptest.NewRequest("GET", "/ui/api-docs", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	})

	t.Run("content type is text/html", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ui/api-docs/index.html", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	})
}

func TestSPAServeCSS(t *testing.T) {
	handler := SPAHandler(testFS())

	req := httptest.NewRequest("GET", "/ui/assets/style-def456.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "body{}")

	_ = require.NotNil // satisfy import
}
