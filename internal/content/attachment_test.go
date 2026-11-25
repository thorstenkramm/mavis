package content

import (
	"mime"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestAttachmentDisposition(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		wantFallback string
	}{
		{name: "ASCII", filename: "guide.md", wantFallback: "guide.md"},
		{name: "Unicode and header specials", filename: "Handbücher/Über \"uns\".md", wantFallback: "Über \"uns\".md"},
		{name: "control characters", filename: "bad\r\nname.txt", wantFallback: "bad__name.txt"},
		{name: "non-ASCII boundary", filename: "bound\u0080ary.txt", wantFallback: "bound\u0080ary.txt"},
		{name: "extended delimiters", filename: "quote'and*star.txt", wantFallback: "quote'and*star.txt"},
		{name: "missing name", filename: "", wantFallback: "download"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header := AttachmentDisposition(test.filename)
			mediaType, parameters, err := mime.ParseMediaType(header)
			if err != nil {
				t.Fatalf("ParseMediaType(%q): %v", header, err)
			}
			if mediaType != "attachment" || parameters["filename"] != test.wantFallback {
				t.Fatalf("disposition = %q, parameters = %+v", header, parameters)
			}
			if strings.ContainsAny(header, "\r\n") {
				t.Fatalf("header contains a line break: %q", header)
			}
			if !utf8.ValidString(parameters["filename"]) {
				t.Fatalf("filename is not UTF-8: %q", parameters["filename"])
			}
		})
	}
}

func TestAttachmentDispositionEscapesExtendedValueDelimiters(t *testing.T) {
	header := AttachmentDisposition("quote'and*star\u0080.txt")
	for _, encoded := range []string{"%27", "%2A", "%C2%80"} {
		if !strings.Contains(header, encoded) {
			t.Errorf("disposition %q does not contain %q", header, encoded)
		}
	}
}

func TestFileHandlerDownloadDispositionAndCurrentBytes(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root+"/Über uns.txt", "before")
	handler := NewFileHandler(newResolver(t, root, config.Default()), nil)

	inline := serveFileRequest(handler, "/%C3%9Cber%20uns.txt?raw=1")
	if inline.Code != http.StatusOK || inline.Header().Get("Content-Disposition") != "" {
		t.Fatalf("inline response = %d, disposition = %q", inline.Code, inline.Header().Get("Content-Disposition"))
	}
	download := serveFileRequest(handler, "/%C3%9Cber%20uns.txt?raw=1&download=1")
	if download.Code != http.StatusOK || download.Body.String() != "before" {
		t.Fatalf("download response = %d %q", download.Code, download.Body.String())
	}
	mediaType, parameters, err := mime.ParseMediaType(download.Header().Get("Content-Disposition"))
	if err != nil || mediaType != "attachment" || parameters["filename"] != "Über uns.txt" {
		t.Fatalf("disposition = %q, parameters = %+v, error = %v", download.Header().Get("Content-Disposition"), parameters, err)
	}
	if download.Header().Get("Cache-Control") != "no-store" || download.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("download headers = %+v", download.Header())
	}

	makeFile(t, root+"/Über uns.txt", "after")
	if refreshed := serveFileRequest(handler, "/%C3%9Cber%20uns.txt?raw=1&download=1"); refreshed.Body.String() != "after" {
		t.Fatalf("refreshed body = %q", refreshed.Body.String())
	}
}

func TestFileHandlerDownloadKeepsContentBoundaries(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root+"/.mavis.conf", "include_hidden = true\n")
	makeFile(t, root+"/.hidden.txt", "secret")
	configuration := config.Default()
	configuration.Exclude = []string{"excluded.txt"}
	makeFile(t, root+"/excluded.txt", "secret")
	handler := NewFileHandler(newResolver(t, root, configuration), nil)

	for _, requestPath := range []string{
		"/missing.txt?raw=1&download=1",
		"/.mavis.conf?raw=1&download=1",
		"/.hidden.txt?raw=1&download=1",
		"/excluded.txt?raw=1&download=1",
		"/../outside.txt?raw=1&download=1",
	} {
		response := serveFileRequest(handler, requestPath)
		if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "secret") {
			t.Errorf("GET %q = %d %q", requestPath, response.Code, response.Body.String())
		}
	}
}
