package docx

import (
	"encoding/base64"
	"fmt"
	stdhtml "html"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/thorstenkramm/mavis/internal/render"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var imageMIMETypes = map[string]string{
	".gif":  "image/gif",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".svg":  "image/svg+xml",
}

func (converter *Converter) articleHTML(source []byte, documentPath string) (string, error) {
	document, err := converter.renderer.Render(source, documentPath, converter.authorize)
	if err != nil {
		return "", fmt.Errorf("render document: %w", err)
	}

	// The renderer already sanitizes its output. Sanitize at this trust boundary as
	// well so alternate Renderer implementations cannot pass active content to pandoc.
	fragment := render.NewSanitizer().Sanitize(document.HTML)
	context := &xhtml.Node{Type: xhtml.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := xhtml.ParseFragment(strings.NewReader(fragment), context)
	if err != nil {
		return "", fmt.Errorf("parse rendered document: %w", err)
	}
	article := &xhtml.Node{Type: xhtml.ElementNode, Data: "article", DataAtom: atom.Article}
	for _, node := range nodes {
		article.AppendChild(node)
	}
	remainingResources := int64(maxOutputSize)
	converter.inlineImagesInNode(article, documentPath, &remainingResources)

	var body strings.Builder
	if err := xhtml.Render(&body, article); err != nil {
		return "", fmt.Errorf("serialize rendered document: %w", err)
	}
	page := "<!doctype html><html><head><meta charset=\"utf-8\"><title>" +
		stdhtml.EscapeString(document.Title) +
		"</title></head><body>" + body.String() + "</body></html>"
	if len(page) > maxOutputSize {
		return "", fmt.Errorf("self-contained article exceeds %d bytes", maxOutputSize)
	}
	return page, nil
}

func (converter *Converter) authorize(rootRelativePath string) bool {
	if converter.resolver == nil {
		return false
	}
	_, err := converter.resolver.ResolveLogical(rootRelativePath)
	return err == nil
}

func (converter *Converter) inlineImagesInNode(node *xhtml.Node, documentPath string, remaining *int64) {
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		converter.inlineImagesInNode(child, documentPath, remaining)
		child = next
	}
	if node.Type != xhtml.ElementNode || node.DataAtom != atom.Img {
		return
	}

	sourceIndex := -1
	for index, attribute := range node.Attr {
		if attribute.Key == "src" {
			sourceIndex = index
			break
		}
	}
	if sourceIndex < 0 {
		if node.Parent != nil {
			node.Parent.RemoveChild(node)
		}
		return
	}
	dataURI, ok := converter.imageDataURI(node.Attr[sourceIndex].Val, documentPath, remaining)
	if !ok {
		// Remove the whole image, rather than leaving a URL which pandoc could
		// interpret as a resource to retrieve.
		if node.Parent != nil {
			node.Parent.RemoveChild(node)
		}
		return
	}
	node.Attr[sourceIndex].Val = dataURI
	attributes := node.Attr[:0]
	for _, attribute := range node.Attr {
		switch attribute.Key {
		case "srcset", "crossorigin", "loading":
			continue
		}
		attributes = append(attributes, attribute)
	}
	node.Attr = attributes
}

func (converter *Converter) imageDataURI(source, documentPath string, remaining *int64) (dataURI string, ok bool) {
	logicalPath, ok := localImagePath(source, documentPath)
	if !ok || converter.resolver == nil {
		return "", false
	}
	resource, err := converter.resolver.ResolveLogical(logicalPath)
	if err != nil || resource.Info.IsDir() || !resource.Info.Mode().IsRegular() {
		return "", false
	}
	mimeType, ok := imageMIMETypes[strings.ToLower(path.Ext(resource.TargetPath))]
	if !ok {
		return "", false
	}

	file, err := os.Open(resource.Path)
	if err != nil {
		return "", false
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			dataURI = ""
			ok = false
		}
	}()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(resource.Info, info) || info.Size() > maxOutputSize {
		return "", false
	}
	encodedSize := int64(base64.StdEncoding.EncodedLen(int(info.Size())) + len("data:;base64,") + len(mimeType))
	if remaining == nil || encodedSize > *remaining {
		return "", false
	}
	data, err := io.ReadAll(io.LimitReader(file, maxOutputSize+1))
	if err != nil || int64(len(data)) > maxOutputSize {
		return "", false
	}
	dataURI = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
	*remaining -= int64(len(dataURI))
	return dataURI, true
}

func localImagePath(source, documentPath string) (string, bool) {
	if source == "" || strings.TrimSpace(source) != source || strings.ContainsAny(source, "\\\x00\r\n") {
		return "", false
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Opaque != "" ||
		strings.HasPrefix(source, "//") || parsed.Path == "" || strings.Contains(parsed.Path, "\\") {
		return "", false
	}

	if strings.HasPrefix(parsed.Path, "/") {
		// Root-relative paths must already be canonical. ResolveLogical performs
		// segment, volume, and decoded-path validation.
		return strings.TrimPrefix(parsed.Path, "/"), true
	}
	base := path.Dir(strings.TrimPrefix(documentPath, "/"))
	if base == "." {
		base = ""
	}
	logicalPath := path.Join(base, parsed.Path)
	if logicalPath == "." || logicalPath == ".." || strings.HasPrefix(logicalPath, "../") || strings.HasPrefix(logicalPath, "/") {
		return "", false
	}
	return logicalPath, true
}
