package web

import (
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/render"
)

func TestParseFrontMatterIsSchemaFree(t *testing.T) {
	tree, ok := parseFrontMatter("owner: someone\napprover: ''\ntags: []\nmeta: {}\ncount: 3\nnothing:\n" +
		"gaps:\n  - field: general\n    notes: |\n      first line\n      second line\n  - plain\n" +
		"nested:\n  deeper:\n    deepest: value\n")
	if !ok || len(tree) != 8 {
		t.Fatalf("tree = %+v, ok = %v", tree, ok)
	}
	if frontMatterCount(tree) != "8 fields" || frontMatterCount(tree[:1]) != "1 field" {
		t.Fatalf("count labels = %q, %q", frontMatterCount(tree), frontMatterCount(tree[:1]))
	}
	want := []frontMatterNode{
		{Key: "owner", Value: "someone"},
		{Key: "approver", Value: "empty", Empty: true},
		{Key: "tags", Value: "empty list", Empty: true},
		{Key: "meta", Value: "empty map", Empty: true},
		{Key: "count", Value: "3"},
		{Key: "nothing", Value: "empty", Empty: true},
	}
	for index, expected := range want {
		got := tree[index]
		if got.Key != expected.Key || got.Value != expected.Value || got.Empty != expected.Empty || got.Composite {
			t.Errorf("node %d = %+v, want %+v", index, got, expected)
		}
	}
	gaps := tree[6]
	if !gaps.Composite || gaps.Summary != "list · 2 items" || len(gaps.Children) != 2 {
		t.Fatalf("list node = %+v", gaps)
	}
	first := gaps.Children[0]
	if first.Key != "1" || first.Summary != "map · 2 keys" || first.Children[1].Value != "first line\nsecond line\n" {
		t.Fatalf("list item = %+v", first)
	}
	if gaps.Children[1].Key != "2" || gaps.Children[1].Value != "plain" {
		t.Fatalf("scalar list item = %+v", gaps.Children[1])
	}
	nested := tree[7]
	if nested.Summary != "map · 1 key" || nested.Children[0].Children[0].Value != "value" {
		t.Fatalf("nested map = %+v", nested)
	}
}

func TestParseFrontMatterRootShapesAndFailures(t *testing.T) {
	if list, ok := parseFrontMatter("- one\n- two\n"); !ok || len(list) != 2 || list[1].Key != "2" {
		t.Fatalf("root list = %+v, %v", list, ok)
	}
	if scalar, ok := parseFrontMatter("just text\n"); !ok || len(scalar) != 1 || scalar[0].Value != "just text" {
		t.Fatalf("root scalar = %+v, %v", scalar, ok)
	}
	for _, source := range []string{"", "# only a comment\n", "key: [unclosed\n", "~\n"} {
		if tree, ok := parseFrontMatter(source); ok {
			t.Errorf("parseFrontMatter(%q) = %+v, want fallback", source, tree)
		}
	}
	complexKey, ok := parseFrontMatter("? [a, b]\n: value\n")
	if !ok || complexKey[0].Key != "(complex key)" {
		t.Fatalf("complex key = %+v, %v", complexKey, ok)
	}
}

func TestParseFrontMatterBoundsAliasesDepthAndSize(t *testing.T) {
	aliases, ok := parseFrontMatter("base: &base\n  a: 1\ncopy: *base\n")
	if !ok || aliases[1].Value != "*base" || aliases[1].Composite {
		t.Fatalf("alias was expanded: %+v", aliases)
	}

	deep := strings.Repeat("[", maxFrontMatterDepth+4) + strings.Repeat("]", maxFrontMatterDepth+4)
	node, ok := parseFrontMatter("k: " + deep + "\n")
	if !ok {
		t.Fatal("deep front matter was rejected")
	}
	current, depth := node[0], 0
	for current.Composite {
		current, depth = current.Children[0], depth+1
	}
	if current.Value != frontMatterElided || depth > maxFrontMatterDepth {
		t.Fatalf("depth = %d, leaf = %+v", depth, current)
	}

	var wide strings.Builder
	for index := 0; index < maxFrontMatterNodes+50; index++ {
		wide.WriteString("- item\n")
	}
	items, ok := parseFrontMatter(wide.String())
	if !ok || items[len(items)-1].Value != frontMatterElided || items[0].Value != "item" {
		t.Fatalf("oversized front matter was not elided: last = %+v", items[len(items)-1])
	}
}

func TestFrontMatterPanelRendersNestedTreeAndRawFallback(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "nested.md", "---\nowner: a\ngaps:\n  - field: <b>x</b>\n---\n# Nested\n")
	writeWebFile(t, root, "broken.md", "---\nkey: [unclosed\n---\n# Broken\n")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)

	nested := requestWeb(handler, "/nested.md").Body.String()
	for _, want := range []string{
		`<span class="front-matter-count">2 fields</span>`,
		`<details class="front-matter-branch"><summary><span class="front-matter-key">gaps</span><span class="front-matter-summary">list · 1 item</span></summary>`,
		`<span class="front-matter-key">field</span><span class="front-matter-value">&lt;b&gt;x&lt;/b&gt;</span>`,
	} {
		if !strings.Contains(nested, want) {
			t.Errorf("nested front matter does not contain %q: %s", want, nested)
		}
	}
	if strings.Contains(nested, " open>") {
		t.Fatalf("front matter or a branch is expanded by default: %s", nested)
	}

	broken := requestWeb(handler, "/broken.md").Body.String()
	if !strings.Contains(broken, `<summary><span class="front-matter-title">Front matter</span></summary><pre><code class="language-yaml">key: [unclosed`) ||
		strings.Contains(broken, "front-matter-list") {
		t.Fatalf("unparseable front matter did not fall back to source: %s", broken)
	}
}
