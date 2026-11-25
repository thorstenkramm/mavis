package pdf

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestOutlineLinksNestByNearestSmallerLevel(t *testing.T) {
	none := -1
	tests := []struct {
		name   string
		levels []int
		want   []outlineLink
	}{
		{name: "empty", levels: nil, want: []outlineLink{}},
		{name: "flat", levels: []int{2, 2}, want: []outlineLink{
			{parent: none, previous: none, next: 1, first: none, last: none},
			{parent: none, previous: 0, next: none, first: none, last: none},
		}},
		{name: "nested with return", levels: []int{1, 2, 3, 3, 2, 1}, want: []outlineLink{
			{parent: none, previous: none, next: 5, first: 1, last: 4},
			{parent: 0, previous: none, next: 4, first: 2, last: 3},
			{parent: 1, previous: none, next: 3, first: none, last: none},
			{parent: 1, previous: 2, next: none, first: none, last: none},
			{parent: 0, previous: 1, next: none, first: none, last: none},
			{parent: none, previous: 0, next: none, first: none, last: none},
		}},
		{name: "skipped and leading lower levels", levels: []int{3, 1, 4, 2}, want: []outlineLink{
			{parent: none, previous: none, next: 1, first: none, last: none},
			{parent: none, previous: 0, next: none, first: 2, last: 3},
			{parent: 1, previous: none, next: 3, first: none, last: none},
			{parent: 1, previous: 2, next: none, first: none, last: none},
		}},
	}
	for _, test := range tests {
		if got := outlineLinks(test.levels); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: outlineLinks(%v) =\n%+v\nwant\n%+v", test.name, test.levels, got, test.want)
		}
	}
}

func TestRecordingBackendCollectsHeadingsWithoutBookmarks(t *testing.T) {
	converter, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	backend := &layoutPDF{}
	converter.newBackend = recordingFactory(backend)
	converter.observeOutline = func(outline *documentOutline) { backend.outline = outline }
	source := "---\ntitle: hidden\n---\n# Title with *emphasis*, `code` and 😀\n\n" +
		"Soft\nbreak <br>and [link](https://example.com)\n---\n\n## Repeated\n\n## Repeated\n\n#\n\n### Deep\n"
	if err := converter.Convert(context.Background(), []byte(source), "guide.md", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if anchors := strings.Join(backend.anchors, ","); !strings.Contains(anchors, "repeated,repeated-1") || !strings.Contains(anchors, "deep") {
		t.Fatalf("heading anchors = %v", backend.anchors)
	}
	outline := backend.outline
	var titles []string
	for _, entry := range outline.entries {
		if entry.bookmark != nil {
			t.Fatalf("backend without outline support received a bookmark: %+v", entry)
		}
		titles = append(titles, fmt.Sprintf("%d:%s", entry.level, entry.title))
	}
	want := []string{"1:Title with emphasis, code and", "2:Soft break and link", "2:Repeated", "2:Repeated", "3:Deep"}
	if !reflect.DeepEqual(titles, want) {
		t.Fatalf("outline entries = %q, want %q", titles, want)
	}
	outline.link() // Must be a no-op without bookmarks.
}

func TestRealPDFContainsNestedBookmarks(t *testing.T) {
	converter, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var source strings.Builder
	source.WriteString("# Grüße & Überblick\n\ntext\n\n")
	for section := 1; section <= 3; section++ {
		fmt.Fprintf(&source, "## Section %d\n\ntext\n\n### Detail %d.1\n\ntext\n\n### Detail %d.2\n\ntext\n\n", section, section, section)
	}
	var output bytes.Buffer
	if err := converter.Convert(context.Background(), []byte(source.String()), "guide.md", &output); err != nil {
		t.Fatal(err)
	}
	document := output.String()
	if !strings.Contains(document, "/Outlines") {
		t.Fatal("PDF catalog has no outline")
	}
	bookmarks := regexp.MustCompile(`(?s)<<\n  /Parent \d+ 0 R\n.*?/Title <FEFF[0-9A-F]+>\n>>`).FindAllString(document, -1)
	if len(bookmarks) != 10 {
		t.Fatalf("PDF has %d bookmarks, want 10", len(bookmarks))
	}
	// "Grüße & Überblick" in UTF-16: the title is stored as text, not glyphs.
	if !strings.Contains(bookmarks[0], "/Title <FEFF004700720000FC00DF0065") && !strings.Contains(bookmarks[0], "/Title <FEFF0047007200FC00DF0065") {
		t.Fatalf("first bookmark title is not the H1: %s", bookmarks[0])
	}
	withChildren, leaves := 0, 0
	for _, bookmark := range bookmarks {
		if strings.Contains(bookmark, "/First ") && strings.Contains(bookmark, "/Last ") {
			withChildren++
		} else {
			leaves++
		}
	}
	if withChildren != 4 || leaves != 6 {
		t.Fatalf("bookmark hierarchy: %d parents and %d leaves, want 4 and 6", withChildren, leaves)
	}
	if strings.Contains(bookmarks[0], "/Prev ") || strings.Contains(bookmarks[0], "/Next ") {
		t.Fatalf("the only top-level bookmark has siblings: %s", bookmarks[0])
	}
}
