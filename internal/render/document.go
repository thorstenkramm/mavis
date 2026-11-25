package render

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Document is one rendered Markdown document and its current-document TOC.
type Document struct {
	HTML           string
	TOC            []TOCItem
	Title          string
	FrontMatter    string
	HasFrontMatter bool
}

// TOCItem links a heading to its generated fragment and nested subheadings.
type TOCItem struct {
	Title    string
	ID       string
	Fragment string
	Level    int
	Children []TOCItem
}

// AccessFunc reports whether a normalized root-relative content path is eligible.
type AccessFunc func(rootRelativePath string) bool

type renderFunc func(source []byte, documentPath string, authorize AccessFunc) (Document, error)

// Renderer renders independent documents without shared request state.
type Renderer struct {
	render renderFunc
}

// NewRenderer constructs an independent Markdown renderer.
func NewRenderer() *Renderer {
	return &Renderer{render: renderDocument}
}

// Render renders one document or returns an isolated request error.
func (renderer *Renderer) Render(source []byte, documentPath string, authorize AccessFunc) (Document, error) {
	return renderer.render(source, documentPath, authorize)
}

// Render parses Markdown once and renders HTML with matching heading navigation.
func Render(source []byte, documentPath string, authorize AccessFunc) (Document, error) {
	return NewRenderer().Render(source, documentPath, authorize)
}

// FallbackDocument returns a safe, non-revealing rendering-error page fragment.
func FallbackDocument() Document {
	return Document{HTML: "<p>This document could not be rendered. Try another document or check the source file.</p>"}
}

func renderDocument(source []byte, documentPath string, authorize AccessFunc) (Document, error) {
	frontMatter, body, hasFrontMatter := splitFrontMatter(source)
	markdown := NewMarkdown()
	ctx := parser.NewContext()
	// Keep document fragments distinct from controls in web/templates/page.html.
	for _, reserved := range []string{"main-content", "navigation-toggle", "file-navigation", "navigation-close", "navigation-backdrop", "sidebar-resizer", "search-region", "documentation-search"} {
		ctx.IDs().Put([]byte(reserved))
	}
	parsed := markdown.Parser().Parse(text.NewReader(body), parser.WithContext(ctx))
	if err := rewriteLinks(parsed, documentPath, authorize); err != nil {
		return Document{}, fmt.Errorf("rewrite document links: %w", err)
	}
	toc, title, err := extractTOC(parsed, body)
	if err != nil {
		return Document{}, fmt.Errorf("extract table of contents: %w", err)
	}
	var output bytes.Buffer
	if err := markdown.Renderer().Render(&output, body, parsed); err != nil {
		return Document{}, fmt.Errorf("render Markdown: %w", err)
	}
	return Document{
		HTML:           NewSanitizer().Sanitize(output.String()),
		TOC:            toc,
		Title:          title,
		FrontMatter:    frontMatter,
		HasFrontMatter: hasFrontMatter,
	}, nil
}

func extractTOC(document ast.Node, source []byte) ([]TOCItem, string, error) {
	var roots []TOCItem
	var stack []*TOCItem
	var title string
	var titleFound bool
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node.Kind() != ast.KindHeading {
			return ast.WalkContinue, nil
		}
		heading, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkStop, fmt.Errorf("unexpected heading node %T", node)
		}
		id, err := headingID(heading)
		if err != nil {
			return ast.WalkStop, err
		}
		item := TOCItem{
			Title:    headingText(heading, source),
			ID:       id,
			Fragment: "#" + id,
			Level:    heading.Level,
		}
		if heading.Level == 1 && !titleFound {
			title = item.Title
			titleFound = true
		}
		for len(stack) > 0 && stack[len(stack)-1].Level >= item.Level {
			stack = stack[:len(stack)-1]
		}
		var current *TOCItem
		if len(stack) == 0 {
			roots = append(roots, item)
			current = &roots[len(roots)-1]
		} else {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, item)
			current = &parent.Children[len(parent.Children)-1]
		}
		stack = append(stack, current)
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, "", err
	}
	return roots, title, nil
}

func headingText(heading *ast.Heading, source []byte) string {
	return plainText(heading, source)
}

func appendInlineText(output *strings.Builder, node ast.Node, source []byte) {
	switch inline := node.(type) {
	case *ast.Text:
		output.Write(inline.Value(source))
		if inline.SoftLineBreak() || inline.HardLineBreak() {
			output.WriteByte(' ')
		}
	case *ast.String:
		output.Write(inline.Value)
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		appendInlineText(output, child, source)
	}
}

func headingID(heading *ast.Heading) (string, error) {
	value, ok := heading.AttributeString("id")
	if !ok {
		return "", fmt.Errorf("heading at level %d has no generated ID", heading.Level)
	}
	switch id := value.(type) {
	case []byte:
		return string(id), nil
	case string:
		return id, nil
	default:
		return "", fmt.Errorf("heading ID has unexpected type %T", value)
	}
}
