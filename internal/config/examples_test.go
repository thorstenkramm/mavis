package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var fencedTOML = regexp.MustCompile("(?ms)^```toml\\n(.*?)^```")

func TestDocumentationTOMLExamples(t *testing.T) {
	paths := []string{
		"../../README.md",
		"../../docs/content/reference/configuration.md",
		"../../docs/internal/PRD.md",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		matches := fencedTOML.FindAllSubmatch(data, -1)
		if len(matches) == 0 {
			t.Fatalf("%s contains no fenced TOML examples", path)
		}
		for index, match := range matches {
			t.Run(fmt.Sprintf("%s/%d", path, index), func(t *testing.T) {
				validateExample(t, string(match[1]))
			})
		}
	}
}

func TestHomepageTOMLExample(t *testing.T) {
	data, err := os.ReadFile("../../docs/content/_index.md")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	for index, line := range lines {
		if line != "example: |-" {
			continue
		}
		var example []string
		for _, following := range lines[index+1:] {
			if following != "" && !strings.HasPrefix(following, "  ") {
				break
			}
			example = append(example, strings.TrimPrefix(following, "  "))
		}
		validateExample(t, strings.Join(example, "\n"))
		return
	}
	t.Fatal("homepage example TOML block not found")
}

func validateExample(t *testing.T, example string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(example), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("documented example is invalid: %v\n%s", err, example)
	}
}
