package content

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestSearchLiteralMatching(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		source    string
		wantMatch string
	}{
		{
			name:      "German compound and Unicode whitespace trimming",
			query:     "\u2003SiChErHeIt\t",
			source:    "Informationssicherheit ist wichtig.",
			wantMatch: "sicherheit",
		},
		{
			name:      "Unicode lowercase changes UTF-8 byte length",
			query:     "k",
			source:    "Das Zeichen K steht hier.",
			wantMatch: "K",
		},
		{
			name:      "Capital sharp S uses simple lowercase mapping",
			query:     "groß",
			source:    "GROẞ",
			wantMatch: "GROẞ",
		},
		{
			name:      "multi-word phrase",
			query:     "alpha beta",
			source:    "zero alpha beta gamma",
			wantMatch: "alpha beta",
		},
		{
			name:   "phrase words are not independent terms",
			query:  "alpha gamma",
			source: "zero alpha beta gamma",
		},
		{
			name:   "internal whitespace remains significant",
			query:  "alpha  beta",
			source: "zero alpha beta gamma",
		},
		{
			name:   "no accent folding",
			query:  "fur",
			source: "für",
		},
		{
			name:   "no multi-character case folding",
			query:  "gross",
			source: "groß",
		},
		{
			name:      "raw front matter",
			query:     "title: sicherheit",
			source:    "---\ntitle: Sicherheit\n---\n# Start\n",
			wantMatch: "title: Sicherheit",
		},
		{
			name:      "raw Markdown punctuation",
			query:     "# setup",
			source:    "before\n# Setup\nafter",
			wantMatch: "# Setup",
		},
		{
			name:   "filename-only match is excluded",
			query:  "needle",
			source: "unrelated content",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			makeFile(t, filepath.Join(root, "needle.md"), test.source)
			resolver := newResolver(t, root, config.Default())

			results, err := resolver.Search(context.Background(), test.query)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			if results.Incomplete || results.Truncated {
				t.Fatalf("Search() notices = incomplete %v truncated %v", results.Incomplete, results.Truncated)
			}
			if test.wantMatch == "" {
				if len(results.Items) != 0 {
					t.Fatalf("Search() items = %+v, want none", results.Items)
				}
				return
			}
			if len(results.Items) != 1 {
				t.Fatalf("Search() returned %d items, want one: %+v", len(results.Items), results.Items)
			}
			matches := matchingSearchSegments(results.Items[0].Excerpt)
			if !reflect.DeepEqual(matches, []string{test.wantMatch}) {
				t.Fatalf("match segments = %q, want %q; excerpt = %+v", matches, test.wantMatch, results.Items[0].Excerpt)
			}
		})
	}
}

func TestSearchEmptyQueryDoesNotScan(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "guide.md"), "needle")
	resolver := newResolver(t, root, config.Default())
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}

	for _, query := range []string{"", " \t\n", "\u2003\u00a0"} {
		results, err := resolver.Search(context.Background(), query)
		if err != nil {
			t.Fatalf("Search(%q) error = %v", query, err)
		}
		if len(results.Items) != 0 || results.Incomplete || results.Truncated {
			t.Fatalf("Search(%q) = %+v, want empty results", query, results)
		}
	}
}

func TestSearchExcerptWordContext(t *testing.T) {
	before := make([]string, 10)
	after := make([]string, 14)
	for index := range before {
		before[index] = fmt.Sprintf("b%02d", index)
	}
	for index := range after {
		after[index] = fmt.Sprintf("a%02d", index)
	}
	words := append(append(append([]string{}, before...), "needle"), after...)
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "guide.md"), strings.Join(words, " "))
	resolver := newResolver(t, root, config.Default())

	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Items) != 1 {
		t.Fatalf("Search() items = %+v", results.Items)
	}
	excerpt := searchExcerptText(results.Items[0].Excerpt)
	for _, want := range []string{"b02", "b09", "needle", "a00", "a11"} {
		if !strings.Contains(excerpt, want) {
			t.Errorf("excerpt %q does not contain %q", excerpt, want)
		}
	}
	for _, omitted := range []string{"b01", "a12", "a13"} {
		if strings.Contains(excerpt, omitted) {
			t.Errorf("excerpt %q contains omitted word %q", excerpt, omitted)
		}
	}
	if !strings.HasPrefix(excerpt, "… ") || !strings.HasSuffix(excerpt, " …") {
		t.Fatalf("excerpt %q does not have omission ellipses", excerpt)
	}
}

