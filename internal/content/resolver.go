package content

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thorstenkramm/mavis/internal/config"
)

// ErrNotFound hides missing, invalid, and outside-root paths uniformly.
var ErrNotFound = errors.New("content not found")

// Resolver resolves request paths within one served root.
type Resolver struct {
	root          string
	includeHidden bool
	exclusions    config.Exclusions
}

// Resource is a resolved eligible filesystem entry.
type Resource struct {
	Path         string
	RelativePath string
	TargetPath   string
	Info         os.FileInfo
}

// NewResolver validates a content root and its visibility configuration.
func NewResolver(root string, configuration config.Config) (*Resolver, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve content root: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve content root: %w", err)
	}
	info, err := os.Stat(resolvedRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect content root: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("content root is not a directory")
	}
	exclusions, err := config.CompileExclusions(configuration.Exclude)
	if err != nil {
		return nil, err
	}
	return &Resolver{
		root:          resolvedRoot,
		includeHidden: configuration.IncludeHidden,
		exclusions:    exclusions,
	}, nil
}

// Root returns the resolved absolute content root.
func (resolver *Resolver) Root() string {
	return resolver.root
}

// Resolve validates a URL-style path and resolves its current filesystem target.
func (resolver *Resolver) Resolve(requestPath string) (*Resource, error) {
	decoded, err := url.PathUnescape(requestPath)
	if err != nil {
		return nil, ErrNotFound
	}
	return resolver.ResolveLogical(decoded)
}

// ResolveLogical accepts a decoded root-relative name, never a URL escape sequence.
func (resolver *Resolver) ResolveLogical(decoded string) (*Resource, error) {
	logicalPath, err := cleanLogicalPath(decoded)
	if err != nil {
		return nil, ErrNotFound
	}
	return resolver.resolveEntry(logicalPath, false)
}

// hasExactSpelling rejects filesystem aliases (including case variants) that
// are not the actual entry name. This keeps the same policy on all volumes.
func (resolver *Resolver) hasExactSpelling(relative string) bool {
	dir := resolver.root
	if relative == "" || relative == "." {
		return true
	}
	for _, component := range strings.Split(relative, "/") {
		entries, err := os.ReadDir(dir)
		if err != nil || !slices.ContainsFunc(entries, func(e fs.DirEntry) bool { return e.Name() == component }) {
			return false
		}
		dir = filepath.Join(dir, component)
	}
	return true
}

// fromListing means ReadDir supplied the name and its parent was already verified.
func (resolver *Resolver) resolveEntry(logicalPath string, fromListing bool) (*Resource, error) {
	if !resolver.visible(logicalPath) {
		return nil, ErrNotFound
	}
	candidate := filepath.Join(resolver.root, filepath.FromSlash(logicalPath))
	resolvedPath, err := filepath.EvalSymlinks(candidate)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resolve content: %w", err)
	}
	targetPath, err := filepath.Rel(resolver.root, resolvedPath)
	if err != nil || escapesRoot(targetPath) {
		return nil, ErrNotFound
	}
	target := filepath.ToSlash(targetPath)
	if !resolver.visible(target) || (!fromListing && !resolver.hasExactSpelling(logicalPath)) ||
		(target != logicalPath && !resolver.hasExactSpelling(target)) {
		return nil, ErrNotFound
	}
	info, err := os.Stat(resolvedPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("inspect content: %w", err)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return nil, ErrNotFound
	}
	if info.Mode().IsRegular() {
		private, err := os.Stat(filepath.Join(resolver.root, config.FileName))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("inspect private configuration: %w", err)
		}
		if private != nil && os.SameFile(private, info) {
			return nil, ErrNotFound
		}
	}
	return &Resource{
		Path:         resolvedPath,
		RelativePath: logicalPath,
		TargetPath:   filepath.ToSlash(targetPath),
		Info:         info,
	}, nil
}

// List returns eligible entries in one directory without walking descendants.
func (resolver *Resolver) List(requestPath string) ([]Resource, error) {
	directory, err := resolver.ResolveLogical(requestPath)
	if err != nil {
		return nil, err
	}
	if !directory.Info.IsDir() {
		return nil, ErrNotFound
	}
	entries, err := os.ReadDir(directory.Path)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	resources := make([]Resource, 0, len(entries))
	for _, entry := range entries {
		logicalPath := entry.Name()
		if directory.RelativePath != "" {
			logicalPath = path.Join(directory.RelativePath, entry.Name())
		}
		if _, err := cleanLogicalPath(logicalPath); err != nil {
			continue
		}
		resource, err := resolver.resolveEntry(logicalPath, true)
		if err != nil {
			continue
		}
		resources = append(resources, *resource)
	}
	return resources, nil
}

func (resolver *Resolver) visible(relativePath string) bool {
	if relativePath == "" || relativePath == "." {
		return true
	}
	if relativePath == config.FileName {
		return false
	}
	if !resolver.includeHidden && config.IsHidden(relativePath) {
		return false
	}
	return !resolver.exclusions.Excluded(relativePath)
}

func cleanLogicalPath(decoded string) (string, error) {
	if strings.ContainsRune(decoded, '\x00') || strings.Contains(decoded, "\\") {
		return "", errors.New("invalid path encoding")
	}
	if decoded == "" || decoded == "/" {
		return "", nil
	}
	if strings.HasPrefix(decoded, "//") {
		return "", errors.New("absolute path")
	}
	decoded = strings.TrimPrefix(decoded, "/")
	decoded = strings.TrimSuffix(decoded, "/")
	if decoded == "" {
		return "", nil
	}
	if hasVolumePrefix(decoded) {
		return "", errors.New("volume path")
	}
	segments := strings.Split(decoded, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", errors.New("non-canonical path")
		}
	}
	return strings.Join(segments, "/"), nil
}

func escapesRoot(relative string) bool {
	return filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func hasVolumePrefix(path string) bool {
	return len(path) >= 2 && path[1] == ':' && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z'))
}
