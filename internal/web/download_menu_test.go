package web

import (
	"strings"
	"testing"
)

func TestMarkdownDownloadMenuUsesNativeAccessibleFormats(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "README.md", "# Guide")
	handler := newExportWebHandler(t, root, &fakeExporter{docx: true})
	body := requestWeb(handler, "/").Body.String()
	for _, want := range []string{
		`<div class="document-toolbar">`,
		`<details class="download-menu"><summary class="download-summary">`,
		`aria-hidden="true" focusable="false"`,
		`<span class="visually-hidden">Download</span>`, `class="download-caret"`,
		`href="/README.md?download=markdown">Markdown</a>`,
		`href="/README.md?download=pdf">PDF</a>`,
		`href="/README.md?download=docx">DOCX</a>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("download menu does not contain %q: %s", want, body)
		}
	}
	if strings.Contains(readEmbeddedText(t, "static/app.js"), "download-menu") {
		t.Fatal("native download menu unexpectedly requires JavaScript")
	}
}

func TestMarkdownDownloadMenuOmitsUnavailableDOCX(t *testing.T) {
	for _, docx := range []bool{false, true} {
		root := t.TempDir()
		writeWebFile(t, root, "guide.md", "# Guide")
		handler := newExportWebHandler(t, root, &fakeExporter{docx: docx})
		body := requestWeb(handler, "/guide.md").Body.String()
		if strings.Contains(body, `download=docx`) != docx {
			t.Errorf("DOCX available %v, menu = %s", docx, body)
		}
	}
}

func TestViewerDownloadMenuAndDirectoryAbsence(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "notes.txt", "plain")
	writeWebFile(t, root, "empty/.keep", "keep")
	handler := newExportWebHandler(t, root, &fakeExporter{docx: true})
	viewer := requestWeb(handler, "/notes.txt").Body.String()
	for _, want := range []string{`href="/notes.txt?raw=1">View raw file</a>`, `href="/notes.txt?download=1&amp;raw=1">Raw file</a>`} {
		if !strings.Contains(viewer, want) {
			t.Errorf("viewer does not contain %q: %s", want, viewer)
		}
	}
	if strings.Contains(viewer, `download=markdown`) || strings.Contains(viewer, `download=pdf`) || strings.Contains(viewer, `download=docx`) {
		t.Fatalf("viewer offers Markdown-only formats: %s", viewer)
	}
	if directory := requestWeb(handler, "/empty/").Body.String(); strings.Contains(directory, `class="download-menu"`) {
		t.Fatalf("directory listing has a download menu: %s", directory)
	}
}

func TestDownloadMenuResponsiveAndFocusStyles(t *testing.T) {
	stylesheet := readEmbeddedText(t, "static/app.css")
	for _, want := range []string{
		".document-toolbar", "justify-content: flex-end", ".download-options", "position: absolute",
		"right: 0", "max-width: calc(100vw - 2rem)", "min-width: 44px", "min-height: 44px",
		"border: 1px solid var(--control-border)", ":focus-visible",
	} {
		if !strings.Contains(stylesheet, want) {
			t.Errorf("stylesheet does not contain %q", want)
		}
	}
}
