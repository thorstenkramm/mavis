package config

import (
	"errors"
	"net/netip"
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
	// Browsers interpret a hostname ending in a numeric label as IPv4.
	// Require canonical dotted decimal rather than accepting ambiguous
	// abbreviated, octal or hexadecimal addresses.
	host := strings.TrimSuffix(u.Hostname(), ".")
	if !strings.HasPrefix(u.Host, "[") && endsInNumber(host) {
		addr, err := netip.ParseAddr(host)
		if err != nil || !addr.Is4() {
			return EditTemplate{}, invalid
		}
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

func endsInNumber(host string) bool {
	label := strings.ToLower(host[strings.LastIndexByte(host, '.')+1:])
	if digits, hex := strings.CutPrefix(label, "0x"); hex {
		return strings.Trim(digits, "0123456789abcdef") == ""
	}
	return label != "" && strings.Trim(label, "0123456789") == ""
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
