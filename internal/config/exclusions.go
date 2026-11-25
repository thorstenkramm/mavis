package config

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Exclusions is an immutable set of validated root-relative patterns.
type Exclusions struct {
	patterns []string
}

// CompileExclusions validates and compiles slash-based exclusion patterns.
func CompileExclusions(patterns []string) (Exclusions, error) {
	compiled := Exclusions{patterns: make([]string, len(patterns))}
	for index, pattern := range patterns {
		normalized := strings.ReplaceAll(pattern, "\\", "/")
		if normalized == "" {
			return Exclusions{}, fmt.Errorf("exclude[%d] must not be empty", index)
		}
		if strings.HasPrefix(normalized, "/") || filepath.IsAbs(pattern) || hasVolumePrefix(normalized) {
			return Exclusions{}, fmt.Errorf("exclude[%d] %q must be root-relative", index, pattern)
		}
		for _, segment := range strings.Split(normalized, "/") {
			if segment == ".." {
				return Exclusions{}, fmt.Errorf("exclude[%d] %q must not escape the served root", index, pattern)
			}
		}
		normalized = strings.TrimPrefix(normalized, "./")
		if !doublestar.ValidatePattern(normalized) {
			return Exclusions{}, fmt.Errorf("exclude[%d] %q is malformed", index, pattern)
		}
		compiled.patterns[index] = normalized
	}
	return compiled, nil
}

// Excluded reports whether a path or any of its parent directories matches.
func (exclusions Exclusions) Excluded(rootRelative string) bool {
	normalized := normalizeSlashes(rootRelative)
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.Trim(normalized, "/")
	if normalized == "" || normalized == "." {
		return false
	}

	for candidate := normalized; candidate != "." && candidate != ""; candidate = path.Dir(candidate) {
		for _, pattern := range exclusions.patterns {
			if doublestar.MatchUnvalidated(pattern, candidate) {
				return true
			}
		}
	}
	return false
}

// IsHidden reports whether any path component begins with a dot.
func IsHidden(rootRelative string) bool {
	for _, component := range strings.Split(normalizeSlashes(rootRelative), "/") {
		if component != "." && component != ".." && strings.HasPrefix(component, ".") {
			return true
		}
	}
	return false
}

func normalizeSlashes(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}

func hasVolumePrefix(pattern string) bool {
	return len(pattern) >= 2 && pattern[1] == ':' && ((pattern[0] >= 'A' && pattern[0] <= 'Z') || (pattern[0] >= 'a' && pattern[0] <= 'z'))
}
