package render

import (
	"net/url"
	"path"
	"strings"

	"github.com/yuin/goldmark/ast"
)

func rewriteLinks(document ast.Node, documentPath string, authorize AccessFunc) error {
	return ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination *[]byte
		switch link := node.(type) {
		case *ast.Link:
			destination = &link.Destination
		case *ast.Image:
			destination = &link.Destination
		default:
			return ast.WalkContinue, nil
		}
		rewritten, ok := rewriteDestination(string(*destination), documentPath, authorize)
		if !ok {
			*destination = nil
			return ast.WalkContinue, nil
		}
		*destination = []byte(rewritten)
		return ast.WalkContinue, nil
	})
}

func rewriteDestination(destination, documentPath string, authorize AccessFunc) (string, bool) {
	parsed, err := url.Parse(destination)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(destination, "//") || parsed.Path == "" {
		return destination, err == nil
	}
	if strings.Contains(parsed.Path, "\\") {
		return "", false
	}

	var target string
	if strings.HasPrefix(parsed.Path, "/") {
		if hasParentSegment(parsed.Path) {
			return "", false
		}
		target = strings.TrimPrefix(parsed.Path, "/")
	} else {
		base := path.Dir(documentPath)
		if base == "." {
			base = ""
		}
		target = path.Join(base, parsed.Path)
	}
	if target == ".." || strings.HasPrefix(target, "../") || strings.HasPrefix(target, "/") {
		return "", false
	}
	if target == "." {
		target = ""
	}
	if authorize != nil && !authorize(target) {
		return "", false
	}

	result := contentURL(target)
	if strings.HasSuffix(parsed.Path, "/") && result != "/" {
		result += "/"
	}
	if parsed.ForceQuery || parsed.RawQuery != "" {
		result += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		result += "#" + parsed.EscapedFragment()
	}
	return result, true
}

func hasParentSegment(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func contentURL(rootRelativePath string) string {
	if rootRelativePath == "" {
		return "/"
	}
	segments := strings.Split(rootRelativePath, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	return "/" + strings.Join(segments, "/")
}
