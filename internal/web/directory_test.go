package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

func TestDirectoryListingAndEmptyStates(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "guide.md", "# Guide\n")
	writeWebFile(t, root, "folder/.keep", "not visible\n")
	writeWebFile(t, root, "empty/placeholder.txt", "placeholder\n")
	resolver, err := content.NewResolver(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(resolver, nil)
	if err != nil {
		t.Fatal(err)
	}

	listing := requestWeb(handler, "/")
	if listing.Code != http.StatusOK || !strings.Contains(listing.Body.String(), `href="/guide.md"`) {
		t.Fatalf("root listing = %d %s", listing.Code, listing.Body.String())
	}
	if !strings.Contains(listing.Body.String(), `href="/folder/"`) || strings.Contains(listing.Body.String(), ".keep") {
		t.Fatalf("root folder listing = %s", listing.Body.String())
	}
	empty := requestWeb(handler, "/folder/")
	if empty.Code != http.StatusOK || !strings.Contains(empty.Body.String(), "This directory is empty.") {
		t.Fatalf("empty directory = %d %s", empty.Code, empty.Body.String())
	}
}

func TestMarkdownLinksUseSharedAuthorization(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "docs/current.md", "[Allowed](target.md)\n\n[Hidden](.secret.md)\n")
	writeWebFile(t, root, "docs/target.md", "# Target\n")
	writeWebFile(t, root, "docs/.secret.md", "secret\n")
	handler := newWebHandler(t, root, nilRenderer{}, nil)
	response := requestWeb(handler, "/docs/current.md")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `href="/docs/target.md"`) {
		t.Fatalf("eligible link missing: %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), ".secret.md") {
		t.Fatalf("hidden destination exposed: %s", response.Body.String())
	}
}

func TestDynamicRoutesRejectUnsupportedMethods(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "README.md", "# Root\n")
	handler := newWebHandler(t, root, nilRenderer{}, nil)
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST response = %d, headers = %+v", response.Code, response.Header())
	}
}

type nilRenderer struct{}

func (nilRenderer) Render(source []byte, documentPath string, authorize render.AccessFunc) (render.Document, error) {
	return render.NewRenderer().Render(source, documentPath, authorize)
}
