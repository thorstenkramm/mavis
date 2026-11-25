// Package render converts Markdown into safe navigable HTML.
package render

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// NewMarkdown constructs the baseline Markdown parser.
func NewMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(
			util.Prioritized(newTaskCheckBoxRenderer(), 499),
		)),
	)
}

// IsMarkdown reports whether a filename has a supported Markdown extension.
func IsMarkdown(name string) bool {
	extension := strings.ToLower(filepath.Ext(name))
	return extension == ".md" || extension == ".markdown"
}

// NewSanitizer constructs the HTML sanitizer for goldmark-generated markup.
func NewSanitizer() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^language-[a-zA-Z0-9_-]+$`)).OnElements("code")
	policy.AllowAttrs("type").Matching(regexp.MustCompile(`(?i)^checkbox$`)).OnElements("input")
	policy.AllowAttrs("checked").Matching(regexp.MustCompile(`(?i)^(|checked)$`)).OnElements("input")
	policy.AllowAttrs("disabled").Matching(regexp.MustCompile(`(?i)^(|disabled)$`)).OnElements("input")
	policy.AllowAttrs("aria-label").Matching(regexp.MustCompile(`^.{1,500}$`)).OnElements("input")
	return policy
}
