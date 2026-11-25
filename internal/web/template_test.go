package web

import (
	"bytes"
	"html/template"
	"regexp"
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
			Name: "guide", RelativePath: "guide", URL: "/guide/", Directory: true,
			Children: []content.NavigationItem{{Name: "setup.md", RelativePath: "guide/setup.md", URL: "/guide/setup.md", Current: true}},
		}, {
			Name: "reference", RelativePath: "reference", URL: "/reference/", Directory: true,
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
		`aria-label="Collapse guide" aria-expanded="true" data-navigation-path="guide"`,
		`aria-label="Expand reference" aria-expanded="false" data-navigation-path="reference"`,
		`href="/guide/"`, `href="/reference/"`,
		`class="navigation-icon"`, `aria-hidden="true" focusable="false"`,
		`<ul class="navigation-children" hidden></ul>`,
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
	for _, want := range []string{
		`aria-label="Collapse docs" aria-expanded="true"`,
		`aria-label="Collapse guide" aria-expanded="true"`,
		`aria-label="Expand unrelated" aria-expanded="false"`,
		`href="/docs/guide/"`, `href="/unrelated/"`,
	} {
		if !strings.Contains(document.Body.String(), want) {
			t.Errorf("server-rendered navigation does not contain %q: %s", want, document.Body.String())
		}
	}
	readme := requestWeb(handler, "/docs/")
	if !strings.Contains(readme.Body.String(), `href="/docs/" aria-current="page"`) || !strings.Contains(readme.Body.String(), `<h1 id="documentation">Documentation</h1>`) {
		t.Fatalf("README index state missing: %s", readme.Body.String())
	}
}

func TestHeadingIDsNeverCollideWithTemplateControls(t *testing.T) {
	matches := regexp.MustCompile(` id="([a-z-]+)"`).FindAllStringSubmatch(readEmbeddedText(t, "templates/page.html"), -1)
	if len(matches) < 6 {
		t.Fatalf("template control IDs = %v", matches)
	}
	var source strings.Builder
	source.WriteString("# Title\n")
	for _, match := range matches {
		if match[1] == "search-results-title" {
			continue // This control is only rendered on the search route.
		}
		source.WriteString("\n## " + strings.ReplaceAll(match[1], "-", " ") + "\n")
	}
	root := t.TempDir()
	writeWebFile(t, root, "guide.md", source.String())
	body := requestWeb(newWebHandler(t, root, render.NewRenderer(), nil), "/guide.md").Body.String()
	for _, match := range matches {
		id := match[1]
		if id == "search-results-title" {
			continue
		}
		if strings.Count(body, `id="`+id+`"`) != 1 || !strings.Contains(body, `id="`+id+`-1"`) || !strings.Contains(body, `href="#`+id+`-1"`) {
			t.Errorf("heading ID collides with template control %q", id)
		}
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
