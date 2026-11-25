package web

import (
	"bytes"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

func TestMarkdownDownloadReturnsOriginalBytes(t *testing.T) {
	root := t.TempDir()
	original := append([]byte{0xef, 0xbb, 0xbf}, []byte("---\r\ntitle: Metadata\r\n---\r\n# Visible\r\n")...)
	writeWebFile(t, root, "Über uns.md", string(original))
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	requestPath := content.URLFor("Über uns.md", false) + "?download=markdown"

	response := requestWeb(handler, requestPath)
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), original) {
		t.Fatalf("download = %d %v, want original %v", response.Code, response.Body.Bytes(), original)
	}
	mediaType, parameters, err := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
	if err != nil || mediaType != "attachment" || parameters["filename"] != "Über uns.md" {
		t.Fatalf("disposition = %q, parameters = %+v, error = %v", response.Header().Get("Content-Disposition"), parameters, err)
	}
	if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("X-Content-Type-Options") != "nosniff" ||
		response.Header().Get("Content-Type") != "text/markdown; charset=utf-8" {
		t.Fatalf("download headers = %+v", response.Header())
	}

	edited := []byte("# Current source\n")
	writeWebFile(t, root, "Über uns.md", string(edited))
	if refreshed := requestWeb(handler, requestPath); !bytes.Equal(refreshed.Body.Bytes(), edited) {
		t.Fatalf("refreshed source = %q, want %q", refreshed.Body.Bytes(), edited)
	}
}

func TestDirectoryREADMEDownloadTargetsFile(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "guide/README.md", "# Guide\n")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	response := requestWeb(handler, "/guide/?download=markdown")
	if response.Code != http.StatusOK || response.Body.String() != "# Guide\n" {
		t.Fatalf("README download = %d %q", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(disposition, "README.md") {
		t.Fatalf("README disposition = %q", disposition)
	}
}

func TestRawAttachmentAndUnknownDownloadFormats(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "notes.txt", "plain text")
	writeWebFile(t, root, "guide.md", "# Guide")
	writeWebFile(t, root, "empty/.keep", "keep")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)

	attachment := requestWeb(handler, "/notes.txt?raw=1&download=1")
	if attachment.Code != http.StatusOK || attachment.Body.String() != "plain text" ||
		!strings.HasPrefix(attachment.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatalf("raw attachment = %d %q, headers = %+v", attachment.Code, attachment.Body.String(), attachment.Header())
	}
	for _, requestPath := range []string{
		"/guide.md?download=unknown",
		"/guide.md?raw=1&download=1",
		"/notes.txt?download=markdown",
		"/notes.txt?download=1",
		"/empty/?download=markdown",
	} {
		if response := requestWeb(handler, requestPath); response.Code != http.StatusBadRequest {
			t.Errorf("GET %q = %d, want 400", requestPath, response.Code)
		}
	}
}

func TestMarkdownDownloadKeepsContentBoundaries(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, ".mavis.conf", "include_hidden = true\n")
	writeWebFile(t, root, ".hidden.md", "hidden")
	writeWebFile(t, root, "excluded.md", "excluded")
	configuration := config.Default()
	configuration.Exclude = []string{"excluded.md"}
	resolver, err := content.NewResolver(root, configuration)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandlerWithRenderer(resolver, render.NewRenderer(), nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, requestPath := range []string{
		"/missing.md?download=markdown",
		"/.mavis.conf?download=markdown",
		"/.hidden.md?download=markdown",
		"/excluded.md?download=markdown",
		"/%2e%2e/outside.md?download=markdown",
	} {
		response := requestWeb(handler, requestPath)
		if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), root) {
			t.Errorf("GET %q = %d %q", requestPath, response.Code, response.Body.String())
		}
	}
}

func TestMarkdownDownloadRejectsOutsideRootSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require additional privileges on Windows")
	}
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "alias.md")); err != nil {
		t.Fatal(err)
	}
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	response := requestWeb(handler, "/alias.md?download=markdown")
	if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "outside secret") {
		t.Fatalf("outside symlink download = %d %q", response.Code, response.Body.String())
	}
}
