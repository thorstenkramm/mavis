package pdf

import (
	"strings"

	basepdf "github.com/signintech/gopdf"
	"github.com/stephenafamo/goldmark-pdf/gopdf"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// outlineHeading is one heading of the source document, found before rendering.
type outlineHeading struct {
	level int
	title string
}

// outlineEntry is a heading that was rendered and received a bookmark.
type outlineEntry struct {
	level    int
	title    string
	bookmark *basepdf.OutlineObj
}

// documentOutline turns the document's headings into the PDF's bookmark tree,
// the navigation pane of PDF viewers. Headings are collected from the syntax
// tree, bookmarked at the position where they are drawn, and linked into a
// hierarchy once rendering is complete.
type documentOutline struct {
	headings map[string]outlineHeading
	entries  []outlineEntry
	// bookmark records a destination at the backend's current position. It is
	// nil for backends without outline support; entries are still collected.
	bookmark func(title string) *basepdf.OutlineObj
}

func newDocumentOutline(backend any) *documentOutline {
	outline := &documentOutline{headings: make(map[string]outlineHeading)}
	if implementation, ok := backend.(*gopdf.Impl); ok && implementation.GoPdf != nil {
		outline.bookmark = implementation.GoPdf.AddOutlineWithPosition
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

// outlineTitle is the heading's plain text on one line. Bookmark titles are
// stored as UTF-16 by a library that cannot encode characters outside the
// Basic Multilingual Plane, so those are left out.
func outlineTitle(heading ast.Node, source []byte) string {
	var title strings.Builder
	err := ast.Walk(heading, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := node.(type) {
		case *ast.Text:
			title.Write(node.Segment.Value(source))
			if node.SoftLineBreak() || node.HardLineBreak() {
				title.WriteByte(' ')
			}
		case *ast.String:
			title.Write(node.Value)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		panic(err)
	}
	plain := strings.Map(func(r rune) rune {
		if r > 0xFFFF || r == 0xFEFF {
			return -1
		}
		return r
	}, title.String())
	return strings.Join(strings.Fields(plain), " ")
}

// mark bookmarks the heading with this anchor at the current position.
func (outline *documentOutline) mark(anchor string) {
	heading, ok := outline.headings[anchor]
	if !ok {
		return
	}
	entry := outlineEntry{level: heading.level, title: heading.title}
	if outline.bookmark != nil {
		entry.bookmark = outline.bookmark(heading.title)
	}
	outline.entries = append(outline.entries, entry)
}

// outlineLink places one entry in the bookmark tree by entry index; -1 means
// none.
type outlineLink struct {
	parent, previous, next, first, last int
}

// outlineLinks derives the tree from heading levels in document order: an
// entry belongs to the nearest preceding entry with a smaller level, so a
// document that skips levels or starts below H1 still nests sensibly.
func outlineLinks(levels []int) []outlineLink {
	links := make([]outlineLink, len(levels))
	lastChild := map[int]int{}
	var ancestors []int
	for index, level := range levels {
		for len(ancestors) > 0 && levels[ancestors[len(ancestors)-1]] >= level {
			ancestors = ancestors[:len(ancestors)-1]
		}
		link := outlineLink{parent: -1, previous: -1, next: -1, first: -1, last: -1}
		if len(ancestors) > 0 {
			link.parent = ancestors[len(ancestors)-1]
		}
		if previous, ok := lastChild[link.parent]; ok {
			link.previous = previous
			links[previous].next = index
		} else if link.parent >= 0 {
			links[link.parent].first = index
		}
		if link.parent >= 0 {
			links[link.parent].last = index
		}
		lastChild[link.parent] = index
		links[index] = link
		ancestors = append(ancestors, index)
	}
	return links
}

// link replaces the flat chain the backend builds while bookmarks are added
// with the heading hierarchy. Top-level entries keep the outline root as
// their parent.
func (outline *documentOutline) link() {
	if outline.bookmark == nil {
		return
	}
	levels := make([]int, len(outline.entries))
	for index, entry := range outline.entries {
		levels[index] = entry.level
	}
	object := func(index int) int {
		if index < 0 {
			return -1
		}
		return outline.entries[index].bookmark.GetIndex()
	}
	for index, link := range outlineLinks(levels) {
		bookmark := outline.entries[index].bookmark
		bookmark.SetPrev(object(link.previous))
		bookmark.SetNext(object(link.next))
		if link.parent >= 0 {
			bookmark.SetParent(object(link.parent))
		}
		if link.first >= 0 {
			bookmark.SetFirst(object(link.first))
			bookmark.SetLast(object(link.last))
		}
	}
}
