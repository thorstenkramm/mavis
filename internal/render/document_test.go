package render

import (
	"strings"
	"testing"
)

func TestRenderHeadingIDsAndHierarchicalTOC(t *testing.T) {
	source := []byte("# Title\n\n## Section One\n\n### Deep `code`\n\n## Section One\n\n#### Skipped Level\n")
	document, err := Render(source, "docs/guide/current.md", nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, fragment := range []string{
		`<h1 id="title">Title</h1>`,
		`<h2 id="section-one">Section One</h2>`,
		`<h3 id="deep-code">Deep <code>code</code></h3>`,
		`<h2 id="section-one-1">Section One</h2>`,
		`<h4 id="skipped-level">Skipped Level</h4>`,
	} {
		if !strings.Contains(document.HTML, fragment) {
			t.Errorf("HTML does not contain %q:\n%s", fragment, document.HTML)
		}
	}

	if len(document.TOC) != 1 {
		t.Fatalf("TOC roots = %d, want 1: %+v", len(document.TOC), document.TOC)
	}
	title := document.TOC[0]
	assertTOCItem(t, title, "Title", "title", 1)
	if len(title.Children) != 2 {
		t.Fatalf("Title children = %d, want 2: %+v", len(title.Children), title.Children)
	}
	first := title.Children[0]
	assertTOCItem(t, first, "Section One", "section-one", 2)
	if len(first.Children) != 1 {
		t.Fatalf("Section One children = %+v", first.Children)
	}
	assertTOCItem(t, first.Children[0], "Deep code", "deep-code", 3)
	second := title.Children[1]
	assertTOCItem(t, second, "Section One", "section-one-1", 2)
	if len(second.Children) != 1 {
		t.Fatalf("duplicate Section One children = %+v", second.Children)
	}
	assertTOCItem(t, second.Children[0], "Skipped Level", "skipped-level", 4)
}

func TestRenderUsesFirstH1AsTitle(t *testing.T) {
	document, err := Render([]byte("## Introduction\n\n# First *title* with `code`\n\n# Second title\n"), "guide.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	if document.Title != "First title with code" {
		t.Fatalf("Title = %q", document.Title)
	}
}

func TestRenderWithoutHeadings(t *testing.T) {
	document, err := Render([]byte("A paragraph without headings.\n"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.TOC) != 0 {
		t.Fatalf("TOC = %+v, want empty", document.TOC)
	}
	if document.Title != "" {
		t.Fatalf("Title = %q, want empty", document.Title)
	}
	if document.HTML != "<p>A paragraph without headings.</p>\n" {
		t.Fatalf("HTML = %q", document.HTML)
	}
}

func assertTOCItem(t *testing.T, item TOCItem, title, id string, level int) {
	t.Helper()
	if item.Title != title || item.ID != id || item.Fragment != "#"+id || item.Level != level {
		t.Fatalf("TOC item = %+v, want title %q ID %q level %d", item, title, id, level)
	}
}
