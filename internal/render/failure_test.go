package render

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderingFailureHasSafeFallbackAndDoesNotAffectOtherDocuments(t *testing.T) {
	renderer := NewRenderer()
	calls := 0
	renderer.render = func(source []byte, documentPath string, authorize AccessFunc) (Document, error) {
		calls++
		if calls == 1 {
			return Document{}, errors.New("private rendering diagnostic")
		}
		return renderDocument(source, documentPath, authorize)
	}

	_, err := renderer.Render([]byte("# Broken\n"), "broken.md", nil)
	if err == nil {
		t.Fatal("first Render() error = nil")
	}
	fallback := FallbackDocument()
	if strings.Contains(fallback.HTML, err.Error()) || strings.Contains(fallback.HTML, "private") {
		t.Fatalf("fallback exposes rendering diagnostics: %q", fallback.HTML)
	}
	if !strings.Contains(fallback.HTML, "could not be rendered") || len(fallback.TOC) != 0 {
		t.Fatalf("fallback = %+v", fallback)
	}

	document, err := renderer.Render([]byte("# Healthy\n"), "healthy.md", nil)
	if err != nil {
		t.Fatalf("second Render() error = %v", err)
	}
	if !strings.Contains(document.HTML, `<h1 id="healthy">Healthy</h1>`) || len(document.TOC) != 1 {
		t.Fatalf("healthy document after failure = %+v", document)
	}
}