func TestSearchExcerptHighlights(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		query       string
		wantMatches []string
	}{
		{
			name:        "repeated occurrences preserve source case",
			source:      "needle one NEEDLE two Needle",
			query:       "needle",
			wantMatches: []string{"needle", "NEEDLE", "Needle"},
		},
		{
			name:        "repeated phrase including internal space",
			source:      "alpha beta and ALPHA BETA",
			query:       "alpha beta",
			wantMatches: []string{"alpha beta", "ALPHA BETA"},
		},
		{
			name:        "match inside a longer word",
			source:      "Informationssicherheit und Datensicherheit",
			query:       "sicherheit",
			wantMatches: []string{"sicherheit", "sicherheit"},
		},
		{
			name:        "source markup remains literal segment text",
			source:      `<script>alert("needle")</script>`,
			query:       "needle",
			wantMatches: []string{"needle"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			makeFile(t, filepath.Join(root, "guide.md"), test.source)
			resolver := newResolver(t, root, config.Default())
			results, err := resolver.Search(context.Background(), test.query)
			if err != nil {
				t.Fatal(err)
			}
			if len(results.Items) != 1 {
				t.Fatalf("Search() items = %+v", results.Items)
			}
			matches := matchingSearchSegments(results.Items[0].Excerpt)
			if !reflect.DeepEqual(matches, test.wantMatches) {
				t.Fatalf("matches = %q, want %q; segments = %+v", matches, test.wantMatches, results.Items[0].Excerpt)
			}
			joined := searchExcerptText(results.Items[0].Excerpt)
			if strings.Contains(joined, "…") {
				t.Fatalf("short excerpt %q unexpectedly contains an ellipsis", joined)
			}
		})
	}
}

func TestSearchExcerptRuneCap(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "guide.md"), "needle"+strings.Repeat("🙂", 300))
	resolver := newResolver(t, root, config.Default())

	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Items) != 1 {
		t.Fatalf("Search() items = %+v", results.Items)
	}
	excerpt := searchExcerptText(results.Items[0].Excerpt)
	if !utf8.ValidString(excerpt) {
		t.Fatal("excerpt is not valid UTF-8")
	}
	if got := utf8.RuneCountInString(excerpt); got != maxSearchExcerpt {
		t.Fatalf("excerpt contains %d runes, want %d", got, maxSearchExcerpt)
	}
	if !strings.HasPrefix(excerpt, "needle") || !strings.HasSuffix(excerpt, "…") {
		t.Fatalf("capped excerpt has wrong boundaries: %q", excerpt)
	}
	if matches := matchingSearchSegments(results.Items[0].Excerpt); !reflect.DeepEqual(matches, []string{"needle"}) {
		t.Fatalf("matches = %q, want needle", matches)
	}
}

func TestSearchExcerptCapKeepsMatchAfterLongToken(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "guide.md"), strings.Repeat("ü", 300)+" needle "+strings.Repeat("🙂", 300))
	resolver := newResolver(t, root, config.Default())
	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Items) != 1 {
		t.Fatalf("Search() items = %+v", results.Items)
	}
	excerpt := searchExcerptText(results.Items[0].Excerpt)
	if !utf8.ValidString(excerpt) || utf8.RuneCountInString(excerpt) > maxSearchExcerpt {
		t.Fatalf("capped excerpt is invalid or too long: %q", excerpt)
	}
	if !strings.HasPrefix(excerpt, "… needle ") || !strings.HasSuffix(excerpt, "…") {
		t.Fatalf("match is not preserved with omission ellipses: %q", excerpt)
	}
	if matches := matchingSearchSegments(results.Items[0].Excerpt); !reflect.DeepEqual(matches, []string{"needle"}) {
		t.Fatalf("matches = %q, want needle", matches)
	}
}

