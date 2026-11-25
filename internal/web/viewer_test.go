package web

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
)

func TestClassifyViewer(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 1, 2}
	for _, test := range []struct {
		name   string
		source []byte
		want   viewerKind
	}{
		{name: "notes.txt", source: []byte("plain text\n"), want: viewerText},
		{name: "source.go", source: []byte("package main\n"), want: viewerText},
		{name: "markup.html", source: []byte("<h1>source</h1>"), want: viewerText},
		{name: "misleading.png", source: []byte("not an image"), want: viewerText},
		{name: "misleading.txt", source: png, want: viewerImage},
		{name: "vector.svg", source: []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), want: viewerImage},
		{name: "fake.svg", source: []byte("plain text"), want: viewerText},
		{name: "manual.txt", source: []byte("%PDF-1.7\ncontent"), want: viewerPDF},
		{name: "manual.pdf", source: []byte("not a PDF"), want: viewerText},
		{name: "archive.bin", source: []byte{0x00, 0x01, 0x02}, want: viewerNone},
		{name: "large.bin", source: append(bytes.Repeat([]byte("a"), viewerSampleSize), 0x00), want: viewerText},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyViewer(test.name, test.source); got != test.want {
				t.Fatalf("classifyViewer() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestTextViewerEscapesSourceAndRefreshes(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "example.html", "<script>alert('source')</script>\n")
	writeWebFile(t, root, "notes.txt", "before\n")
	handler := newWebHandler(t, root, nil, nil)

	page := requestWeb(handler, "/example.html").Body.String()
	for _, want := range []string{
		`<pre class="text-viewer"><code>&lt;script&gt;alert(&#39;source&#39;)&lt;/script&gt;`,
		`<a class="navigation-link" href="/example.html" aria-current="page">`,
		`<span class="navigation-name">example.html</span>`,
		`<a href="/example.html?raw=1">View raw file</a>`,
		`aria-label="Documentation files"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("text viewer does not contain %q: %s", want, page)
		}
	}
	if strings.Contains(page, `<script>alert`) {
		t.Fatalf("viewer created active source markup: %s", page)
	}
	if refreshed := requestWeb(handler, "/notes.txt").Body.String(); !strings.Contains(refreshed, ">before\n</code></pre>") {
		t.Fatalf("initial text = %s", refreshed)
	}
	writeWebFile(t, root, "notes.txt", "after\n")
	if refreshed := requestWeb(handler, "/notes.txt").Body.String(); !strings.Contains(refreshed, ">after\n</code></pre>") {
		t.Fatalf("refreshed text = %s", refreshed)
	}
}

func TestImageAndPDFViewersUseSharedLayout(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "image.png", string([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}))
	writeWebFile(t, root, "manual.pdf", "%PDF-1.7\ncontent\n")
	handler := newWebHandler(t, root, nil, nil)

	image := requestWeb(handler, "/image.png")
	for _, want := range []string{
		`<img class="image-viewer" src="/image.png?raw=1" alt="image.png">`,
		`<a href="/image.png?raw=1">View raw file</a>`,
		`id="file-navigation"`, `class="reading-layout"`,
	} {
		if !strings.Contains(image.Body.String(), want) {
			t.Errorf("image viewer does not contain %q: %s", want, image.Body.String())
		}
	}
	pdf := requestWeb(handler, "/manual.pdf")
	for _, want := range []string{
		`<iframe class="pdf-viewer" src="/manual.pdf?raw=1" title="manual.pdf PDF viewer"></iframe>`,
		`<a href="/manual.pdf?raw=1">View raw file</a>`,
		`id="file-navigation"`, `class="reading-layout"`,
	} {
		if !strings.Contains(pdf.Body.String(), want) {
			t.Errorf("PDF viewer does not contain %q: %s", want, pdf.Body.String())
		}
	}
	policy := pdf.Header().Get("Content-Security-Policy")
	for _, want := range []string{"connect-src 'self'", "frame-src 'self'", "object-src 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(policy, want) {
			t.Errorf("application CSP %q does not contain %q", policy, want)
		}
	}
	if got := pdf.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("viewer page X-Frame-Options = %q, want DENY", got)
	}
	// The served raw PDF must be frameable by the viewer under both header generations.
	rawPDF := requestWeb(handler, "/manual.pdf?raw=1")
	if got := rawPDF.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" ||
		!strings.Contains(rawPDF.Header().Get("Content-Security-Policy"), "frame-ancestors 'self'") {
		t.Errorf("raw PDF framing headers = %+v", rawPDF.Header())
	}
	if got := requestWeb(handler, "/image.png?raw=1").Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("raw image X-Frame-Options = %q, want DENY", got)
	}
}

func TestTextViewerReadFailureIsGeneric(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "notes.txt", "content")
	handler := newFailingReadHandler(t, root)
	response := requestWeb(handler, "/notes.txt")
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "private filesystem detail") {
		t.Fatalf("read failure = %d %s", response.Code, response.Body.String())
	}
}

func TestOnlyTextViewersReadCompleteFiles(t *testing.T) {
	root := t.TempDir()
	binary := string(append([]byte{0x00, 0x01, 0x02}, bytes.Repeat([]byte{0xff}, 2*viewerSampleSize)...))
	writeWebFile(t, root, "archive.bin", binary)
	writeWebFile(t, root, "image.png", string([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}))
	writeWebFile(t, root, "manual.pdf", "%PDF-1.7\ncontent\n")
	writeWebFile(t, root, "vector.svg", `<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	handler := newFailingReadHandler(t, root)

	archive := requestWeb(handler, "/archive.bin")
	if archive.Code != http.StatusOK || archive.Body.String() != binary {
		t.Fatalf("raw-only file = %d with %d bytes", archive.Code, archive.Body.Len())
	}
	for path, want := range map[string]string{
		"/image.png":  `<img class="image-viewer" src="/image.png?raw=1"`,
		"/manual.pdf": `<iframe class="pdf-viewer" src="/manual.pdf?raw=1"`,
		"/vector.svg": `<img class="image-viewer" src="/vector.svg?raw=1"`,
	} {
		response := requestWeb(handler, path)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), want) {
			t.Errorf("GET %s = %d, want viewer containing %q: %s", path, response.Code, want, response.Body.String())
		}
	}
}

