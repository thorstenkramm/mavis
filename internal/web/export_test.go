package web

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	documentexport "github.com/thorstenkramm/mavis/internal/export"
	"github.com/thorstenkramm/mavis/internal/render"
)

type fakeExporter struct {
	result documentexport.Result
	err    error
	docx   bool
	format documentexport.Format
}

func (exporter *fakeExporter) Export(_ context.Context, _ content.Resource, format documentexport.Format) (documentexport.Result, error) {
	exporter.format = format
	return exporter.result, exporter.err
}

func (exporter *fakeExporter) DOCXAvailable() bool { return exporter.docx }

func TestGeneratedDownloadResponses(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "Handbuch.md", "# Handbuch")
	exporter := &fakeExporter{result: documentexport.Result{
		Body:        []byte("%PDF-1.7\ncomplete"),
		Filename:    "Handbuch.pdf",
		ContentType: "application/pdf",
	}}
	handler := newExportWebHandler(t, root, exporter)
	response := requestWeb(handler, "/Handbuch.md?download=pdf")
	if response.Code != http.StatusOK || response.Body.String() != "%PDF-1.7\ncomplete" {
		t.Fatalf("PDF response = %d %q", response.Code, response.Body.String())
	}
	if exporter.format != documentexport.PDF {
		t.Fatalf("export format = %q", exporter.format)
	}
	mediaType, parameters, err := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
	if err != nil || mediaType != "attachment" || parameters["filename"] != "Handbuch.pdf" {
		t.Fatalf("disposition = %q, parameters = %+v, error = %v", response.Header().Get("Content-Disposition"), parameters, err)
	}
	if response.Header().Get("Content-Length") != strconv.Itoa(len(exporter.result.Body)) ||
		response.Header().Get("Content-Type") != "application/pdf" ||
		response.Header().Get("Cache-Control") != "no-store" ||
		response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("PDF headers = %+v", response.Header())
	}

	request := httptest.NewRequest(http.MethodHead, "/Handbuch.md?download=pdf", nil)
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, request)
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Content-Length") == "" {
		t.Fatalf("HEAD response = %d %q, headers = %+v", head.Code, head.Body.String(), head.Header())
	}
}

func TestGeneratedDownloadErrorsAreGeneric(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "unknown", err: documentexport.ErrUnknownFormat, status: http.StatusBadRequest},
		{name: "unavailable", err: documentexport.ErrUnavailable, status: http.StatusNotFound},
		{name: "source too large", err: documentexport.ErrSourceTooLarge, status: http.StatusRequestEntityTooLarge},
		{name: "output too large", err: documentexport.ErrOutputTooLarge, status: http.StatusInternalServerError},
		{name: "timeout", err: documentexport.ErrTimeout, status: http.StatusServiceUnavailable},
		{name: "conversion", err: fmt.Errorf("%w: /private/path pandoc details", documentexport.ErrConversion), status: http.StatusInternalServerError},
		{name: "panic", err: errors.New("private panic"), status: http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeWebFile(t, root, "guide.md", "# Guide")
			handler := newExportWebHandler(t, root, &fakeExporter{err: test.err})
			response := requestWeb(handler, "/guide.md?download=docx")
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			if strings.Contains(response.Body.String(), "/private/path") || strings.Contains(response.Body.String(), "pandoc details") ||
				strings.Contains(response.Body.String(), "private panic") {
				t.Fatalf("error leaked details: %q", response.Body.String())
			}
		})
	}
}

func TestGeneratedDownloadWithoutServiceIsUnavailable(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "guide.md", "# Guide")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	for _, format := range []string{"pdf", "docx"} {
		if response := requestWeb(handler, "/guide.md?download="+format); response.Code != http.StatusNotFound {
			t.Errorf("%s without service = %d, want 404", format, response.Code)
		}
	}
}

func newExportWebHandler(t *testing.T, root string, exporter Exporter) http.Handler {
	t.Helper()
	resolver, err := content.NewResolver(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandlerWithExporter(resolver, render.NewRenderer(), exporter, nil, config.EditTemplate{})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