func TestExcerptContextBoundedForLargeWordCounts(t *testing.T) {
	source := []rune(strings.Repeat("a ", 100000) + "needle" + strings.Repeat(" b", 100000))
	allocs := testing.AllocsPerRun(5, func() {
		start, end := searchExcerptBounds(source, 200000, 200006)
		if start != 200000-16 || end != 200006+24 {
			t.Fatalf("unexpected window [%d:%d]", start, end)
		}
	})
	if allocs != 0 {
		t.Fatalf("context allocated %v times", allocs)
	}
}

func TestSearchVisitsLeafWithPermanentOmissionsOnce(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "docs/sub/.DS_Store"), "needle")
	makeFile(t, filepath.Join(root, "docs/sub/ignored.txt"), "needle")
	for _, link := range []string{"current", "latest"} {
		if err := os.Symlink("docs", filepath.Join(root, link)); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
	}
	cfg := config.Default()
	cfg.Exclude = []string{"**/ignored.txt"}
	r := newResolver(t, root, cfg)
	reads := 0
	readDir := r.readDir
	r.readDir = func(path string) ([]os.DirEntry, error) {
		if path == filepath.Join(r.root, "docs/sub") {
			reads++
		}
		return readDir(path)
	}
	assertSearchPaths(t, r, "needle", nil)
	if reads != 1 {
		t.Fatalf("read physical leaf %d times, want once", reads)
	}
}

func TestRestrictedAliasDoesNotHideCanonicalMatch(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "docs", "sub", "guide.md"), "needle")
	makeFile(t, filepath.Join(root, "docs", "sub", "shared.md"), "needle")
	if err := os.Symlink("docs", filepath.Join(root, "alias")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	cfg := config.Default()
	cfg.Exclude = []string{"alias/sub/guide.md"}
	assertSearchPaths(t, newResolver(t, root, cfg), "needle", []string{"alias/sub/shared.md", "docs/sub/guide.md"})
}

func TestCancelDuringDirectoryListing(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "guide.md"), "needle")
	r := newResolver(t, root, config.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.readDir = func(path string) ([]os.DirEntry, error) { cancel(); return os.ReadDir(path) }
	listing, err := r.list(ctx, "")
	if !errors.Is(err, context.Canceled) || len(listing.entries) != 0 {
		t.Fatalf("canceled listing: %+v, %v", listing, err)
	}
}

func TestSearchUsesResolverVisibilityAndConfinement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	root := t.TempDir()
	files := []string{
		".mavis.conf",
		".hidden.md",
		".private/secret.md",
		"drafts/secret.md",
		"docs/visible.md",
		"docs/Über % #1.md",
	}
	for _, name := range files {
		makeFile(t, filepath.Join(root, name), "needle")
	}
	outside := t.TempDir()
	makeFile(t, filepath.Join(outside, "outside.md"), "needle")
	if err := os.Symlink(outside, filepath.Join(root, "outside-link")); err != nil {
		t.Fatal(err)
	}
	configuration := config.Default()
	configuration.Exclude = []string{"drafts/**"}
	resolver := newResolver(t, root, configuration)

	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"docs/visible.md", "docs/Über % #1.md"}
	if got := searchResultPaths(results.Items); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
	if results.Incomplete {
		t.Fatal("denied and binary-like entries must not mark results incomplete")
	}
	if got := results.Items[1].Name; got != "Über % #1.md" {
		t.Fatalf("Name = %q, want exact stored spelling", got)
	}
}

