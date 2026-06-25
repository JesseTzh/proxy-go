package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAttachWebServesPublicSvgBeforeSPAFallback(t *testing.T) {
	webRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<!doctype html><title>proxy-go</title>"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	svgDir := filepath.Join(webRoot, "svg")
	if err := os.MkdirAll(svgDir, 0755); err != nil {
		t.Fatalf("mkdir svg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(svgDir, "sing-box.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), 0644); err != nil {
		t.Fatalf("write svg: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	attachWeb(router, webRoot)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/svg/sing-box.svg", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "<!doctype html>") {
		t.Fatalf("expected svg body, got SPA fallback: %q", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "<svg") {
		t.Fatalf("expected svg body, got %q", response.Body.String())
	}
}
