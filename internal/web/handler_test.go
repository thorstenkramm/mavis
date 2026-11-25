package web

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

func TestRootREADMEUsesDynamicSecureHTML(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "README.md", "# Welcome\n\n## Section\n\nCurrent **documentation**.\n")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	response := requestWeb(handler, "/")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, want := range []string{
		`<article class="document">`, `<h1 id="welcome">Welcome</h1>`, `<strong>documentation</strong>`,
		`aria-label="Documentation files"`, `aria-label="On this page"`, `href="#section"`,
		`href="/_mavis/assets/app.css"`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("body does not contain %q:\n%s", want, response.Body.String())
		}
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	csp := response.Header().Get("Content-Security-Policy")
	for _, want := range []string{"default-src 'none'", "script-src 'self'", "object-src 'none'", "base-uri 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP %q does not contain %q", csp, want)
		}
	}
}

func TestMarkdownRefreshReadsCurrentSource(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "guide.md", "# Before\n")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	if response := requestWeb(handler, "/guide.md"); !strings.Contains(response.Body.String(), "Before") {
		t.Fatalf("initial body = %s", response.Body.String())
	}
	writeWebFile(t, root, "guide.md", "# After\n\nEdited on disk.\n")
	response := requestWeb(handler, "/guide.md")
	if !strings.Contains(response.Body.String(), "After") || !strings.Contains(response.Body.String(), "Edited on disk.") {
		t.Fatalf("refreshed body = %s", response.Body.String())
	}
}

func TestLiteralPercentNamesAndPageIDs(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "100% complete/guide%20.md", "# Main content\n\n- [ ] "+strings.Repeat("界", 510)+"\n")
	writeWebFile(t, root, "README.md", "[Guide](100%25%20complete/guide%2520.md)\n")
	h := newWebHandler(t, root, render.NewRenderer(), nil)
	const url = "/100%25%20complete/guide%2520.md"
	for _, from := range []string{"/", "/100%25%20complete/"} {
		if page := requestWeb(h, from); !strings.Contains(page.Body.String(), `href="`+url+`"`) {
			t.Fatalf("missing link to literal percent name on %s", from)
		}
	}
	page := requestWeb(h, url)
	body := page.Body.String()
	if page.Code != http.StatusOK || strings.Count(body, `id="main-content"`) != 1 ||
		!strings.Contains(body, `id="main-content-1"`) || !strings.Contains(body, `href="#main-content-1"`) ||
		!strings.Contains(body, `aria-label="`+strings.Repeat("界", 499)+`…"`) {
		t.Fatalf("unexpected rendered page: %d %s", page.Code, body)
	}
}

func TestEmbeddedAssetsUseReservedRoute(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "_mavis/assets/private.txt", "user secret")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)

	asset := requestWeb(handler, "/_mavis/assets/app.css")
	if asset.Code != http.StatusOK || !strings.Contains(asset.Body.String(), "grid-template-columns") {
		t.Fatalf("embedded asset response = %d %s", asset.Code, asset.Body.String())
	}
	if asset.Header().Get("Content-Security-Policy") == "" || asset.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("asset security headers = %+v", asset.Header())
	}
	private := requestWeb(handler, "/_mavis/assets/private.txt")
	if private.Code != http.StatusNotFound || strings.Contains(private.Body.String(), "user secret") {
		t.Fatalf("reserved route exposed user content: %d %s", private.Code, private.Body.String())
	}
}

func TestNonMarkdownFileUsesResolvedFileHandler(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "image.png", string([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}))
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	response := requestWeb(handler, "/image.png")
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("file response = %d, headers = %+v", response.Code, response.Header())
	}
	if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("file headers = %+v", response.Header())
	}
	for _, raw := range []string{"example.html", "drawing.svg", "example.js"} {
		writeWebFile(t, root, raw, "fixture")
		res := requestWeb(handler, "/"+raw)
		policy := res.Header().Get("Content-Security-Policy")
		if res.Code != 200 || !strings.Contains(policy, "sandbox;") || !strings.Contains(policy, "script-src 'none'") {
			t.Errorf("raw %s: status %d policy %q", raw, res.Code, policy)
		}
	}
}

func TestMissingAndPrivateContentHaveGenericErrors(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, ".mavis.conf", "include_hidden = true\n")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	for _, path := range []string{"/missing.md", "/.mavis.conf", "/%2e%2e/secret.md"} {
		response := requestWeb(handler, path)
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %q status = %d, body = %s", path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), root) || strings.Contains(response.Body.String(), "include_hidden") {
			t.Fatalf("GET %q exposed private details: %s", path, response.Body.String())
		}
	}
}

func TestRenderingFailureIsSafeAndIsolated(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "broken.md", "# Broken\n")
	writeWebFile(t, root, "healthy.md", "# Healthy\n")
	renderer := &isolatedTestRenderer{renderer: render.NewRenderer()}
	var logs bytes.Buffer
	handler := newWebHandler(t, root, renderer, log.New(&logs, "", 0))

	broken := requestWeb(handler, "/broken.md")
	if broken.Code != http.StatusInternalServerError || !strings.Contains(broken.Body.String(), "could not be rendered") {
		t.Fatalf("broken response = %d %s", broken.Code, broken.Body.String())
	}
	if strings.Contains(broken.Body.String(), "private rendering diagnostic") || !strings.Contains(logs.String(), "private rendering diagnostic") {
		t.Fatalf("render diagnostics leaked incorrectly: body = %s, logs = %s", broken.Body.String(), logs.String())
	}
	healthy := requestWeb(handler, "/healthy.md")
	if healthy.Code != http.StatusOK || !strings.Contains(healthy.Body.String(), `<h1 id="healthy">Healthy</h1>`) {
		t.Fatalf("healthy response after failure = %d %s", healthy.Code, healthy.Body.String())
	}
}

type isolatedTestRenderer struct {
	renderer *render.Renderer
}

func (renderer *isolatedTestRenderer) Render(source []byte, documentPath string, authorize render.AccessFunc) (render.Document, error) {
	if documentPath == "broken.md" {
		return render.Document{}, errors.New("private rendering diagnostic")
	}
	return renderer.renderer.Render(source, documentPath, authorize)
}

func newWebHandler(t *testing.T, root string, renderer MarkdownRenderer, logger *log.Logger) http.Handler {
	t.Helper()
	resolver, err := content.NewResolver(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandlerWithRenderer(resolver, renderer, logger)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func requestWeb(handler http.Handler, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func writeWebFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