func TestSearchDeduplicatesInternalDirectoryAliasesAndCycles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "docs", "nested", "page.md"), "needle")
	if err := os.Symlink(filepath.Join(root, "docs"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(root, "docs", "nested", "back")); err != nil {
		t.Fatal(err)
	}
	resolver := newResolver(t, root, config.Default())

	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alias/nested/page.md"}
	if got := searchResultPaths(results.Items); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %q, want one logical alias %q", got, want)
	}
	if results.Incomplete || results.Truncated {
		t.Fatalf("unexpected notices: %+v", results)
	}
}

func TestSearchResultsUseDeterministicPathOrder(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		"z.md",
		"a/z.md",
		"a.md",
		"a/a.md",
		"B.md",
		"same/x.md",
		"same2/x.md",
	}
	for _, name := range paths {
		makeFile(t, filepath.Join(root, name), "needle")
	}
	sort.Strings(paths)
	resolver := newResolver(t, root, config.Default())

	first, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	second, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	if got := searchResultPaths(first.Items); !reflect.DeepEqual(got, paths) {
		t.Fatalf("paths = %q, want %q", got, paths)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated searches differ:\nfirst: %+v\nsecond: %+v", first, second)
	}
}

func TestSearchTruncatesToFirstHundredPaths(t *testing.T) {
	root := t.TempDir()
	for index := 0; index < 105; index++ {
		makeFile(t, filepath.Join(root, fmt.Sprintf("file%03d.md", index)), "needle")
	}
	makeFile(t, filepath.Join(root, "file0000.md"), "no match")
	resolver := newResolver(t, root, config.Default())

	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Items) != 100 || !results.Truncated || results.Incomplete {
		t.Fatalf("Search() = %d items, truncated %v, incomplete %v", len(results.Items), results.Truncated, results.Incomplete)
	}
	for index, item := range results.Items {
		want := fmt.Sprintf("file%03d.md", index)
		if item.RelativePath != want {
			t.Fatalf("item %d path = %q, want %q", index, item.RelativePath, want)
		}
	}
}

func TestSearchTextDetectionAndFailures(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxSearchFileSize+1)
	copy(oversized, "needle")
	tests := []struct {
		name           string
		files          map[string][]byte
		wantPaths      []string
		wantIncomplete bool
	}{
		{
			name:      "extensionless UTF-8",
			files:     map[string][]byte{"README": []byte("extensionless needle")},
			wantPaths: []string{"README"},
		},
		{
			name:      "unknown extension with detected plain text",
			files:     map[string][]byte{"guide.custom": []byte("plain needle")},
			wantPaths: []string{"guide.custom"},
		},
		{
			name:  "PNG image is silently skipped",
			files: map[string][]byte{"image.png": append([]byte("\x89PNG\r\n\x1a\n"), []byte("needle")...)},
		},
		{
			name:  "image extension is skipped even for ASCII sample",
			files: map[string][]byte{"image.png": []byte("needle")},
		},
		{
			name:  "PDF is silently skipped",
			files: map[string][]byte{"document.pdf": []byte("%PDF-1.7\nneedle")},
		},
		{
			name:  "obvious binary is silently skipped",
			files: map[string][]byte{"data.bin": {0x00, 0x01, 0xff, 'n', 'e', 'e', 'd', 'l', 'e'}},
		},
		{
			name:           "invalid UTF-8 text candidate is incomplete",
			files:          map[string][]byte{"bad.md": {'n', 'e', 'e', 'd', 'l', 'e', 0xff}},
			wantIncomplete: true,
		},
		{
			name: "failure is isolated from healthy files",
			files: map[string][]byte{
				"bad.md":    {0xff},
				"good.md":   []byte("needle"),
				"image.png": []byte("needle"),
			},
			wantPaths:      []string{"good.md"},
			wantIncomplete: true,
		},
		{
			name:           "oversized text candidate is incomplete",
			files:          map[string][]byte{"large.txt": oversized},
			wantIncomplete: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for name, source := range test.files {
				makeFileBytes(t, filepath.Join(root, name), source)
			}
			resolver := newResolver(t, root, config.Default())
			results, err := resolver.Search(context.Background(), "needle")
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			if got := searchResultPaths(results.Items); !reflect.DeepEqual(got, test.wantPaths) {
				t.Fatalf("paths = %q, want %q", got, test.wantPaths)
			}
			if results.Incomplete != test.wantIncomplete {
				t.Fatalf("Incomplete = %v, want %v", results.Incomplete, test.wantIncomplete)
			}
		})
	}
}

