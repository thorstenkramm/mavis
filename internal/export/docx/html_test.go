package docx

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

type rendererFunc func([]byte, string, render.AccessFunc) (render.Document, error)

func (function rendererFunc) Render(source []byte, documentPath string, authorize render.AccessFunc) (render.Document, error) {
	return function(source, documentPath, authorize)
}

func TestArticleHTMLPreservesBodyAndEmbedsEligibleImages(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"docs/images/pixel.png":  {0x89, 'P', 'N', 'G', 0, 1, 2},
		"docs/images/photo.jpg":  {0xff, 0xd8, 0xff, 3, 4},
		"docs/images/motion.gif": []byte("GIF89a-test"),
		"docs/images/vector.svg": []byte(`<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg>`),
		"docs/reference.md":      []byte("# Reference\n"),
	}
	for name, data := range files {
		writeFixture(t, filepath.Join(root, name), data)
	}
	converter := New("/unused/pandoc", newTestResolver(t, root, config.Default()), render.NewRenderer(), nil)
	source := []byte(`---
title: front-matter-secret
owner: private-metadata
---
# Current H1

Current **body**.

| Feature | Status |
| --- | --- |
| Tables | retained |

- [x] Complete
- [ ] Pending

[Guide](reference.md?mode=full#part)

` + "```go\nfmt.Println(\"code\")\n```\n\n" +
		`![PNG](images/pixel.png)

![JPEG](images/photo.jpg)

![GIF](images/motion.gif)

![SVG](images/vector.svg)
`)
	article, err := converter.articleHTML(source, "docs/guide.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<!doctype html>", "<article>", `<h1 id="current-h1">Current H1</h1>`,
		"Current <strong>body</strong>", "<table>", "<th>Feature</th>", "<td>Tables</td>",
		`type="checkbox"`, `checked=""`, `disabled=""`,
		`href="/docs/reference.md?mode=full#part"`, `<code class="language-go">`,
		`data:image/png;base64,` + base64.StdEncoding.EncodeToString(files["docs/images/pixel.png"]),
		`data:image/jpeg;base64,` + base64.StdEncoding.EncodeToString(files["docs/images/photo.jpg"]),
		`data:image/gif;base64,` + base64.StdEncoding.EncodeToString(files["docs/images/motion.gif"]),
		`data:image/svg+xml;base64,` + base64.StdEncoding.EncodeToString(files["docs/images/vector.svg"]),
	} {
		if !strings.Contains(article, want) {
			t.Errorf("article does not contain %q:\n%s", want, article)
		}
	}
	for _, unwanted := range []string{
		"front-matter-secret", "private-metadata", "<nav", "<aside", "table-of-contents",
		`src="/docs/images`, "?raw=1", root,
	} {
		if strings.Contains(article, unwanted) {
			t.Errorf("article contains forbidden content %q:\n%s", unwanted, article)
		}
	}
}

func TestArticleHTMLSanitizesAlternateRendererOutput(t *testing.T) {
	renderer := rendererFunc(func([]byte, string, render.AccessFunc) (render.Document, error) {
		return render.Document{
			Title:       `A <title> & "quoted"`,
			HTML:        `<h1>Safe</h1><script>alert(1)</script><iframe src="https://example.com">frame</iframe><img src="x" onerror="alert(2)"><a href="javascript:alert(3)">bad link</a><p>Body</p>`,
			FrontMatter: "front-matter-secret",
		}, nil
	})
	converter := New("/unused/pandoc", nil, renderer, nil)
	article, err := converter.articleHTML(nil, "guide.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{
		"<script", "alert(1)", "<iframe", "onerror", "alert(2)", "javascript:", "alert(3)",
		"front-matter-secret", `<title>A <title>`,
	} {
		if strings.Contains(article, unwanted) {
			t.Errorf("article contains unsafe content %q: %s", unwanted, article)
		}
	}
	for _, want := range []string{"<h1>Safe</h1>", "<p>Body</p>", `<title>A &lt;title&gt; &amp; &#34;quoted&#34;</title>`} {
		if !strings.Contains(article, want) {
			t.Errorf("article does not contain %q: %s", want, article)
		}
	}
}

