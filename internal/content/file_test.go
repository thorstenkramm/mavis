package content

import (
	"bytes"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestFileHandlerContentTypes(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"image.png":              {0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0},
		"notes.txt":              []byte("plain text\n"),
		"manual.pdf":             []byte("%PDF-1.7\nexample\n"),
		"disguised.txt":          []byte("%PDF-1.7\nexample\n"),
		"download.unknown-mavis": {0x00, 0x01, 0x02, 0x03, 0x04},
		"sniffed":                {0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0},
	}
	for name, content := range files {
		makeFileBytes(t, filepath.Join(root, name), content)
	}
	handler := NewFileHandler(newResolver(t, root, config.Default()), nil)

	tests := []struct {
		name        string
		path        string
		contentType string
	}{
		{name: "PNG image", path: "/image.png", contentType: "image/png"},
		{name: "plain text", path: "/notes.txt", contentType: "text/plain; charset=utf-8"},
		{name: "PDF", path: "/manual.pdf", contentType: "application/pdf"},
		{name: "PDF signature overrides text extension", path: "/disguised.txt", contentType: "application/pdf"},
		{name: "unknown binary", path: "/download.unknown-mavis", contentType: "application/octet-stream"},
		{name: "sniffed content", path: "/sniffed", contentType: "image/png"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serveFileRequest(handler, test.path)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); got != test.contentType {
				t.Fatalf("Content-Type = %q, want %q", got, test.contentType)
			}
			if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
			}
			policy := response.Header().Get("Content-Security-Policy")
			frameOptions := response.Header().Get("X-Frame-Options")
			if test.contentType == "application/pdf" && (!strings.Contains(policy, "frame-ancestors 'self'") || frameOptions != "SAMEORIGIN") {
				t.Errorf("raw PDF policy = %q, X-Frame-Options = %q", policy, frameOptions)
			}
			if test.contentType != "application/pdf" && (!strings.Contains(policy, "frame-ancestors 'none'") || frameOptions != "") {
				t.Errorf("raw non-PDF policy = %q, X-Frame-Options = %q", policy, frameOptions)
			}
			if !bytes.Equal(response.Body.Bytes(), files[strings.TrimPrefix(test.path, "/")]) {
				t.Fatalf("body differs from source: %v", response.Body.Bytes())
			}
		})
	}
}

func TestFileHandlerMissingContentIsNotRevealing(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, ".mavis.conf"), "private = true\n")
	handler := NewFileHandler(newResolver(t, root, config.Default()), nil)

	for _, path := range []string{"/missing.txt", "/.mavis.conf", "/../outside.txt"} {
		response := serveFileRequest(handler, path)
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %q status = %d, want 404", path, response.Code)
		}
		if strings.Contains(response.Body.String(), root) || strings.Contains(response.Body.String(), "outside.txt") {
			t.Fatalf("GET %q exposed filesystem details: %q", path, response.Body.String())
		}
		if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("GET %q X-Content-Type-Options = %q", path, got)
		}
	}
}

func TestFileHandlerUnreadableContentIsGenericAndIsolated(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "unreadable.txt"), "secret")
	makeFile(t, filepath.Join(root, "available.txt"), "available")
	resolver := newResolver(t, root, config.Default())
	var logs bytes.Buffer
	handler, ok := NewFileHandler(resolver, log.New(&logs, "", 0)).(*fileHandler)
	if !ok {
		t.Fatal("NewFileHandler() returned an unexpected handler type")
	}
	handler.open = func(path string) (openedFile, error) {
		if strings.HasSuffix(path, "unreadable.txt") {
			return nil, errors.New("open /private/system/path: permission denied")
		}
		return os.Open(path)
	}

	response := serveFileRequest(handler, "/unreadable.txt")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if response.Body.String() != "Internal Server Error\n" {
		t.Fatalf("body = %q, want generic server error", response.Body.String())
	}
	if strings.Contains(response.Body.String(), "/private/system/path") || !strings.Contains(logs.String(), "/private/system/path") {
		t.Fatalf("error details leaked incorrectly: body = %q, logs = %q", response.Body.String(), logs.String())
	}

	available := serveFileRequest(handler, "/available.txt")
	if available.Code != http.StatusOK || available.Body.String() != "available" {
		t.Fatalf("unrelated file response = %d %q", available.Code, available.Body.String())
	}
}

func TestFileHandlerMapsDisappearingFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "notes.txt"), "notes")
	if err := os.Mkdir(filepath.Join(root, "folder"), 0o700); err != nil {
		t.Fatal(err)
	}
	resolver := newResolver(t, root, config.Default())
	handler, ok := NewFileHandler(resolver, nil).(*fileHandler)
	if !ok {
		t.Fatal("NewFileHandler() returned an unexpected handler type")
	}
	handler.open = func(string) (openedFile, error) {
		return nil, fs.ErrNotExist
	}
	if response := serveFileRequest(handler, "/notes.txt"); response.Code != http.StatusNotFound {
		t.Fatalf("disappearing file status = %d, want 404", response.Code)
	}
	if response := serveFileRequest(handler, "/folder/"); response.Code != http.StatusNotFound {
		t.Fatalf("directory status = %d, want 404", response.Code)
	}
}

func TestFileHandlerReadsCurrentSource(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	makeFile(t, path, "before")
	handler := NewFileHandler(newResolver(t, root, config.Default()), nil)
	if response := serveFileRequest(handler, "/notes.txt"); response.Body.String() != "before" {
		t.Fatalf("first body = %q", response.Body.String())
	}
	makeFile(t, path, "after")
	if response := serveFileRequest(handler, "/notes.txt"); response.Body.String() != "after" {
		t.Fatalf("refreshed body = %q", response.Body.String())
	}
}

func serveFileRequest(handler http.Handler, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func makeFileBytes(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}