func TestSearchAcceptsFileAtReadLimit(t *testing.T) {
	root := t.TempDir()
	source := bytes.Repeat([]byte("a"), maxSearchFileSize)
	copy(source, "needle")
	makeFileBytes(t, filepath.Join(root, "limit.txt"), source)
	resolver := newResolver(t, root, config.Default())

	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	if results.Incomplete || len(results.Items) != 1 {
		t.Fatalf("Search() = %+v, want one complete result", results)
	}
}

func TestSearchDisappearingFileIsIncomplete(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "gone.md"), "needle")
	resolver := newResolver(t, root, config.Default())
	entries, err := resolver.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("List() entries = %+v", entries)
	}
	if err := os.Remove(entries[0].Path); err != nil {
		t.Fatal(err)
	}

	result, incomplete, err := readSearchResult(context.Background(), entries[0], newTestSearchQuery("needle"))
	if err != nil {
		t.Fatalf("readSearchResult() error = %v", err)
	}
	if result != nil || !incomplete {
		t.Fatalf("readSearchResult() = result %+v, incomplete %v; want omitted incomplete file", result, incomplete)
	}
}

func TestSearchEntryDisappearingDuringListingIsIncomplete(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "healthy.md"), "needle")
	makeFile(t, filepath.Join(root, "gone.md"), "needle")
	resolver := newResolver(t, root, config.Default())
	readDir := resolver.readDir
	removed := false
	resolver.readDir = func(name string) ([]os.DirEntry, error) {
		entries, err := readDir(name)
		if err == nil && filepath.Clean(name) == filepath.Clean(resolver.root) && !removed {
			removed = true
			if err := os.Remove(filepath.Join(root, "gone.md")); err != nil {
				return nil, err
			}
		}
		return entries, err
	}
	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	if !removed || !results.Incomplete {
		t.Fatalf("Search() removed = %v, incomplete = %v", removed, results.Incomplete)
	}
	if got := searchResultPaths(results.Items); !reflect.DeepEqual(got, []string{"healthy.md"}) {
		t.Fatalf("paths = %q, want healthy.md", got)
	}
}

func TestSearchInternalAliasTargetDisappearingDuringSpellingCheckIsIncomplete(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "healthy.md"), "needle")
	makeFile(t, filepath.Join(root, "docs", "gone.md"), "needle")
	if err := os.Symlink(filepath.Join(root, "docs"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	resolver := newResolver(t, root, config.Default())
	readDir := resolver.readDir
	rootReads := 0
	resolver.readDir = func(name string) ([]os.DirEntry, error) {
		if filepath.Clean(name) == filepath.Clean(resolver.root) {
			rootReads++
			if rootReads == 2 {
				// The alias has resolved, but its target disappears before the
				// resolved path's exact spelling can be verified.
				if err := os.RemoveAll(filepath.Join(root, "docs")); err != nil {
					return nil, err
				}
			}
		}
		return readDir(name)
	}

	results, err := resolver.Search(context.Background(), "needle")
	if err != nil {
		t.Fatal(err)
	}
	if rootReads < 2 || !results.Incomplete {
		t.Fatalf("Search() root reads = %d, incomplete = %v", rootReads, results.Incomplete)
	}
	if got := searchResultPaths(results.Items); !reflect.DeepEqual(got, []string{"healthy.md"}) {
		t.Fatalf("paths = %q, want healthy.md", got)
	}
}

func TestSearchDescendantDirectoryFailureIsIsolated(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "healthy.md"), "needle")
	makeFile(t, filepath.Join(root, "removed", "gone.md"), "needle")
	resolver := newResolver(t, root, config.Default())
	entries, err := resolver.List("")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "removed")); err != nil {
		t.Fatal(err)
	}
	scanner := searchScanner{
		resolver: resolver,
		query:    newTestSearchQuery("needle"),
		visited:  make(map[string]struct{}),
		active:   map[string]struct{}{".": {}},
		collector: searchCollector{
			items: make([]SearchResult, 0, maxSearchResults+1),
		},
	}
	if _, err := scanner.walk(context.Background(), entries); err != nil {
		t.Fatalf("walk() error = %v", err)
	}
	if !scanner.incomplete {
		t.Fatal("walk() Incomplete = false, want true")
	}
	if got := searchResultPaths(scanner.collector.items); !reflect.DeepEqual(got, []string{"healthy.md"}) {
		t.Fatalf("paths = %q, want healthy.md", got)
	}
}

