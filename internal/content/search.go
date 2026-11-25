package content

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxSearchFileSize = 10 << 20
	searchSampleSize  = 512
	maxSearchResults  = 100
	maxSearchExcerpt  = 240
	searchWordsBefore = 8
	searchWordsAfter  = 12
)

// SearchSegment is a run of exact source text in a search excerpt.
type SearchSegment struct {
	Text  string
	Match bool
}

// SearchResult describes one matching file and its logical content path.
type SearchResult struct {
	Name         string
	RelativePath string
	Excerpt      []SearchSegment
	targetPath   string // Deduplicate results reached through different directory aliases.
}

// SearchResults contains path-ordered matches and bounded-scan notices.
type SearchResults struct {
	Items      []SearchResult
	Truncated  bool
	Incomplete bool
}

type searchQuery struct {
	lower  []rune
	prefix []int
}

type searchScanner struct {
	resolver   *Resolver
	query      searchQuery
	visited    map[string]struct{}
	active     map[string]struct{}
	include    func(Resource) bool
	collector  searchCollector
	incomplete bool
}

type searchCollector struct {
	items []SearchResult
}

// Search scans current eligible file contents for one literal,
// case-insensitive substring. Files and descendant directories that cannot be
// searched are omitted and reported through SearchResults.Incomplete.
func (resolver *Resolver) Search(ctx context.Context, query string) (SearchResults, error) {
	return resolver.SearchWhere(ctx, query, nil)
}

// SearchWhere additionally skips entries the caller cannot present. Rejecting
// a directory skips its subtree under that logical path only.
func (resolver *Resolver) SearchWhere(ctx context.Context, query string, include func(Resource) bool) (SearchResults, error) {
	results := SearchResults{Items: make([]SearchResult, 0)}
	if err := ctx.Err(); err != nil {
		return results, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return results, nil
	}

	queryRunes := lowerSearchRunes([]rune(query))
	scanner := searchScanner{
		resolver: resolver,
		include:  include,
		query: searchQuery{
			lower:  queryRunes,
			prefix: searchPrefixTable(queryRunes),
		},
		visited: make(map[string]struct{}),
		active:  map[string]struct{}{".": {}},
		collector: searchCollector{
			items: make([]SearchResult, 0, maxSearchResults+1),
		},
	}
	listing, err := resolver.list(ctx, "")
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return results, ctxErr
		}
		return results, fmt.Errorf("search content root: %w", err)
	}
	scanner.incomplete = listing.incomplete
	if _, err := scanner.walk(ctx, listing.entries); err != nil {
		return results, err
	}
	if err := ctx.Err(); err != nil {
		return results, err
	}
	results.Items = scanner.collector.items
	if len(results.Items) > maxSearchResults {
		results.Items = results.Items[:maxSearchResults]
		results.Truncated = true
	}
	results.Incomplete = scanner.incomplete
	return results, nil
}

// covered is false if an omission could be recovered under another logical path.
func (scanner *searchScanner) walk(ctx context.Context, entries []Resource) (covered bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].RelativePath < entries[right].RelativePath
	})
	covered = true
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if scanner.include != nil && !scanner.include(entry) {
			covered = false
			continue
		}
		if entry.Info.IsDir() {
			if _, cycle := scanner.active[entry.TargetPath]; cycle {
				covered = false
				continue
			}
			if _, seen := scanner.visited[entry.TargetPath]; seen {
				continue
			}
			children, err := scanner.resolver.list(ctx, entry.RelativePath)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return false, ctxErr
				}
				scanner.incomplete = true
				covered = false
				continue
			}
			if children.incomplete {
				scanner.incomplete = true
			}
			scanner.active[entry.TargetPath] = struct{}{}
			subtreeCovered, err := scanner.walk(ctx, children.entries)
			delete(scanner.active, entry.TargetPath)
			if err != nil {
				return false, err
			}
			if subtreeCovered && !children.logicalOmissions && !children.incomplete {
				scanner.visited[entry.TargetPath] = struct{}{}
			} else {
				// An unvisited path may still be eligible through a different alias.
				covered = false
			}
			continue
		}

		result, incomplete, err := readSearchResult(ctx, entry, scanner.query)
		if err != nil {
			return false, err
		}
		if incomplete {
			scanner.incomplete = true
			covered = false
		}
		if result != nil {
			scanner.collector.add(*result)
		}
	}
	return covered, nil
}

