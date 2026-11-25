package render

import (
	"strings"
	"testing"
)

func TestRenderSeparatesLeadingFrontMatter(t *testing.T) {
	for _, test := range []struct {
		name       string
		source     string
		front      string
		wantHeader string
	}{
		{name: "LF", source: "---\ntitle: Metadata\n---\n# Visible\n", front: "title: Metadata\n", wantHeader: `<h1 id="visible">Visible</h1>`},
		{name: "CRLF", source: "---\r\ntitle: Metadata\r\n---\r\n# Visible\r\n", front: "title: Metadata\r\n", wantHeader: `<h1 id="visible">Visible</h1>`},
		{name: "YAML document end", source: "---\ntitle: Metadata\n...\n# Visible\n", front: "title: Metadata\n", wantHeader: `<h1 id="visible">Visible</h1>`},
		{name: "empty block", source: "---\n---\n# Visible\n", front: "", wantHeader: `<h1 id="visible">Visible</h1>`},
		{name: "byte order mark", source: "\xef\xbb\xbf---\ntitle: Metadata\n---\n# Visible\n", front: "title: Metadata\n", wantHeader: `<h1 id="visible">Visible</h1>`},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := Render([]byte(test.source), "guide.md", nil)
			if err != nil {
				t.Fatal(err)
			}
			if !document.HasFrontMatter || document.FrontMatter != test.front {
				t.Fatalf("front matter = %q, present %v; want %q, true", document.FrontMatter, document.HasFrontMatter, test.front)
			}
			if strings.Contains(document.HTML, "Metadata") || !strings.Contains(document.HTML, test.wantHeader) {
				t.Fatalf("rendered HTML = %q", document.HTML)
			}
			if len(document.TOC) != 1 || document.TOC[0].Title != "Visible" {
				t.Fatalf("TOC = %+v", document.TOC)
			}
		})
	}
}

func TestMarkdownBodyMatchesRendererFrontMatterRules(t *testing.T) {
	source := []byte("\xef\xbb\xbf---\r\ntitle: Metadata\r\n---\r\n# Visible\r\n")
	if body := string(MarkdownBody(source)); body != "# Visible\r\n" {
		t.Fatalf("MarkdownBody() = %q", body)
	}
	ordinary := []byte("\xef\xbb\xbf# Visible\n")
	if body := MarkdownBody(ordinary); string(body) != string(ordinary) {
		t.Fatalf("ordinary Markdown body = %q, want original bytes", body)
	}
}

func TestRenderDoesNotInventFrontMatter(t *testing.T) {
	for _, source := range []string{
		"# Visible\n\n---\ntitle: Ordinary Markdown\n",
		"---\ntitle: Unterminated\n# Visible\n",
		"\xef\xbb\xbf---\ntitle: Unterminated after a byte order mark\n",
		"\xef\xbb\xbf# Visible\n",
		"---",
		"",
	} {
		document, err := Render([]byte(source), "guide.md", nil)
		if err != nil {
			t.Fatal(err)
		}
		if document.HasFrontMatter {
			t.Errorf("Render(%q) unexpectedly detected front matter", source)
		}
	}
}
