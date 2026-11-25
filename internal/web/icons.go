package web

import (
	"path"
	"strings"

	"github.com/thorstenkramm/mavis/internal/render"
)

const (
	iconFolder   = "folder"
	iconFile     = "file"
	iconMarkdown = "markdown"
	iconYAML     = "yaml"
	iconPython   = "python"
)

// iconFor names the embedded icon for one file-browser or listing entry.
// Unknown types share the neutral file icon, so any content stays presentable.
func iconFor(name string, directory bool) string {
	if directory {
		return iconFolder
	}
	if render.IsMarkdown(name) {
		return iconMarkdown
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".yaml", ".yml":
		return iconYAML
	case ".py":
		return iconPython
	default:
		return iconFile
	}
}

// kindFor describes an entry by structure alone: folder, Markdown, or its
// upper-cased extension. It never interprets file contents.
func kindFor(name string, directory bool) string {
	if directory {
		return "Folder"
	}
	if render.IsMarkdown(name) {
		return "Markdown"
	}
	extension := path.Ext(name)
	// A dotfile such as ".env" is a name, not an extension.
	if len(extension) < 2 || extension == name {
		return "File"
	}
	return strings.ToUpper(extension[1:])
}