func readSearchResult(ctx context.Context, resource Resource, query searchQuery) (result *SearchResult, incomplete bool, returnErr error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	file, err := os.Open(resource.Path)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		return nil, true, nil
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			result = nil
			incomplete = true
		}
	}()

	sample := make([]byte, searchSampleSize)
	read, err := io.ReadFull(file, sample)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, false, ctxErr
	}
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, true, nil
	}
	sample = sample[:read]
	if !isSearchTextCandidate(resource.RelativePath, sample) {
		return nil, false, nil
	}

	source, err := io.ReadAll(io.LimitReader(
		io.MultiReader(bytes.NewReader(sample), file),
		maxSearchFileSize+1,
	))
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, false, ctxErr
	}
	if err != nil || len(source) > maxSearchFileSize || !utf8.Valid(source) {
		return nil, true, nil
	}

	runes := []rune(string(source))
	lower := lowerSearchRunes(runes)
	first := findSearchRunes(lower, query.lower, query.prefix, 0)
	if first < 0 {
		return nil, false, nil
	}
	return &SearchResult{
		Name:         path.Base(resource.RelativePath),
		RelativePath: resource.RelativePath,
		targetPath:   resource.TargetPath,
		Excerpt:      buildSearchExcerpt(runes, lower, first, query),
	}, false, nil
}

func lowerSearchRunes(runes []rune) []rune {
	lower := make([]rune, len(runes))
	for index, current := range runes {
		lower[index] = unicode.ToLower(current)
	}
	return lower
}

func searchPrefixTable(query []rune) []int {
	prefix := make([]int, len(query))
	for index, matched := 1, 0; index < len(query); {
		if query[index] == query[matched] {
			matched++
			prefix[index] = matched
			index++
		} else if matched > 0 {
			matched = prefix[matched-1]
		} else {
			index++
		}
	}
	return prefix
}

func findSearchRunes(source, query []rune, prefix []int, start int) int {
	if len(query) == 0 || start > len(source)-len(query) {
		return -1
	}
	for index, matched := start, 0; index < len(source); {
		if source[index] == query[matched] {
			index++
			matched++
			if matched == len(query) {
				return index - matched
			}
		} else if matched > 0 {
			matched = prefix[matched-1]
		} else {
			index++
		}
	}
	return -1
}

func buildSearchExcerpt(source, lower []rune, first int, query searchQuery) []SearchSegment {
	start, end := searchExcerptBounds(source, first, first+len(query.lower))
	prefix := ""
	if start > 0 {
		prefix = "… "
	}
	suffix := ""
	if end < len(source) {
		suffix = " …"
	}

	if utf8.RuneCountInString(prefix)+len(source[start:end])+utf8.RuneCountInString(suffix) > maxSearchExcerpt {
		suffix = "…"
		available := maxSearchExcerpt - utf8.RuneCountInString(prefix) - utf8.RuneCountInString(suffix)
		if first < start || first+len(query.lower) > start+available {
			// Keep the complete first match visible when a long preceding token
			// would otherwise consume the entire capped excerpt.
			start = first
			prefix = "… "
			available = maxSearchExcerpt - utf8.RuneCountInString(prefix) - utf8.RuneCountInString(suffix)
		}
		end = min(end, start+available)
		if end == len(source) {
			suffix = ""
		}
	}
	segments := make([]SearchSegment, 0, 5)
	appendSearchSegment(&segments, SearchSegment{Text: prefix})
	highlightSearchMatches(&segments, source[start:end], lower[start:end], query)
	appendSearchSegment(&segments, SearchSegment{Text: suffix})
	return segments
}

func searchExcerptBounds(source []rune, matchStart, matchEnd int) (int, int) {
	before := contextRunes(source, matchStart-1, -1, searchWordsBefore)
	after := contextRunes(source, matchEnd, 1, searchWordsAfter)
	return matchStart - before, matchEnd + after
}

// Start inside the matched word: its remaining runes don't count as another
// word. One rune beyond the display cap is enough to signal truncation.
func contextRunes(source []rune, i, step, wordLimit int) int {
	count, words, lastNonSpace := 0, 0, 0
	inWord := true
	for ; i >= 0 && i < len(source) && count <= maxSearchExcerpt; i += step {
		if unicode.IsSpace(source[i]) {
			inWord = false
		} else {
			if !inWord {
				if words == wordLimit {
					return lastNonSpace
				}
				words++
			}
			inWord = true
			lastNonSpace = count + 1
		}
		count++
	}
	return count
}