func TestArticleHTMLRemovesUnauthorizedAndNonLocalImages(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"docs/images/excluded.png", "docs/images/.hidden.png", "docs/images/not-image.txt", ".mavis.conf",
	} {
		writeFixture(t, filepath.Join(root, name), []byte("secret-"+name))
	}
	configuration := config.Default()
	configuration.Exclude = []string{"docs/images/excluded.png"}
	converter := New("/unused/pandoc", newTestResolver(t, root, configuration), render.NewRenderer(), nil)

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()

	sources := []string{
		"images/excluded.png", "images/.hidden.png", "/.mavis.conf", "images/not-image.txt",
		"../../../outside.png", "/docs/../outside.png", `..\outside.png`, "missing.png",
		server.URL + "/remote.png", "//example.com/remote.png", "https://example.com/remote.png",
		"file:///etc/passwd", "data:image/png;base64,c2VjcmV0", "javascript:alert(1)", "C:/Windows/win.ini",
	}
	var markdown strings.Builder
	for _, source := range sources {
		markdown.WriteString("![](" + source + ")\n\n")
	}
	article, err := converter.articleHTML([]byte(markdown.String()), "docs/guide.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(article, "<img") || strings.Contains(article, "data:") {
		t.Fatalf("unsafe or unsupported images remained in article: %s", article)
	}
	for _, secret := range []string{"excluded.png", ".hidden.png", "not-image.txt", "outside.png", "c2VjcmV0"} {
		if strings.Contains(article, secret) {
			t.Errorf("article exposes denied resource %q: %s", secret, article)
		}
	}
	if requests != 0 {
		t.Fatalf("remote image requests = %d, want zero", requests)
	}
}

func TestArticleHTMLRejectsOutsideRootAndPrivateAliases(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	writeFixture(t, filepath.Join(outside, "outside.png"), []byte("outside-secret"))
	writeFixture(t, filepath.Join(root, ".mavis.conf"), []byte("private-secret"))
	if err := os.MkdirAll(filepath.Join(root, "docs/images"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "outside.png"), filepath.Join(root, "docs/images/outside.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, ".mavis.conf"), filepath.Join(root, "docs/images/private.png")); err != nil {
		t.Fatal(err)
	}
	converter := New("/unused/pandoc", newTestResolver(t, root, config.Default()), render.NewRenderer(), nil)
	article, err := converter.articleHTML([]byte("![](images/outside.png)\n\n![](images/private.png)\n"), "docs/guide.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(article, "<img") || strings.Contains(article, "outside-secret") || strings.Contains(article, "private-secret") {
		t.Fatalf("outside-root or private image was embedded: %s", article)
	}
}

func TestLocalImagePathBoundaries(t *testing.T) {
	tests := []struct {
		source string
		want   string
		ok     bool
	}{
		{source: "images/a.png?raw=1#preview", want: "docs/images/a.png", ok: true},
		{source: "../shared/a.png", want: "shared/a.png", ok: true},
		{source: "/shared/a.png", want: "shared/a.png", ok: true},
		{source: "../../../a.png"},
		{source: "/docs/../a.png", want: "docs/../a.png", ok: true}, // resolver rejects non-canonical input
		{source: `..\a.png`},
		{source: "https://example.com/a.png"},
		{source: "//example.com/a.png"},
		{source: "data:image/png;base64,AA=="},
		{source: "file:///etc/passwd"},
		{source: "C:/Windows/win.ini"},
		{source: ""},
		{source: " images/a.png"},
		{source: "%zz"},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got, ok := localImagePath(test.source, "docs/guide.md")
			if got != test.want || ok != test.ok {
				t.Fatalf("localImagePath(%q) = %q, %v; want %q, %v", test.source, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestImageDataURIRemovesDisappearingAndUnsupportedFiles(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, "docs/pixel.png")
	writeFixture(t, name, []byte("png"))
	writeFixture(t, filepath.Join(root, "docs/notes.txt"), []byte("text"))
	converter := New("/unused/pandoc", newTestResolver(t, root, config.Default()), nil, nil)
	remaining := int64(maxOutputSize)
	if _, ok := converter.imageDataURI("pixel.png", "docs/guide.md", &remaining); !ok {
		t.Fatal("eligible PNG was not embedded")
	}
	if _, ok := converter.imageDataURI("notes.txt", "docs/guide.md", &remaining); ok {
		t.Fatal("unsupported text resource was embedded")
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if _, ok := converter.imageDataURI("pixel.png", "docs/guide.md", &remaining); ok {
		t.Fatal("missing image was embedded")
	}
}

func TestImageDataURIEnforcesCumulativeBudget(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "docs/pixel.png"), []byte("png"))
	converter := New("/unused/pandoc", newTestResolver(t, root, config.Default()), nil, nil)
	unlimited := int64(maxOutputSize)
	dataURI, ok := converter.imageDataURI("pixel.png", "docs/guide.md", &unlimited)
	if !ok {
		t.Fatal("eligible PNG was not embedded")
	}
	remaining := int64(len(dataURI))
	if _, ok := converter.imageDataURI("pixel.png", "docs/guide.md", &remaining); !ok || remaining != 0 {
		t.Fatalf("first budgeted embedding = %v, remaining %d", ok, remaining)
	}
	if _, ok := converter.imageDataURI("pixel.png", "docs/guide.md", &remaining); ok {
		t.Fatal("repeated image exceeded cumulative resource budget")
	}
}

func newTestResolver(t *testing.T, root string, configuration config.Config) *content.Resolver {
	t.Helper()
	resolver, err := content.NewResolver(root, configuration)
	if err != nil {
		t.Fatal(err)
	}
	return resolver
}

func writeFixture(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
