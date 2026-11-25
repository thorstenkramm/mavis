package web

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

func TestPageTemplateSemanticStates(t *testing.T) {
	pages, err := template.ParseFS(embedded, "templates/page.html")
	if err != nil {
		t.Fatal(err)
	}
	model := pageModel{
		Title:     "<Documentation>",
		ShowTitle: true,
		Navigation: []content.NavigationItem{{
			Name: "guide", URL: "/guide/", Directory: true,
			Children: []content.NavigationItem{{Name: "setup.md", URL: "/guide/setup.md", Current: true}},
		}},
		TOC: []render.TOCItem{{
			Title: "Setup", Fragment: "#setup", Level: 1,
			Children: []render.TOCItem{{Title: "Install", Fragment: "#install", Level: 2}},
		}},
		Entries:     []entryModel{{Name: "guide", URL: "/guide/"}},
		ShowListing: true,
	}
	var output bytes.Buffer
	if err := pages.Execute(&output, model); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	for _, want := range []string{
		`<nav class="sidebar" id="file-navigation" aria-label="Documentation files">`,
		`<main class="content" id="main-content" tabindex="-1">`, `<article class="document">`,
		`<aside class="toc" aria-label="On this page">`,
		`href="/guide/setup.md" aria-current="page"`,
		`href="#setup"`, `href="#install"`, `class="directory-list"`,
		`&lt;Documentation&gt;`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("template output does not contain %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "<Documentation>") {
		t.Fatalf("template title was not escaped: %s", body)
	}
}

func TestNestedDocumentAndREADMENavigationStates(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "docs/README.md", "# Documentation\n")
	writeWebFile(t, root, "docs/guide/setup.md", "# Setup\n")
	writeWebFile(t, root, "unrelated/other.md", "# Other\n")
	handler := newWebHandler(t, root, nilRenderer{}, nil)

	document := requestWeb(handler, "/docs/guide/setup.md")
	if !strings.Contains(document.Body.String(), `href="/docs/guide/setup.md" aria-current="page"`) {
		t.Fatalf("current document state missing: %s", document.Body.String())
	}
	if strings.Contains(document.Body.String(), `href="/unrelated/other.md" aria-current="page"`) {
		t.Fatalf("unrelated document marked current: %s", document.Body.String())
	}
	readme := requestWeb(handler, "/docs/")
	if !strings.Contains(readme.Body.String(), `href="/docs/" aria-current="page"`) || !strings.Contains(readme.Body.String(), `<h1 id="documentation">Documentation</h1>`) {
		t.Fatalf("README index state missing: %s", readme.Body.String())
	}
}

func TestErrorPageRetainsSemanticNavigation(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "guide.md", "# Guide\n")
	handler := newWebHandler(t, root, nilRenderer{}, nil)
	response := requestWeb(handler, "/missing.md")
	if response.Code != 404 {
		t.Fatalf("status = %d", response.Code)
	}
	for _, want := range []string{`<h1>Not found</h1>`, `aria-label="Documentation files"`, `href="/guide.md"`, `<main class="content" id="main-content" tabindex="-1">`} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("error page does not contain %q:\n%s", want, response.Body.String())
		}
	}
}