func highlightSearchMatches(segments *[]SearchSegment, source, lower []rune, query searchQuery) {
	for offset := 0; offset < len(source); {
		matched := findSearchRunes(lower, query.lower, query.prefix, offset)
		if matched < 0 {
			appendSearchSegment(segments, SearchSegment{Text: string(source[offset:])})
			return
		}
		if matched > offset {
			appendSearchSegment(segments, SearchSegment{Text: string(source[offset:matched])})
		}
		end := matched + len(query.lower)
		appendSearchSegment(segments, SearchSegment{Text: string(source[matched:end]), Match: true})
		offset = end
	}
}

func appendSearchSegment(segments *[]SearchSegment, segment SearchSegment) {
	if segment.Text == "" {
		return
	}
	if len(*segments) > 0 && (*segments)[len(*segments)-1].Match == segment.Match {
		(*segments)[len(*segments)-1].Text += segment.Text
		return
	}
	*segments = append(*segments, segment)
}

func (collector *searchCollector) add(result SearchResult) {
	for i, existing := range collector.items {
		if existing.targetPath == result.targetPath {
			if existing.RelativePath <= result.RelativePath {
				return
			}
			collector.items = append(collector.items[:i], collector.items[i+1:]...)
			break
		}
	}
	if len(collector.items) == maxSearchResults+1 &&
		result.RelativePath >= collector.items[len(collector.items)-1].RelativePath {
		return
	}
	index := sort.Search(len(collector.items), func(index int) bool {
		return collector.items[index].RelativePath >= result.RelativePath
	})
	if index < len(collector.items) && collector.items[index].RelativePath == result.RelativePath {
		return
	}
	if len(collector.items) < maxSearchResults+1 {
		collector.items = append(collector.items, SearchResult{})
		copy(collector.items[index+1:], collector.items[index:])
	} else {
		copy(collector.items[index+1:], collector.items[index:len(collector.items)-1])
	}
	collector.items[index] = result
}

func isSearchTextCandidate(name string, sample []byte) bool {
	extension := strings.ToLower(path.Ext(name))
	if obviousSearchBinaryExtension(extension) {
		return false
	}
	contentType, _, _ := strings.Cut(http.DetectContentType(sample), ";")
	if obviousSearchBinaryContentType(contentType) {
		return false
	}
	if recognizedSearchTextExtension(extension) || strings.HasPrefix(contentType, "text/") {
		return true
	}
	switch contentType {
	case "application/javascript", "application/json", "application/xml", "application/x-sh":
		return true
	default:
		return false
	}
}

func recognizedSearchTextExtension(extension string) bool {
	switch extension {
	case ".bash", ".c", ".cc", ".cfg", ".cjs", ".conf", ".cpp", ".cs", ".css",
		".csv", ".fish", ".go", ".h", ".hpp", ".htm", ".html", ".ini", ".java",
		".js", ".json", ".jsonc", ".jsx", ".log", ".markdown", ".md", ".mdown",
		".mjs", ".php", ".ps1", ".py", ".rb", ".rs", ".sh", ".sql", ".svg",
		".tex", ".text", ".toml", ".ts", ".tsv", ".tsx", ".txt", ".xml", ".yaml",
		".yml", ".zsh":
		return true
	default:
		return false
	}
}

func obviousSearchBinaryExtension(extension string) bool {
	switch extension {
	case ".7z", ".avif", ".avi", ".bmp", ".bz2", ".class", ".dll", ".dylib",
		".eot", ".exe", ".gif", ".gz", ".ico", ".jar", ".jpeg", ".jpg", ".mov",
		".mp3", ".mp4", ".otf", ".pdf", ".png", ".rar", ".so", ".tar", ".tif",
		".tiff", ".ttf", ".wasm", ".webm", ".webp", ".woff", ".woff2", ".xz",
		".zip":
		return true
	default:
		return false
	}
}

func obviousSearchBinaryContentType(contentType string) bool {
	if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "audio/") ||
		strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "font/") {
		return true
	}
	switch contentType {
	case "application/pdf", "application/x-gzip", "application/x-rar-compressed",
		"application/x-7z-compressed", "application/zip":
		return true
	default:
		return false
	}
}
