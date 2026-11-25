package pdf

import (
	"bytes"
	"context"
	"strings"
	"testing"

	pdfdoc "seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/pagetree"
)

func TestRealPDFContainsNestedBookmarks(t *testing.T) {
	converter, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := "### 😀\n\n##### <https://example.com>\n\n###### Repeated\n\n###### Repeated\n\n" + strings.Repeat("paragraph\n\n", 60) + "# A &amp; B \\*draft\\*\n\n### Last\n"
	var output bytes.Buffer
	if err := converter.Convert(context.Background(), []byte(source), "guide.md", &output); err != nil {
		t.Fatal(err)
	}
	reader, err := pdfdoc.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()), nil)
	if err != nil {
		t.Fatal(err)
	}
	cursor := pdfdoc.NewCursor(reader)
	dict := func(obj pdfdoc.Object) pdfdoc.Dict {
		t.Helper()
		d, err := cursor.Dict(obj)
		if err != nil || d == nil {
			t.Fatalf("dictionary %v: %v", obj, err)
		}
		return d
	}
	rootRef := reader.GetMeta().Catalog.Outlines
	root := dict(rootRef)
	a, b := dict(root["First"]), dict(root["Last"])
	child, last := dict(a["First"]), dict(b["Last"])
	leaf1, leaf2 := dict(child["First"]), dict(child["Last"])
	if root["Count"] != pdfdoc.Integer(6) || a["Count"] != pdfdoc.Integer(3) || b["Count"] != pdfdoc.Integer(1) || child["Count"] != pdfdoc.Integer(2) {
		t.Fatalf("incorrect outline counts: %v %v %v %v", root, a, b, child)
	}
	if a["Parent"] != rootRef || b["Parent"] != rootRef || a["Next"] != root["Last"] || b["Prev"] != root["First"] || child["Parent"] != root["First"] || last["Parent"] != root["Last"] || leaf1["Next"] != child["Last"] || leaf2["Prev"] != child["First"] {
		t.Fatal("inconsistent outline references")
	}
	pages, err := pagetree.FindPages(reader)
	if err != nil || len(pages) < 2 {
		t.Fatalf("pages=%v err=%v", pages, err)
	}
	for _, item := range []struct {
		d     pdfdoc.Dict
		title string
		page  pdfdoc.Reference
	}{{a, "😀", pages[0]}, {child, "https://example.com", pages[0]}, {leaf1, "Repeated", pages[0]}, {leaf2, "Repeated", pages[0]}, {b, "A & B *draft*", pages[len(pages)-1]}, {last, "Last", pages[len(pages)-1]}} {
		title, err := cursor.TextString(item.d["Title"])
		if err != nil || title.String() != item.title {
			t.Fatalf("title=%v err=%v, want %q", title, err, item.title)
		}
		dest, err := cursor.Array(item.d["Dest"])
		if err != nil || len(dest) != 5 || dest[0] != item.page || dest[1] != pdfdoc.Name("XYZ") || dest[2] != pdfdoc.Integer(0) {
			t.Fatalf("destination for %s: %v (%v)", item.title, dest, err)
		}
	}
}
