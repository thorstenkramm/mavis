package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/render"
)

func TestRealPandocIntegration(t *testing.T) {
	executable, err := exec.LookPath("pandoc")
	if err != nil {
		t.Skip("pandoc is not installed")
	}
	root := t.TempDir()
	// A valid 1x1 transparent PNG exercises resource extraction in real pandoc.
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0xf0,
		0x1f, 0x00, 0x05, 0x00, 0x01, 0xff, 0x89, 0x99,
		0x3d, 0x1d, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
		0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	writeFixture(t, filepath.Join(root, "docs/pixel.png"), png)
	converter := New(executable, newTestResolver(t, root, config.Default()), render.NewRenderer(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var destination bytes.Buffer
	source := []byte("---\ntitle: excluded metadata\n---\n# Integration Heading\n\n| A | B |\n| --- | --- |\n| one | two |\n\n![Pixel](pixel.png)\n")
	if err := converter.Convert(ctx, source, "docs/guide.md", &destination); err != nil {
		t.Fatal(err)
	}
	if err := validateDOCX(destination.Bytes()); err != nil {
		t.Fatalf("real pandoc returned invalid DOCX: %v", err)
	}
	archive, err := zip.NewReader(bytes.NewReader(destination.Bytes()), int64(destination.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var document string
	media := 0
	for _, entry := range archive.File {
		if strings.HasPrefix(entry.Name, "word/media/") {
			media++
		}
		if entry.Name != "word/document.xml" {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		if err := reader.Close(); err != nil {
			t.Fatal(err)
		}
		if err != nil {
			t.Fatal(err)
		}
		document = string(data)
	}
	if !strings.Contains(document, "Integration Heading") || !strings.Contains(document, "one") || !strings.Contains(document, "two") {
		t.Fatalf("DOCX document does not preserve body content: %s", document)
	}
	if strings.Contains(document, "excluded metadata") {
		t.Fatal("DOCX document contains front matter")
	}
	if media == 0 {
		t.Fatal("DOCX does not contain the authorized PNG")
	}
}
