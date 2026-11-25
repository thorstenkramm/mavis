package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewMarkdown(t *testing.T) {
	var output bytes.Buffer
	if err := NewMarkdown().Convert([]byte("# Heading\n"), &output); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if !strings.Contains(output.String(), `<h1 id="heading">Heading</h1>`) {
		t.Fatalf("Convert() output = %q", output.String())
	}
}

func TestNewSanitizer(t *testing.T) {
	output := NewSanitizer().Sanitize("<strong>safe</strong><script>alert(1)</script>")
	if output != "<strong>safe</strong>" {
		t.Fatalf("Sanitize() = %q", output)
	}
}
