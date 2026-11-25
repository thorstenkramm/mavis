package content

import (
	"context"
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
	readDir       func(string) ([]os.DirEntry, error)
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
		readDir:       os.ReadDir,
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
func (resolver *Resolver) hasExactSpelling(relative string) (bool, error) {
	dir := resolver.root
	if relative == "" || relative == "." {
		return true, nil
	}
	for _, component := range strings.Split(relative, "/") {
		entries, err := resolver.readDir(dir)
		if err != nil {
			return false, err
		}
		if !slices.ContainsFunc(entries, func(e fs.DirEntry) bool { return e.Name() == component }) {
			return false, nil
		}
		dir = filepath.Join(dir, component)
	}
	return true, nil
}

func (resolver *Resolver) requireExactSpelling(relative, resolvedPath string) error {
	exact, err := resolver.hasExactSpelling(relative)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	if err != nil {
		return fmt.Errorf("verify exact content spelling: %w", err)
	}
	if !exact {
		// EvalSymlinks already found this target. If it no longer exists, the
		// omission is an operational race rather than a spelling-policy denial.
		if _, statErr := os.Stat(resolvedPath); errors.Is(statErr, fs.ErrNotExist) {
			return fmt.Errorf("%w: %w", ErrNotFound, statErr)
		} else if statErr != nil {
			return fmt.Errorf("inspect content after spelling verification: %w", statErr)
		}
		return ErrNotFound
	}
	return nil
}

// fromListing means ReadDir supplied the name and its parent was already verified.
func (resolver *Resolver) resolveEntry(logicalPath string, fromListing bool) (*Resource, error) {
	if !resolver.visible(logicalPath) {
		return nil, ErrNotFound
	}
	candidate := filepath.Join(resolver.root, filepath.FromSlash(logicalPath))
	resolvedPath, err := filepath.EvalSymlinks(candidate)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("resolve content: %w", err)
	}
	targetPath, err := filepath.Rel(resolver.root, resolvedPath)
	if err != nil || escapesRoot(targetPath) {
		return nil, ErrNotFound
	}
	target := filepath.ToSlash(targetPath)
	if !resolver.visible(target) {
		return nil, ErrNotFound
	}
	if !fromListing {
		if err := resolver.requireExactSpelling(logicalPath, resolvedPath); err != nil {
			return nil, err
		}
	}
	if target != logicalPath {
		if err := resolver.requireExactSpelling(target, resolvedPath); err != nil {
			return nil, err
		}
	}
	info, err := os.Stat(resolvedPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
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
	listing, err := resolver.list(context.Background(), requestPath)
	return listing.entries, err
}

type directoryListing struct {
	entries          []Resource
	incomplete       bool
	logicalOmissions bool // A different logical alias may admit these entries.
}

// list additionally reports eligible entries that failed after enumeration.
// Policy exclusions and outside-root links remain silent omissions.
func (resolver *Resolver) list(ctx context.Context, requestPath string) (directoryListing, error) {
	var listing directoryListing
	if err := ctx.Err(); err != nil {
		return listing, err
	}
	directory, err := resolver.ResolveLogical(requestPath)
	if err != nil {
		return listing, err
	}
	if !directory.Info.IsDir() {
		return listing, ErrNotFound
	}
	entries, err := resolver.readDir(directory.Path)
	if err != nil {
		return listing, fmt.Errorf("read directory: %w", err)
	}
	listing.entries = make([]Resource, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return directoryListing{}, err
		}
		logicalPath := entry.Name()
		if directory.RelativePath != "" {
			logicalPath = path.Join(directory.RelativePath, entry.Name())
		}
		if _, err := cleanLogicalPath(logicalPath); err != nil {
			continue
		}
		resource, err := resolver.resolveEntry(logicalPath, true)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || !errors.Is(err, ErrNotFound) {
				listing.incomplete = true
			} else if resolver.aliasMayAdmit(directory.TargetPath, logicalPath, entry) {
				listing.logicalOmissions = true
			}
			continue
		}
		listing.entries = append(listing.entries, *resource)
	}
	return listing, ctx.Err()
}

// An alias changes the directory prefix, not the entry name. A hidden
// basename or an excluded physical target is therefore still ineligible.
func (resolver *Resolver) aliasMayAdmit(parentTarget, logical string, entry fs.DirEntry) bool {
	if logical == config.FileName || (!resolver.includeHidden && config.IsHidden(entry.Name())) || !resolver.exclusions.Excluded(logical) {
		return false
	}
	// For a symlink, the physical target isn't necessarily parentTarget/name.
	return entry.Type()&fs.ModeSymlink != 0 || !resolver.exclusions.Excluded(path.Join(parentTarget, entry.Name()))
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
