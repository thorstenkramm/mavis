package config

import (
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// EditTemplate holds a validated repository edit link. Its zero value disables editing.
type EditTemplate struct{ prefix, suffix string }

func parseEditTemplate(raw string) (EditTemplate, error) {
	const token = "{path}"
	invalid := errors.New("edit_url_template must be an HTTPS URL with a fixed host, no credentials, and one {path} path segment")
	before, after, ok := strings.Cut(raw, token)
	if !ok || strings.ContainsAny(before+after, "{}\\") || strings.IndexFunc(raw, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return EditTemplate{}, invalid
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || !slices.Contains(strings.Split(u.Path, "/"), token) {
		return EditTemplate{}, invalid
	}
	// Check the literal path as well: a percent-encoded lookalike in the
	// path must not validate a real placeholder in the query or fragment.
	pathAndHost, _, _ := strings.Cut(raw, "?")
	pathAndHost, _, _ = strings.Cut(pathAndHost, "#")
	if !slices.Contains(strings.Split(pathAndHost, "/"), token) {
		return EditTemplate{}, invalid
	}
	if port := u.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return EditTemplate{}, invalid
		}
	}
	return EditTemplate{before, after}, nil
}

// URL returns the edit link for an authorized, decoded root-relative file path.
func (template EditTemplate) URL(relative string) string {
	if template.prefix == "" || relative == "" {
		return ""
	}
	parts := strings.Split(relative, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return template.prefix + strings.Join(parts, "/") + template.suffix
}
