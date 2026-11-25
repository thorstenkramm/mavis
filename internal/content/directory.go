package content

import (
	"net/url"
	"path"
	"sort"
	"strings"
)

// DirectoryPage contains a fresh eligible directory listing and optional README.
type DirectoryPage struct {
	Resource Resource
	Entries  []Resource
	README   *Resource
}

// NavigationItem is one file-browser entry with current-ancestry children.
type NavigationItem struct {
	Name         string
	RelativePath string
	URL          string
	Directory    bool
	Current      bool
	Children     []NavigationItem
}

// Directory returns a per-request directory model.
func (resolver *Resolver) Directory(requestPath string) (*DirectoryPage, error) {
	resource, err := resolver.ResolveLogical(requestPath)
	if err != nil {
		return nil, err
	}
	if !resource.Info.IsDir() {
		return nil, ErrNotFound
	}
	entries, err := resolver.List(resource.RelativePath)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(left, right int) bool {
		if entries[left].Info.IsDir() != entries[right].Info.IsDir() {
			return entries[left].Info.IsDir()
		}
		return strings.ToLower(resourceName(entries[left])) < strings.ToLower(resourceName(entries[right]))
	})

	page := &DirectoryPage{Resource: *resource, Entries: entries}
	for index := range entries {
		if !entries[index].Info.IsDir() && resourceName(entries[index]) == "README.md" {
			page.README = &entries[index]
			break
		}
	}
	return page, nil
}

// Navigation builds only the current directory and its ancestry.
func (resolver *Resolver) Navigation(currentPath string) ([]NavigationItem, error) {
	current, err := resolver.ResolveLogical(currentPath)
	if err != nil {
		return nil, err
	}
	currentDirectory := current.RelativePath
	if !current.Info.IsDir() {
		currentDirectory = path.Dir(currentDirectory)
		if currentDirectory == "." {
			currentDirectory = ""
		}
	}
	ancestors := []string{""}
	if currentDirectory != "" {
		for _, component := range strings.Split(currentDirectory, "/") {
			ancestors = append(ancestors, path.Join(ancestors[len(ancestors)-1], component))
		}
	}
	return resolver.navigationLevel(0, ancestors, current.RelativePath)
}

func (resolver *Resolver) navigationLevel(level int, ancestors []string, currentPath string) ([]NavigationItem, error) {
	entries, err := resolver.List(ancestors[level])
	if err != nil {
		return nil, err
	}
	items := make([]NavigationItem, 0, len(entries))
	for _, entry := range entries {
		item := NavigationItem{
			Name:         resourceName(entry),
			RelativePath: entry.RelativePath,
			URL:          URLFor(entry.RelativePath, entry.Info.IsDir()),
			Directory:    entry.Info.IsDir(),
			Current:      entry.RelativePath == currentPath,
		}
		if level+1 < len(ancestors) && entry.RelativePath == ancestors[level+1] {
			item.Children, err = resolver.navigationLevel(level+1, ancestors, currentPath)
			if err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// URLFor returns an escaped Mavis content URL.
func URLFor(relativePath string, directory bool) string {
	if relativePath == "" || relativePath == "." {
		return "/"
	}
	segments := strings.Split(relativePath, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	result := "/" + strings.Join(segments, "/")
	if directory {
		result += "/"
	}
	return result
}

// RawURLFor returns a file URL that requests original bytes instead of a viewer.
func RawURLFor(relativePath string, rawQuery string) string {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		query = url.Values{}
	}
	query.Set("raw", "1")
	return URLFor(relativePath, false) + "?" + query.Encode()
}

func resourceName(resource Resource) string {
	if resource.RelativePath == "" {
		return ""
	}
	return path.Base(resource.RelativePath)
}
