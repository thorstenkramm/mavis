package config

import (
	"strings"
	"testing"
)

func TestExclusionsMatchRootRelativePaths(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{name: "star in root", pattern: "*.tmp", path: "notes.tmp", want: true},
		{name: "star does not cross slash", pattern: "*.tmp", path: "docs/notes.tmp", want: false},
		{name: "recursive match", pattern: "docs/**", path: "docs/guide/setup.md", want: true},
		{name: "recursive suffix", pattern: "**/*.tmp", path: "docs/guide/notes.tmp", want: true},
		{name: "recursive suffix at root", pattern: "**/*.tmp", path: "notes.tmp", want: true},
		{name: "question mark", pattern: "draft?/note.md", path: "draft1/note.md", want: true},
		{name: "character class", pattern: "draft[12]/note.md", path: "draft2/note.md", want: true},
		{name: "alternatives", pattern: "{drafts,tmp}/**", path: "tmp/guide/note.md", want: true},
		{name: "non-match", pattern: "drafts/**", path: "published/note.md", want: false},
		{name: "parent directory match", pattern: "private", path: "private/guide/secret.md", want: true},
		{name: "parent wildcard match", pattern: "docs/*", path: "docs/guide/setup.md", want: true},
		{name: "backslash path normalization", pattern: "docs/**", path: `docs\guide\setup.md`, want: true},
		{name: "leading current directory", pattern: "docs/**", path: "./docs/guide/setup.md", want: true},
		{name: "trailing directory slash", pattern: "docs/**", path: "docs/guide/", want: true},
		{name: "root is not excluded", pattern: "**", path: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exclusions, err := CompileExclusions([]string{test.pattern})
			if err != nil {
				t.Fatalf("CompileExclusions() error = %v", err)
			}
			if got := exclusions.Excluded(test.path); got != test.want {
				t.Fatalf("Excluded(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

func TestCompileExclusionsRejectsInvalidPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "empty", pattern: "", want: "must not be empty"},
		{name: "absolute slash", pattern: "/docs/**", want: "root-relative"},
		{name: "absolute windows", pattern: `C:\docs\**`, want: "root-relative"},
		{name: "parent escape", pattern: "../secrets/**", want: "escape"},
		{name: "nested parent escape", pattern: "docs/../../secrets/**", want: "escape"},
		{name: "backslash parent escape", pattern: `..\secrets\**`, want: "escape"},
		{name: "malformed class", pattern: "docs/[", want: "malformed"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := CompileExclusions([]string{test.pattern})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("CompileExclusions() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestLoadRejectsInvalidExclusion(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "exclude = [\"../secrets/**\"]\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "exclude[0]") {
		t.Fatalf("Load() error = %v, want exclusion diagnostic", err)
	}
}

func TestIsHidden(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: ".git/config", want: true},
		{path: "docs/.draft.md", want: true},
		{path: "docs/.private/guide.md", want: true},
		{path: `docs\.private\guide.md`, want: true},
		{path: "docs/guide.md", want: false},
		{path: "", want: false},
		{path: ".", want: false},
		{path: "..", want: false},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if got := IsHidden(test.path); got != test.want {
				t.Fatalf("IsHidden(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}
