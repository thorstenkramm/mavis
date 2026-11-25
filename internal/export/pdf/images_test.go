package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
)

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func testResolver(t *testing.T, root string, configuration config.Config) *content.Resolver {
	t.Helper()
	resolver, err := content.NewResolver(root, configuration)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}
	return resolver
}

func rasterImage(t *testing.T, format string) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 3, 2))
	picture.Set(0, 0, color.RGBA{R: 255, A: 255})
	picture.Set(1, 0, color.RGBA{G: 255, A: 255})
	picture.Set(2, 0, color.RGBA{B: 255, A: 255})
	var output bytes.Buffer
	var err error
	switch format {
	case "png":
		err = png.Encode(&output, picture)
	case "jpeg":
		err = jpeg.Encode(&output, picture, nil)
	case "gif":
		err = gif.Encode(&output, picture, nil)
	default:
		t.Fatalf("unknown test format %q", format)
	}
	if err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestConvertLocalRasterImagesAndVisiblePlaceholders(t *testing.T) {
	root := t.TempDir()
	images := map[string][]byte{
		"png":  rasterImage(t, "png"),
		"jpeg": rasterImage(t, "jpeg"),
		"gif":  rasterImage(t, "gif"),
	}
	for format, data := range images {
		extension := format
		if extension == "jpeg" {
			extension = "jpg"
		}
		writeTestFile(t, filepath.Join(root, "docs", "images", "pixel."+extension), data)
	}
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>private-svg-payload</text></svg>`)
	writeTestFile(t, filepath.Join(root, "docs", "images", "diagram.svg"), svg)
	writeTestFile(t, filepath.Join(root, "docs", "images", "unknown.bin"), []byte("unsupported-image-payload"))
	resolver := testResolver(t, root, config.Default())
	converter, err := New(resolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := recordingConverter(t, converter)
	source := []byte(`![PNG alt](images/pixel.png)
![JPEG alt](images/pixel.jpg)
![GIF alt](images/pixel.gif)
![SVG alt](images/diagram.svg)
![Other alt](images/unknown.bin)
`)
	if err := converter.Convert(context.Background(), source, "docs/guide.md", io.Discard); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(recorder.images) != 5 {
		t.Fatalf("registered images = %d, want 5: %+v", len(recorder.images), recorder.images)
	}
	byID := make(map[string]recordedImage, len(recorder.images))
	for _, image := range recorder.images {
		byID[image.id] = image
	}
	for _, test := range []struct {
		id     string
		format string
		data   []byte
	}{
		{id: "images/pixel.png", format: "png", data: images["png"]},
		{id: "images/pixel.jpg", format: "jpeg", data: images["jpeg"]},
		{id: "images/pixel.gif", format: "gif", data: images["gif"]},
	} {
		registered := byID[test.id]
		if registered.format != test.format || !bytes.Equal(registered.data, test.data) {
			t.Errorf("%s registered format %q, %d bytes; want %q and original %d bytes",
				test.id, registered.format, len(registered.data), test.format, len(test.data))
		}
	}
	for _, id := range []string{"images/diagram.svg", "images/unknown.bin"} {
		registered := byID[id]
		if registered.format != "png" || !bytes.Equal(registered.data, previewUnavailablePNG) {
			t.Errorf("%s did not use the fixed PNG placeholder", id)
		}
		if bytes.Contains(registered.data, []byte("payload")) {
			t.Errorf("%s leaked unsupported source bytes", id)
		}
	}
	captions := strings.Join(recorder.cells, " ")
	for _, alt := range []string{"PNG alt", "JPEG alt", "GIF alt", "SVG alt", "Other alt"} {
		if !strings.Contains(captions, alt) {
			t.Errorf("alt text %q is missing from captions %q", alt, captions)
		}
	}

	placeholder, format, err := image.DecodeConfig(bytes.NewReader(previewUnavailablePNG))
	if err != nil || format != "png" || placeholder.Width != 480 || placeholder.Height != 120 {
		t.Fatalf("embedded placeholder config = %+v, format %q, error %v", placeholder, format, err)
	}

	realConverter, err := New(resolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := realConverter.Convert(context.Background(), source, "docs/guide.md", &output); err != nil {
		t.Fatalf("real Convert(images) error = %v", err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatal("real image conversion did not produce a PDF")
	}
}

func TestConvertRejectsUnauthorizedImagesWithoutNetworkOrFailure(t *testing.T) {
	root := t.TempDir()
	pixel := rasterImage(t, "png")
	writeTestFile(t, filepath.Join(root, "images", "pixel.png"), pixel)
	writeTestFile(t, filepath.Join(root, ".secret.png"), []byte("hidden-secret"))
	writeTestFile(t, filepath.Join(root, "excluded", "secret.png"), []byte("excluded-secret"))
	writeTestFile(t, filepath.Join(root, config.FileName), []byte("private-config-secret"))
	outside := t.TempDir()
	writeTestFile(t, filepath.Join(outside, "secret.png"), []byte("outside-secret"))
	configuration := config.Default()
	configuration.Exclude = []string{"excluded/**"}
	resolver := testResolver(t, root, configuration)
	converter, err := New(resolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := recordingConverter(t, converter)

	var requests atomic.Int64
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, fmt.Errorf("unexpected request for %s", request.URL)
	})
	oldTransport := http.DefaultTransport
	oldClient := http.DefaultClient
	http.DefaultTransport = transport
	http.DefaultClient = &http.Client{Transport: transport}
	t.Cleanup(func() {
		http.DefaultTransport = oldTransport
		http.DefaultClient = oldClient
	})

	source := []byte(`![remote alt](https://example.invalid/secret.png)
![http alt](http://example.invalid/secret.png)
![data alt](data:image/png;base64,AAAA)
![scheme alt](custom:secret)
![absolute alt](/images/pixel.png)
![traversal alt](../secret.png)
![backslash alt](images\pixel.png)
![hidden alt](.secret.png)
![excluded alt](excluded/secret.png)
![private alt](.mavis.conf)
`)
	if err := converter.Convert(context.Background(), source, "guide.md", io.Discard); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(recorder.images) != 0 {
		t.Fatalf("unauthorized images were registered: %+v", recorder.images)
	}
	if requests.Load() != 0 {
		t.Fatalf("unauthorized images caused %d network requests", requests.Load())
	}
	text := strings.Join(recorder.texts, " ")
	for _, alt := range []string{
		"remote alt", "http alt", "data alt", "scheme alt", "absolute alt",
		"traversal alt", "backslash alt", "hidden alt", "excluded alt", "private alt",
	} {
		if !strings.Contains(text, alt) {
			t.Errorf("omitted image lost alt text %q in %q", alt, text)
		}
	}
}

func TestConvertSymlinkImageBoundaries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	root := t.TempDir()
	pixel := rasterImage(t, "png")
	writeTestFile(t, filepath.Join(root, "docs", "images", "pixel.png"), pixel)
	outside := t.TempDir()
	writeTestFile(t, filepath.Join(outside, "secret.png"), []byte("outside-symlink-secret"))
	if err := os.Symlink(filepath.Join(root, "docs", "images", "pixel.png"), filepath.Join(root, "docs", "images", "internal.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.png"), filepath.Join(root, "docs", "images", "outside.png")); err != nil {
		t.Fatal(err)
	}
	converter, err := New(testResolver(t, root, config.Default()), nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := recordingConverter(t, converter)
	source := []byte("![internal](images/internal.png)\n![outside](images/outside.png)\n")
	if err := converter.Convert(context.Background(), source, "docs/guide.md", io.Discard); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(recorder.images) != 1 || recorder.images[0].id != "images/internal.png" ||
		!bytes.Equal(recorder.images[0].data, pixel) {
		t.Fatalf("symlink images = %+v, want only original internal PNG", recorder.images)
	}
}

func TestResourceFSDisappearingAndSizeLimitedFiles(t *testing.T) {
	root := t.TempDir()
	pixel := rasterImage(t, "png")
	imagePath := filepath.Join(root, "docs", "images", "pixel.png")
	writeTestFile(t, imagePath, pixel)
	resolver := testResolver(t, root, config.Default())
	filesystem, err := newResourceFS(resolver, "docs/guide.md")
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.Open("images/pixel.png")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, pixel) {
		t.Fatal("Open() did not return original PNG bytes")
	}

	filesystem.maxSize = int64(len(pixel) - 1)
	if _, err := filesystem.Open("images/pixel.png"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Open(oversize) error = %v, want not-exist", err)
	}
	filesystem.maxSize = maxImageSize
	_, resolved, err := filesystem.resolve("images/pixel.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(imagePath); err != nil {
		t.Fatal(err)
	}
	if _, err := filesystem.read(resolved); err == nil {
		t.Fatal("read(disappeared) error = nil")
	}
	if _, err := filesystem.Open("images/pixel.png"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Open(disappeared) error = %v, want not-exist", err)
	}
	if _, err := filesystem.Open("images"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Open(directory) error = %v, want not-exist", err)
	}
}

func TestMemoryImageMetadata(t *testing.T) {
	file := newMemoryImage([]byte("image"), "image/png", "pixel.png")
	if file.MimeType() != "image/png" {
		t.Fatalf("MimeType() = %q", file.MimeType())
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != "pixel.png" || info.Size() != 5 || info.IsDir() || !info.Mode().IsRegular() ||
		!info.ModTime().IsZero() || info.Sys() != nil {
		t.Fatalf("Stat() = %+v", info)
	}
	if _, err := file.Readdir(1); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("Readdir() error = %v", err)
	}
}

func TestResourceFSReturnsPlaceholderForUnsupportedContent(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "diagram.svg"), []byte("<svg>secret</svg>"))
	filesystem, err := newResourceFS(testResolver(t, root, config.Default()), "docs/guide.md")
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.Open("diagram.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, previewUnavailablePNG) || bytes.Contains(data, []byte("secret")) {
		t.Fatal("SVG Open() did not return only the fixed placeholder")
	}
}
