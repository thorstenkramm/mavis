package pdf

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const (
	maxImageSize   = int64(10 * 1024 * 1024)
	maxImagePixels = 8 * 1024 * 1024
	blockedImageID = "mavis-pdf-image-omitted"
)

//go:embed assets/preview-unavailable.png
var previewUnavailablePNG []byte

// resourceFS is the http.FileSystem consumed by goldmark-pdf. It exposes only
// regular files authorized by content.Resolver, relative to one Markdown
// document. Files are copied into bounded memory files before the PDF backend
// sees them, preventing subsequent seeks or file growth from bypassing the cap.
type resourceFS struct {
	resolver  *content.Resolver
	base      string
	maxSize   int64
	maxPixels int
}

func newResourceFS(resolver *content.Resolver, documentPath string) (*resourceFS, error) {
	cleanDocument, err := cleanDocumentPath(documentPath)
	if err != nil {
		return nil, err
	}
	return &resourceFS{
		resolver:  resolver,
		base:      path.Dir(cleanDocument),
		maxSize:   maxImageSize,
		maxPixels: maxImagePixels,
	}, nil
}

func cleanDocumentPath(documentPath string) (string, error) {
	if documentPath == "" || strings.HasPrefix(documentPath, "//") ||
		strings.Contains(documentPath, "\\") || strings.ContainsRune(documentPath, '\x00') {
		return "", errors.New("invalid Markdown document path")
	}
	documentPath = strings.TrimPrefix(documentPath, "/")
	for _, segment := range strings.Split(documentPath, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", errors.New("invalid Markdown document path")
		}
	}
	return path.Clean(documentPath), nil
}

func (filesystem *resourceFS) Open(name string) (http.File, error) {
	if name == blockedImageID {
		return nil, fs.ErrNotExist
	}

	// goldmark-pdf appends its own inline and remote filesystems after this
	// one. Returning a local placeholder for schemed resources is therefore
	// necessary to prevent its fallback from fetching remote URLs or decoding
	// data URIs. The AST transformer normally omits these images before here.
	if hasScheme(name) {
		return newMemoryImage(previewUnavailablePNG, "image/png", "preview-unavailable.png"), nil
	}

	_, resource, err := filesystem.resolve(name)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	data, err := filesystem.read(resource)
	if err != nil {
		return nil, fs.ErrNotExist
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	switch format {
	case "png", "jpeg", "gif":
		if err == nil && config.Width > 0 && config.Height > 0 && config.Width <= filesystem.maxPixels/config.Height {
			return newMemoryImage(data, "image/"+format, filepath.Base(resource.Path)), nil
		}
	}
	return newMemoryImage(previewUnavailablePNG, "image/png", "preview-unavailable.png"), nil
}

func hasScheme(name string) bool {
	parsed, err := url.Parse(name)
	return err == nil && parsed.Scheme != ""
}

// resolve validates an already-decoded relative image name and returns the
// cleaned relative name plus the currently authorized filesystem entry.
func (filesystem *resourceFS) resolve(name string) (string, *content.Resource, error) {
	if name == "" || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "//") ||
		strings.Contains(name, "\\") || strings.ContainsRune(name, '\x00') || hasScheme(name) {
		return "", nil, content.ErrNotFound
	}
	segments := strings.Split(name, "/")
	for _, segment := range segments {
		if segment == "" || segment == ".." || strings.HasPrefix(segment, ".") && segment != "." {
			return "", nil, content.ErrNotFound
		}
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || path.IsAbs(cleaned) {
		return "", nil, content.ErrNotFound
	}
	if filesystem.resolver == nil {
		return "", nil, content.ErrNotFound
	}
	logical := cleaned
	if filesystem.base != "." {
		logical = path.Join(filesystem.base, cleaned)
	}
	resource, err := filesystem.resolver.ResolveLogical(logical)
	if err != nil || !resource.Info.Mode().IsRegular() || resource.Info.Size() > filesystem.maxSize {
		return "", nil, content.ErrNotFound
	}
	return cleaned, resource, nil
}

func (filesystem *resourceFS) read(resource *content.Resource) (data []byte, returnErr error) {
	file, err := os.Open(resource.Path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); returnErr == nil && err != nil {
			returnErr = err
		}
	}()

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(resource.Info, info) ||
		info.Size() < 0 || info.Size() > filesystem.maxSize {
		return nil, content.ErrNotFound
	}
	data, err = io.ReadAll(io.LimitReader(file, filesystem.maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > filesystem.maxSize {
		return nil, fmt.Errorf("image exceeds %d bytes", filesystem.maxSize)
	}
	return data, nil
}

// imageASTTransformer validates the original destination before upstream's
// localPath helper strips leading dots and slashes. Unauthorized destinations
// become a harmless missing local identifier, which also prevents upstream's
// remote and inline fallback filesystems from seeing the original URL.
type imageASTTransformer struct {
	images *resourceFS
}

func (transformer *imageASTTransformer) Transform(
	document *ast.Document,
	_ text.Reader,
	_ parser.Context,
) {
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		imageNode, ok := node.(*ast.Image)
		if ok && entering {
			imageNode.Destination = []byte(transformer.destination(imageNode.Destination))
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		panic(err)
	}
}

func (transformer *imageASTTransformer) destination(destination []byte) string {
	decoded, err := url.PathUnescape(string(destination))
	if err != nil {
		return blockedImageID
	}
	cleaned, _, err := transformer.images.resolve(decoded)
	if err != nil {
		return blockedImageID
	}
	return cleaned
}

type memoryImage struct {
	*bytes.Reader
	name string
	mime string
}

func newMemoryImage(data []byte, mime, name string) *memoryImage {
	return &memoryImage{Reader: bytes.NewReader(data), name: name, mime: mime}
}

func (file *memoryImage) Close() error { return nil }

func (file *memoryImage) Readdir(int) ([]fs.FileInfo, error) {
	return nil, fs.ErrInvalid
}

func (file *memoryImage) Stat() (fs.FileInfo, error) {
	return memoryImageInfo{name: file.name, size: file.Size()}, nil
}

func (file *memoryImage) MimeType() string { return file.mime }

type memoryImageInfo struct {
	name string
	size int64
}

func (info memoryImageInfo) Name() string       { return info.name }
func (info memoryImageInfo) Size() int64        { return info.size }
func (info memoryImageInfo) Mode() fs.FileMode  { return 0o444 }
func (info memoryImageInfo) ModTime() time.Time { return time.Time{} }
func (info memoryImageInfo) IsDir() bool        { return false }
func (info memoryImageInfo) Sys() any           { return nil }
