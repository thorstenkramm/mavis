package render

import "bytes"

var utf8ByteOrderMark = []byte{0xEF, 0xBB, 0xBF}

// MarkdownBody returns source without a leading front-matter block for generated exports.
func MarkdownBody(source []byte) []byte {
	_, body, _ := splitFrontMatter(source)
	return body
}

// splitFrontMatter separates a leading YAML block without interpreting it.
// A UTF-8 byte order mark may precede the opening delimiter.
func splitFrontMatter(source []byte) (frontMatter string, body []byte, present bool) {
	text := bytes.TrimPrefix(source, utf8ByteOrderMark)
	firstLine, rest, ok := nextSourceLine(text)
	if !ok || !bytes.Equal(firstLine, []byte("---")) {
		return "", source, false
	}
	blockStart := len(text) - len(rest)
	for len(rest) > 0 {
		line, remaining, _ := nextSourceLine(rest)
		lineStart := len(text) - len(rest)
		if bytes.Equal(line, []byte("---")) || bytes.Equal(line, []byte("...")) {
			return string(text[blockStart:lineStart]), remaining, true
		}
		rest = remaining
	}
	return "", source, false
}

func nextSourceLine(source []byte) (line, remaining []byte, ok bool) {
	newline := bytes.IndexByte(source, '\n')
	if newline < 0 {
		return bytes.TrimSuffix(source, []byte{'\r'}), nil, len(source) > 0
	}
	line = source[:newline]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, source[newline+1:], true
}
