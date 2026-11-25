package render

import (
	"reflect"
	"strings"
	"testing"
)

func TestRenderRewritesLocalLinksAndAssets(t *testing.T) {
	source := []byte(`[Setup](../reference/setup.md?mode=full#prerequisites)

![Diagram](images/diagram%20one.png)

![Root image](/images/architecture.png?version=2#preview)

[Root asset](/images/architecture.png#v2)

[Text viewer](../reference/notes.txt)

[Folder](../reference/)

[Current section](#local)
`)
	document, err := Render(source, "docs/guide/current.md", nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, want := range []string{
		`href="/docs/reference/setup.md?mode=full#prerequisites"`,
		`src="/docs/guide/images/diagram%20one.png?raw=1"`,
		`src="/images/architecture.png?raw=1&amp;version=2#preview"`,
		`href="/images/architecture.png#v2"`,
		`href="/docs/reference/notes.txt"`,
		`href="/docs/reference/"`,
		`href="#local"`,
	} {
		if !strings.Contains(document.HTML, want) {
			t.Errorf("HTML does not contain %q:\n%s", want, document.HTML)
		}
	}
}

func TestRenderPreservesExternalURLs(t *testing.T) {
	source := []byte(`[HTTPS](https://example.com/a?x=1#part)

[Email](mailto:docs@example.com)

[Protocol relative](//example.com/manual)

https://example.com/autolink
`)
	document, err := Render(source, "guide.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`href="https://example.com/a?x=1#part"`,
		`href="mailto:docs@example.com"`,
		`href="//example.com/manual"`,
		`href="https://example.com/autolink"`,
	} {
		if !strings.Contains(document.HTML, want) {
			t.Errorf("HTML does not contain unchanged URL %q:\n%s", want, document.HTML)
		}
	}
}

func TestRenderChecksAccessAndRemovesDeniedDestinations(t *testing.T) {
	source := []byte(`[Excluded](../private/secret.md)

![Excluded image](images/secret.png)

[Outside](../../../outside.md)

![Outside image](../../../outside.png)

[Allowed](../public/guide.md)
`)
	var checked []string
	authorize := func(path string) bool {
		checked = append(checked, path)
		return !strings.Contains(path, "private") && !strings.Contains(path, "secret")
	}
	document, err := Render(source, "docs/guide/current.md", authorize)
	if err != nil {
		t.Fatal(err)
	}
	for _, denied := range []string{"private/secret.md", "images/secret.png", "outside.md", "outside.png"} {
		if strings.Contains(document.HTML, denied) {
			t.Errorf("HTML exposes denied destination %q:\n%s", denied, document.HTML)
		}
	}
	if !strings.Contains(document.HTML, `href="/docs/public/guide.md"`) {
		t.Fatalf("allowed link was not retained:\n%s", document.HTML)
	}
	wantChecked := []string{"docs/private/secret.md", "docs/guide/images/secret.png", "docs/public/guide.md"}
	if !reflect.DeepEqual(checked, wantChecked) {
		t.Fatalf("authorized paths = %#v, want %#v", checked, wantChecked)
	}
}

func TestRenderRejectsInvalidAndRootEscapingDestinations(t *testing.T) {
	for _, source := range []string{
		"[Invalid](%zz)\n",
		"[Backslash](..\\secret.md)\n",
		"[Root traversal](/docs/../secret.md)\n",
	} {
		document, err := Render([]byte(source), "docs/guide.md", nil)
		if err != nil {
			t.Fatalf("Render(%q) error = %v", source, err)
		}
		if strings.Contains(document.HTML, "href=") || strings.Contains(document.HTML, "<a ") {
			t.Errorf("Render(%q) did not remove the link:\n%s", source, document.HTML)
		}
	}
}

func TestLinkRewritingDoesNotTouchCode(t *testing.T) {
	document, err := Render([]byte("`[code](../private.md)`\n\n```text\n[ fenced ](../private.md)\n```\n"), "docs/guide.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(document.HTML, "<a ") {
		t.Fatalf("code example became a link:\n%s", document.HTML)
	}
}