func TestSearchReflectsEditsAdditionsAndDeletions(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.md")
	b := filepath.Join(root, "b.md")
	makeFile(t, a, "needle")
	makeFile(t, b, "needle")
	resolver := newResolver(t, root, config.Default())

	assertSearchPaths(t, resolver, "needle", []string{"a.md", "b.md"})
	makeFile(t, a, "edited without the query")
	makeFile(t, filepath.Join(root, "c.md"), "needle")
	assertSearchPaths(t, resolver, "needle", []string{"b.md", "c.md"})
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	assertSearchPaths(t, resolver, "needle", []string{"c.md"})
}

func TestSearchRootFailure(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "guide.md"), "needle")
	resolver := newResolver(t, root, config.Default())
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}

	results, err := resolver.Search(context.Background(), "needle")
	if err == nil {
		t.Fatal("Search() error = nil, want root traversal failure")
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Search() root error = %v, want non-context error", err)
	}
	if !strings.HasPrefix(err.Error(), "search content root:") {
		t.Fatalf("Search() error = %q, want generic root context", err)
	}
	if len(results.Items) != 0 || results.Incomplete || results.Truncated {
		t.Fatalf("Search() partial results = %+v", results)
	}
}

func TestSearchContextErrors(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	deadline, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
	defer deadlineCancel()
	tests := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{name: "cancellation", ctx: canceled, want: context.Canceled},
		{name: "deadline expiry", ctx: deadline, want: context.DeadlineExceeded},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			makeFile(t, filepath.Join(root, "guide.md"), "needle")
			resolver := newResolver(t, root, config.Default())
			results, err := resolver.Search(test.ctx, "needle")
			if !errors.Is(err, test.want) {
				t.Fatalf("Search() error = %v, want %v", err, test.want)
			}
			if len(results.Items) != 0 || results.Incomplete || results.Truncated {
				t.Fatalf("Search() partial results = %+v", results)
			}
		})
	}
}

func newTestSearchQuery(query string) searchQuery {
	lower := lowerSearchRunes([]rune(query))
	return searchQuery{lower: lower, prefix: searchPrefixTable(lower)}
}

func assertSearchPaths(t *testing.T, resolver *Resolver, query string, want []string) {
	t.Helper()
	results, err := resolver.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got := searchResultPaths(results.Items); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
	if results.Incomplete {
		t.Fatal("Search() Incomplete = true")
	}
}

func searchResultPaths(results []SearchResult) []string {
	if len(results) == 0 {
		return nil
	}
	paths := make([]string, len(results))
	for index, result := range results {
		paths[index] = result.RelativePath
	}
	return paths
}

func matchingSearchSegments(segments []SearchSegment) []string {
	matches := make([]string, 0)
	for _, segment := range segments {
		if segment.Match {
			matches = append(matches, segment.Text)
		}
	}
	return matches
}

func searchExcerptText(segments []SearchSegment) string {
	var excerpt strings.Builder
	for _, segment := range segments {
		excerpt.WriteString(segment.Text)
	}
	return excerpt.String()
}
