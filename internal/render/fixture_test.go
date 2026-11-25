package render

import (
	"os"
	"strings"
	"testing"
)

func TestRepresentativeDocumentationFixtures(t *testing.T) {
	source, err := os.ReadFile("../../testdata/docs/guide/setup.md")
	if err != nil {
		t.Fatal(err)
	}
	document, err := Render(source, "guide/setup.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<table>", `type="checkbox"`, "<del>removed text</del>",
		`href="https://example.com"`, `<code class="language-go">`,
		`href="/guide/reference.md#fragments"`, `href="/README.md"`,
		`src="/images/architecture.svg"`,
	} {
		if !strings.Contains(document.HTML, want) {
			t.Errorf("fixture output does not contain %q:\n%s", want, document.HTML)
		}
	}
	if strings.Contains(document.HTML, "<script") || strings.Contains(document.HTML, "unsafe fixture content") {
		t.Fatalf("fixture active content was not removed:\n%s", document.HTML)
	}

	reference, err := os.ReadFile("../../testdata/docs/guide/reference.md")
	if err != nil {
		t.Fatal(err)
	}
	referenceDocument, err := Render(reference, "guide/reference.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(referenceDocument.HTML, `id="fragments"`) || !strings.Contains(referenceDocument.HTML, `id="fragments-1"`) {
		t.Fatalf("duplicate fixture heading IDs are missing:\n%s", referenceDocument.HTML)
	}
}
