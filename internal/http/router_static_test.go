package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"multi-platform-distribution/internal/config"
)

func TestRouterServesWebConsoleFallback(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("web console"), 0o644); err != nil {
		t.Fatal(err)
	}

	router := NewRouter(config.Config{WebDistDir: dist}, Services{})

	root := httptest.NewRecorder()
	router.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Code != http.StatusOK || root.Body.String() != "web console" {
		t.Fatalf("GET / = %d %q, want 200 web console", root.Code, root.Body.String())
	}

	api := httptest.NewRecorder()
	router.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/missing", nil))
	if api.Code != http.StatusNotFound || api.Body.String() == "web console" {
		t.Fatalf("GET /api/missing = %d %q, want API 404", api.Code, api.Body.String())
	}
}
