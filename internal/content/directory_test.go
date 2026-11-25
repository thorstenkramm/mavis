package content

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestDirectoryListingsAndREADMEIndexes(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "README.md"), "# Root")
	makeFile(t, filepath.Join(root, "z.txt"), "text")
	makeFile(t, filepath.Join(root, "docs", "README.md"), "# Docs")
	makeFile(t, filepath.Join(root, "docs", "guide", "setup.md"), "# Setup")
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	resolver := newResolver(t, root, config.Default())

	rootPage, err := resolver.Directory("/")
	if err != nil {
		t.Fatal(err)
	}
	if rootPage.README == nil || rootPage.README.RelativePath != "README.md" {
		t.Fatalf("root README = %+v", rootPage.README)
	}
	if len(rootPage.Entries) != 4 || !rootPage.Entries[0].Info.IsDir() || resourceName(rootPage.Entries[0]) != "docs" {
		t.Fatalf("root entries are not directory-first and sorted: %+v", rootPage.Entries)
	}

	docsPage, err := resolver.Directory("/docs/")
	if err != nil {
		t.Fatal(err)
	}
	if docsPage.README == nil || docsPage.README.RelativePath != "docs/README.md" {
		t.Fatalf("docs README = %+v", docsPage.README)
	}

	emptyPage, err := resolver.Directory("/empty/")
	if err != nil {
		t.Fatal(err)
	}
	if emptyPage.README != nil || len(emptyPage.Entries) != 0 {
		t.Fatalf("empty page = %+v", emptyPage)
	}
}

func TestDirectoryModelsReflectSourceEditsPerRequest(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "README.md"), "# Original")
	resolver := newResolver(t, root, config.Default())
	first, err := resolver.Directory("/")
	if err != nil || first.README == nil {
		t.Fatalf("first Directory() = %+v, %v", first, err)
	}

	makeFile(t, filepath.Join(root, "new.md"), "# New")
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	second, err := resolver.Directory("/")
	if err != nil {
		t.Fatal(err)
	}
	if second.README != nil || len(second.Entries) != 1 || resourceName(second.Entries[0]) != "new.md" {
		t.Fatalf("updated Directory() = %+v", second)
	}
}

func TestNavigationContainsOnlyCurrentAncestry(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "docs", "guide", "setup.md"), "# Setup")
	makeFile(t, filepath.Join(root, "docs", "guide", "other.md"), "# Other")
	makeFile(t, filepath.Join(root, "unrelated", "child", "grandchild.md"), "# Unrelated")
	resolver := newResolver(t, root, config.Default())

	navigation, err := resolver.Navigation("/docs/guide/setup.md")
	if err != nil {
		t.Fatal(err)
	}
	docs := findNavigation(t, navigation, "docs")
	guide := findNavigation(t, docs.Children, "docs/guide")
	setup := findNavigation(t, guide.Children, "docs/guide/setup.md")
	if !setup.Current || setup.URL != "/docs/guide/setup.md" {
		t.Fatalf("setup navigation = %+v", setup)
	}
	unrelated := findNavigation(t, navigation, "unrelated")
	if len(unrelated.Children) != 0 {
		t.Fatalf("unrelated branch was recursively scanned: %+v", unrelated.Children)
	}
}

func TestNavigationForDirectoryMarksCurrentAndListsChildren(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "docs", "guide.md"), "# Guide")
	resolver := newResolver(t, root, config.Default())
	navigation, err := resolver.Navigation("/docs/")
	if err != nil {
		t.Fatal(err)
	}
	docs := findNavigation(t, navigation, "docs")
	if !docs.Current || len(docs.Children) != 1 || docs.Children[0].RelativePath != "docs/guide.md" {
		t.Fatalf("docs navigation = %+v", docs)
	}
}

func TestDirectoryRejectsFile(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "guide.md"), "# Guide")
	resolver := newResolver(t, root, config.Default())
	if _, err := resolver.Directory("/guide.md"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Directory(file) error = %v", err)
	}
}

func TestURLFor(t *testing.T) {
	tests := []struct {
		path      string
		directory bool
		want      string
	}{
		{path: "", want: "/"},
		{path: ".", want: "/"},
		{path: "docs", directory: true, want: "/docs/"},
		{path: "space dir/setup #1.md", want: "/space%20dir/setup%20%231.md"},
	}
	for _, test := range tests {
		if got := URLFor(test.path, test.directory); got != test.want {
			t.Errorf("URLFor(%q, %v) = %q, want %q", test.path, test.directory, got, test.want)
		}
	}
}

func findNavigation(t *testing.T, items []NavigationItem, path string) NavigationItem {
	t.Helper()
	for _, item := range items {
		if item.RelativePath == path {
			return item
		}
	}
	t.Fatalf("navigation item %q not found in %+v", path, items)
	return NavigationItem{}
}