func TestInvalidUTF8TextRetainsRawResponse(t *testing.T) {
	root := t.TempDir()
	invalid := "valid prefix " + string([]byte{0xff, 0xfe}) + "\n"
	writeWebFile(t, root, "notes.txt", invalid)
	late := strings.Repeat("a", viewerSampleSize) + string([]byte{0xc3, 0x28})
	writeWebFile(t, root, "late.txt", late)
	handler := newWebHandler(t, root, nil, nil)

	for path, want := range map[string]string{"/notes.txt": invalid, "/late.txt": late} {
		response := requestWeb(handler, path)
		if response.Code != http.StatusOK || response.Body.String() != want {
			t.Errorf("GET %s = %d %q, want original bytes", path, response.Code, response.Body.String())
		}
		if strings.HasPrefix(response.Header().Get("Content-Type"), "text/html") {
			t.Errorf("GET %s used an HTML viewer for invalid UTF-8", path)
		}
	}
}

func TestViewerSampleIsBoundedAndReportsMissingFiles(t *testing.T) {
	root := t.TempDir()
	if _, err := readViewerSample(filepath.Join(root, "missing.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("readViewerSample(missing) error = %v", err)
	}
	writeWebFile(t, root, "large.txt", strings.Repeat("a", 2*viewerSampleSize))
	writeWebFile(t, root, "empty.txt", "")
	for name, want := range map[string]int{"large.txt": viewerSampleSize, "empty.txt": 0} {
		sample, err := readViewerSample(filepath.Join(root, name))
		if err != nil || len(sample) != want {
			t.Errorf("readViewerSample(%s) = %d bytes, %v; want %d", name, len(sample), err, want)
		}
	}
}

// newFailingReadHandler fails every complete-file read while samples still work.
func newFailingReadHandler(t *testing.T, root string) *webHandler {
	t.Helper()
	resolver, err := content.NewResolver(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	pages, err := template.ParseFS(embedded, "templates/page.html")
	if err != nil {
		t.Fatal(err)
	}
	return &webHandler{
		resolver: resolver,
		files:    content.NewFileHandler(resolver, nil),
		pages:    pages,
		logger:   log.New(io.Discard, "", 0),
		readFile: func(string) ([]byte, error) {
			return nil, errors.New("private filesystem detail")
		},
	}
}

func TestExcludedContentHasNoViewerOrRawResponse(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "drafts/secret.txt", "private content")
	configuration := config.Default()
	configuration.Exclude = []string{"drafts/**"}
	resolver, err := content.NewResolver(root, configuration)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandlerWithRenderer(resolver, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/drafts/secret.txt", "/drafts/secret.txt?raw=1"} {
		response := requestWeb(handler, url)
		if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "private content") {
			t.Errorf("GET %s = %d %s", url, response.Code, response.Body.String())
		}
	}
}

func TestViewerAndRawRepresentations(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "notes.txt", "plain content\n")
	writeWebFile(t, root, "binary.dat", string([]byte{0x00, 0x01, 0x02}))
	handler := newWebHandler(t, root, nil, nil)

	viewer := requestWeb(handler, "/notes.txt")
	if viewer.Code != http.StatusOK || !strings.HasPrefix(viewer.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("text viewer = %d, headers %+v", viewer.Code, viewer.Header())
	}
	raw := requestWeb(handler, "/notes.txt?raw=1")
	if raw.Code != http.StatusOK || raw.Body.String() != "plain content\n" || raw.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("raw text = %d, %q, headers %+v", raw.Code, raw.Body.String(), raw.Header())
	}
	binary := requestWeb(handler, "/binary.dat")
	if binary.Code != http.StatusOK || binary.Body.String() != string([]byte{0x00, 0x01, 0x02}) || binary.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("raw-only file = %d, %q, headers %+v", binary.Code, binary.Body.String(), binary.Header())
	}
}
