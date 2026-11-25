package web

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// viewerSampleSize matches the prefix inspected by http.DetectContentType.
const viewerSampleSize = 512

type viewerKind string

const (
	viewerNone  viewerKind = ""
	viewerText  viewerKind = "text"
	viewerImage viewerKind = "image"
	viewerPDF   viewerKind = "pdf"
)

type viewerModel struct {
	Kind   viewerKind
	Name   string
	Text   string
	RawURL string
}

// readViewerSample returns the leading bytes used to select a viewer.
func readViewerSample(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	sample := make([]byte, viewerSampleSize)
	read, readErr := io.ReadFull(file, sample)
	if err := file.Close(); err != nil {
		return nil, err
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return nil, readErr
	}
	return sample[:read], nil
}

// classifyViewer selects a viewer from a filename and leading bytes. A text
// result is provisional until the complete source is known to be valid UTF-8.
func classifyViewer(name string, sample []byte) viewerKind {
	if len(sample) > viewerSampleSize {
		sample = sample[:viewerSampleSize]
	}
	detected, _, err := mime.ParseMediaType(http.DetectContentType(sample))
	if err != nil {
		detected = http.DetectContentType(sample)
	}
	if detected == "application/pdf" {
		return viewerPDF
	}
	if browserImageType(detected) {
		return viewerImage
	}

	extension := strings.ToLower(filepath.Ext(name))
	if extension == ".avif" && detected == "application/octet-stream" && bytes.Contains(sample, []byte("ftypavif")) {
		return viewerImage
	}
	if extension == ".svg" && bytes.Contains(bytes.ToLower(sample), []byte("<svg")) {
		return viewerImage
	}
	if strings.HasPrefix(detected, "text/") || textFileExtension(extension) {
		return viewerText
	}
	return viewerNone
}

func browserImageType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif", "image/bmp", "image/x-icon":
		return true
	default:
		return false
	}
}

func textFileExtension(extension string) bool {
	switch extension {
	case ".css", ".go", ".html", ".js", ".json", ".md", ".toml", ".txt", ".xml", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
