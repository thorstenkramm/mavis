package server

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
)

func TestHTTPHandlerDOCXVisibilityFollowsStartupDependencies(t *testing.T) {
	for _, test := range []struct {
		name         string
		dependencies ExportDependencies
		wantDOCX     bool
	}{
		{name: "not found or disabled", dependencies: ExportDependencies{}, wantDOCX: false},
		{name: "resolved at startup", dependencies: ExportDependencies{PandocExecutable: filepath.Join(string(filepath.Separator), "resolved", "pandoc")}, wantDOCX: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeServerFile(t, root, "README.md", "# Guide\n")
			resolver, err := content.NewResolver(root, config.Default())
			if err != nil {
				t.Fatal(err)
			}
			handler, err := newHTTPHandler(resolver, log.New(io.Discard, "", 0), test.dependencies)
			if err != nil {
				t.Fatal(err)
			}
			page := requestServerHandler(handler, "/")
			if strings.Contains(page.Body.String(), "download=docx") != test.wantDOCX {
				t.Fatalf("DOCX visibility = %v, want %v: %s", !test.wantDOCX, test.wantDOCX, page.Body.String())
			}
			download := requestServerHandler(handler, "/README.md?download=docx")
			if !test.wantDOCX && download.Code != http.StatusNotFound {
				t.Fatalf("unavailable DOCX status = %d, want 404", download.Code)
			}
		})
	}
}

func TestHTTPHandlerServesBuiltInPDF(t *testing.T) {
	root := t.TempDir()
	writeServerFile(t, root, "README.md", "---\ntitle: Private metadata\n---\n# Grüße\n\n| Sprache | Status |\n| --- | --- |\n| Deutsch | Gut |\n")
	resolver, err := content.NewResolver(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler, err := newHTTPHandler(resolver, log.New(io.Discard, "", 0), ExportDependencies{})
	if err != nil {
		t.Fatal(err)
	}
	response := requestServerHandler(handler, "/README.md?download=pdf")
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Body.String(), "%PDF-") {
		t.Fatalf("PDF response = %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/pdf" || !strings.Contains(response.Header().Get("Content-Disposition"), "README.pdf") {
		t.Fatalf("PDF headers = %+v", response.Header())
	}
}

func writeServerFile(t *testing.T, root, name, source string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}

func requestServerHandler(handler http.Handler, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
