package content

import (
	"mime"
	"path"
	"strings"
	"unicode/utf8"
)

// AttachmentDisposition returns a safe attachment header for a logical filename.
// Non-ASCII names include both an ASCII fallback and an RFC 5987 UTF-8 filename.
func AttachmentDisposition(filename string) string {
	filename = path.Base(filename)
	if filename == "." || filename == "/" || filename == "" {
		filename = "download"
	}
	filename = strings.Map(func(character rune) rune {
		if character < 0x20 || character == 0x7f {
			return '_'
		}
		return character
	}, filename)

	fallback := strings.Map(func(character rune) rune {
		if character >= utf8.RuneSelf || character == '"' || character == '\\' {
			return '_'
		}
		return character
	}, filename)
	if strings.Trim(fallback, " ._") == "" {
		fallback = "download"
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": fallback})
	if filename != fallback && utf8.ValidString(filename) {
		disposition += "; filename*=utf-8''" + encodeFilename(filename)
	}
	return disposition
}

func encodeFilename(filename string) string {
	const hexadecimal = "0123456789ABCDEF"
	var encoded strings.Builder
	for _, value := range []byte(filename) {
		if (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') ||
			(value >= '0' && value <= '9') || strings.ContainsRune("!#$&+-.^_`|~", rune(value)) {
			encoded.WriteByte(value)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(hexadecimal[value>>4])
		encoded.WriteByte(hexadecimal[value&0x0f])
	}
	return encoded.String()
}
