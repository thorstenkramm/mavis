package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

const (
	maxOutputSize = 50 << 20 // 50 MiB
	maxLogSize    = 64 << 10 // 64 KiB per process stream
)

// Renderer is the Markdown rendering contract used by Converter.
type Renderer interface {
	Render(source []byte, documentPath string, authorize render.AccessFunc) (render.Document, error)
}

// Converter invokes one pre-resolved pandoc executable for each conversion.
type Converter struct {
	executable string
	resolver   *content.Resolver
	renderer   Renderer
	logger     *log.Logger
}

// New constructs a DOCX converter. The executable must already be resolved by
// startup code; Convert never searches PATH.
func New(executable string, resolver *content.Resolver, renderer Renderer, logger *log.Logger) *Converter {
	if renderer == nil {
		renderer = render.NewRenderer()
	}
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Converter{
		executable: executable,
		resolver:   resolver,
		renderer:   renderer,
		logger:     logger,
	}
}

// Convert writes a complete, validated DOCX document to destination.
func (converter *Converter) Convert(ctx context.Context, source []byte, documentPath string, destination io.Writer) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("convert DOCX: %w", err)
	}
	if !directExecutable(converter.executable) {
		return errors.New("convert DOCX: pandoc executable must be a resolved path")
	}
	article, err := converter.articleHTML(source, documentPath)
	if err != nil {
		return fmt.Errorf("convert DOCX: %w", err)
	}
	output, err := converter.run(ctx, []byte(article))
	if err != nil {
		return fmt.Errorf("convert DOCX: %w", err)
	}
	if err := validateDOCX(output); err != nil {
		return fmt.Errorf("convert DOCX: invalid pandoc output: %w", err)
	}
	if _, err := io.Copy(destination, bytes.NewReader(output)); err != nil {
		return fmt.Errorf("convert DOCX: write destination: %w", err)
	}
	return nil
}

func directExecutable(executable string) bool {
	if executable == "" || strings.ContainsRune(executable, '\x00') {
		return false
	}
	// A private working directory would change the meaning of a relative path.
	return filepath.IsAbs(executable)
}

func (converter *Converter) run(ctx context.Context, article []byte) ([]byte, error) {
	directory, err := os.MkdirTemp("", "mavis-docx-")
	if err != nil {
		return nil, fmt.Errorf("create private temporary directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			converter.logger.Printf("remove DOCX temporary directory: %v", err)
		}
	}()

	inputPath := filepath.Join(directory, "document.html")
	outputPath := filepath.Join(directory, "document.docx")
	if err := os.WriteFile(inputPath, article, 0o600); err != nil {
		return nil, fmt.Errorf("write private HTML input: %w", err)
	}

	command := exec.CommandContext(ctx, converter.executable,
		"--sandbox", "-f", "html", "-t", "docx", "-o", outputPath, inputPath)
	command.Dir = directory
	command.Env = []string{}
	command.WaitDelay = time.Second
	stdout := &boundedLog{}
	stderr := &boundedLog{}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		converter.logProcessOutput(stdout, stderr)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("pandoc process: %w", ctxErr)
		}
		return nil, fmt.Errorf("pandoc process: %w", err)
	}
	converter.logProcessOutput(stdout, stderr)
	return readOutput(outputPath)
}

func (converter *Converter) logProcessOutput(stdout, stderr *boundedLog) {
	for _, stream := range []struct {
		name string
		log  *boundedLog
	}{{"stdout", stdout}, {"stderr", stderr}} {
		if stream.log.Len() == 0 && !stream.log.truncated {
			continue
		}
		suffix := ""
		if stream.log.truncated {
			suffix = " (truncated)"
		}
		converter.logger.Printf("pandoc %s%s: %q", stream.name, suffix, stream.log.String())
	}
}

type boundedLog struct {
	buffer    bytes.Buffer
	truncated bool
}

func (log *boundedLog) Write(data []byte) (int, error) {
	remaining := maxLogSize - log.buffer.Len()
	if remaining > 0 {
		if len(data) <= remaining {
			log.buffer.Write(data)
		} else {
			log.buffer.Write(data[:remaining])
			log.truncated = true
		}
	}
	if len(data) > remaining {
		log.truncated = true
	}
	return len(data), nil
}

func (log *boundedLog) Len() int {
	return log.buffer.Len()
}

func (log *boundedLog) String() string {
	return log.buffer.String()
}

func readOutput(outputPath string) (output []byte, returnErr error) {
	info, err := os.Lstat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("inspect pandoc output: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("pandoc output is not a regular file")
	}
	if info.Size() > maxOutputSize {
		return nil, fmt.Errorf("pandoc output exceeds %d bytes", maxOutputSize)
	}

	file, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("open pandoc output: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			output = nil
			returnErr = fmt.Errorf("close pandoc output: %w", closeErr)
		}
	}()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect open pandoc output: %w", err)
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		return nil, errors.New("pandoc output changed while opening")
	}
	output, err = io.ReadAll(io.LimitReader(file, maxOutputSize+1))
	if err != nil {
		return nil, fmt.Errorf("read pandoc output: %w", err)
	}
	if int64(len(output)) > maxOutputSize {
		return nil, fmt.Errorf("pandoc output exceeds %d bytes", maxOutputSize)
	}
	return output, nil
}

func validateDOCX(output []byte) error {
	archive, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
	if err != nil {
		return fmt.Errorf("open ZIP container: %w", err)
	}
	required := map[string]bool{
		"[Content_Types].xml": false,
		"word/document.xml":   false,
	}
	for _, entry := range archive.File {
		if _, ok := required[entry.Name]; ok && !entry.FileInfo().IsDir() {
			required[entry.Name] = true
		}
	}
	for name, found := range required {
		if !found {
			return fmt.Errorf("DOCX container is missing %s", name)
		}
	}
	return nil
}
