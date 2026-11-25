package pdf

import (
	"bytes"
	"io"
	"strings"

	goldpdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/stephenafamo/goldmark-pdf/gopdf"
	"github.com/thorstenkramm/mavis/internal/export"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	pdfdoc "seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/destination"
	"seehuhn.de/go/pdf/outline"
	"seehuhn.de/go/pdf/pagetree"
)

// outlineHeading is one heading of the source document, found before rendering.
type outlineHeading struct {
	level int
	title string
}

// outlineEntry is a heading that was rendered and received a bookmark.
type outlineEntry struct {
	outlineHeading
	page int
	top  float64
}

// documentOutline turns the document's headings into the PDF's bookmark tree,
// the navigation pane of PDF viewers. Headings are collected from the syntax
// tree, bookmarked at the position where they are drawn, and linked into a
// hierarchy once rendering is complete.
type documentOutline struct {
	headings map[string]outlineHeading
	entries  []outlineEntry
	backend  *gopdf.Impl
}

func newDocumentOutline(backend any) *documentOutline {
	outline := &documentOutline{headings: make(map[string]outlineHeading)}
	if implementation, ok := backend.(*gopdf.Impl); ok && implementation.GoPdf != nil {
		outline.backend = implementation
	}
	return outline
}

// Transform records every heading under the ID that the renderer later uses
// as its anchor.
func (outline *documentOutline) Transform(document *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		heading, ok := node.(*ast.Heading)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		id, _ := heading.AttributeString("id")
		anchor, ok := id.([]byte)
		if title := outlineTitle(heading, source); ok && title != "" {
			outline.headings[string(anchor)] = outlineHeading{level: heading.Level, title: title}
		}
		return ast.WalkSkipChildren, nil
	})
	if err != nil {
		panic(err)
	}
}

// outlineTitle is the heading's plain text on one line.
func outlineTitle(heading ast.Node, source []byte) string {
	return strings.Join(strings.Fields(headingText(heading, source)), " ")
}

func headingText(heading ast.Node, source []byte) string {
	var title strings.Builder
	err := ast.Walk(heading, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := node.(type) {
		case *ast.Text:
			value := node.Segment.Value(source)
			if !node.IsRaw() {
				value = util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations(value)))
			}
			title.Write(value)
			if node.HardLineBreak() {
				title.WriteByte('\n')
			} else if node.SoftLineBreak() {
				title.WriteByte(' ')
			}
		case *ast.String:
			title.Write(node.Value)
		case *ast.AutoLink:
			title.Write(node.Label(source))
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		panic(err)
	}
	return title.String()
}

// mark bookmarks the heading with this anchor at the current position.
func (outline *documentOutline) mark(anchor string) {
	heading, ok := outline.headings[anchor]
	if !ok {
		return
	}
	entry := outlineEntry{outlineHeading: heading}
	if backend := outline.backend; backend != nil {
		_, height := backend.GetPageSize()
		entry.page, entry.top = backend.GoPdf.GetNumberOfPages(), height-backend.GetY()
	}
	outline.entries = append(outline.entries, entry)
}

// gopdf's outline serializer cannot set the root endpoints or branch counts.
// Append a catalog update to the bounded, locally generated PDF, leaving its
// page streams, fonts, images and annotations byte-for-byte intact.
func (o *documentOutline) write(backend goldpdf.PDF, out io.Writer) error {
	if o.backend == nil || len(o.entries) == 0 {
		return backend.Write(out)
	}
	base := export.NewOutputBuffer()
	if err := backend.Write(base); err != nil {
		return err
	}
	if err := base.Err(); err != nil {
		return err
	}
	data := base.Bytes()
	w, err := pdfdoc.NewUpdaterTo(bytes.NewReader(data), int64(len(data)), out, nil)
	if err != nil {
		return err
	}
	pages, err := pagetree.FindPages(w)
	if err != nil {
		return err
	}
	var root outline.Outline
	var parents [7]*outline.Item
	for _, e := range o.entries {
		var parent *outline.Item
		for level := e.level - 1; level > 0 && parent == nil; level-- {
			parent = parents[level]
		}
		title := pdfdoc.Text(e.title)
		var item *outline.Item
		if parent == nil {
			item = root.AddItem(title)
		} else {
			item = parent.AddChild(title)
		}
		item.Open = true
		item.Destination = &destination.XYZ{Page: pages[e.page-1], Top: e.top, Zoom: destination.Unset}
		parents[e.level] = item
		clear(parents[e.level+1:])
	}
	rm := pdfdoc.NewResourceManager(w)
	ref, err := rm.Store(&root)
	if err != nil {
		return err
	}
	if err := rm.Close(); err != nil {
		return err
	}
	catalog := *w.GetMeta().Catalog
	catalog.Outlines, catalog.PageMode = ref, "UseOutlines"
	w.GetMeta().Catalog = &catalog
	return w.Close()
}
