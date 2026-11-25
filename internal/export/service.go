// Package export coordinates bounded, complete document conversions.
package export

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/thorstenkramm/mavis/internal/content"
)

const (
	// MaxSourceSize is the largest Markdown source accepted for conversion.
	MaxSourceSize = 10 << 20
	// MaxOutputSize is the largest generated attachment accepted for download.
	MaxOutputSize = 50 << 20
	// ConversionTimeout bounds each PDF or DOCX conversion.
	ConversionTimeout = 30 * time.Second
	// ConcurrentConversions is the shared PDF/DOCX worker limit.
	ConcurrentConversions = 2
)

// Format identifies a generated document representation.
type Format string

const (
	PDF  Format = "pdf"
	DOCX Format = "docx"
)

var (
	ErrUnknownFormat  = errors.New("unknown export format")
	ErrUnavailable    = errors.New("export format unavailable")
	ErrSourceTooLarge = errors.New("export source too large")
	ErrOutputTooLarge = errors.New("export output too large")
	ErrTimeout        = errors.New("export timed out")
	ErrCanceled       = errors.New("export canceled")
	ErrConversion     = errors.New("export conversion failed")
)

// Result is a completely generated and validated attachment.
type Result struct {
	Body        []byte
	Filename    string
	ContentType string
}

// Converter generates one representation from current Markdown source.
type Converter interface {
	Convert(ctx context.Context, source []byte, documentPath string, destination io.Writer) error
}

// Service owns shared conversion limits and worker lifetime.
type Service struct {
	pdf       Converter
	docx      Converter
	workers   chan struct{}
	timeout   time.Duration
	maxSource int64
	maxOutput int
}

// NewService constructs a converter service. A nil DOCX converter disables DOCX.
func NewService(pdfConverter, docxConverter Converter) *Service {
	return &Service{
		pdf:       pdfConverter,
		docx:      docxConverter,
		workers:   make(chan struct{}, ConcurrentConversions),
		timeout:   ConversionTimeout,
		maxSource: MaxSourceSize,
		maxOutput: MaxOutputSize,
	}
}

// DOCXAvailable reports whether startup resolved a DOCX converter.
func (service *Service) DOCXAvailable() bool {
	return service != nil && service.docx != nil
}

// Export reads current source and returns a complete generated attachment.
func (service *Service) Export(ctx context.Context, resource content.Resource, format Format) (Result, error) {
	if service == nil {
		return Result{}, ErrUnavailable
	}
	converter := service.pdf
	extension := ".pdf"
	contentType := "application/pdf"
	switch format {
	case PDF:
	case DOCX:
		converter = service.docx
		extension = ".docx"
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return Result{}, ErrUnknownFormat
	}
	if converter == nil {
		return Result{}, ErrUnavailable
	}

	if err := ctx.Err(); err != nil {
		return Result{}, contextError(err)
	}
	conversionContext, cancel := context.WithTimeout(ctx, service.timeout)
	defer cancel()
	select {
	case service.workers <- struct{}{}:
	case <-conversionContext.Done():
		return Result{}, contextError(conversionContext.Err())
	}

	source, err := service.readSource(resource.Path)
	if err != nil {
		<-service.workers
		return Result{}, err
	}
	if err := conversionContext.Err(); err != nil {
		<-service.workers
		return Result{}, contextError(err)
	}
	resultChannel := make(chan conversionResult, 1)
	go func() {
		defer func() { <-service.workers }()
		output := &limitedBuffer{maximum: service.maxOutput}
		err := convertSafely(conversionContext, converter, source, resource.RelativePath, output)
		if err == nil {
			err = output.err
		}
		resultChannel <- conversionResult{body: output.Bytes(), err: err}
	}()

	select {
	case conversion := <-resultChannel:
		if err := conversionContext.Err(); err != nil {
			return Result{}, contextError(err)
		}
		if conversion.err != nil {
			return Result{}, conversion.err
		}
		if err := validate(format, conversion.body); err != nil {
			return Result{}, err
		}
		base := path.Base(resource.RelativePath)
		filename := strings.TrimSuffix(base, path.Ext(base)) + extension
		return Result{Body: conversion.body, Filename: filename, ContentType: contentType}, nil
	case <-conversionContext.Done():
		return Result{}, contextError(conversionContext.Err())
	}
}

type conversionResult struct {
	body []byte
	err  error
}

func contextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	return ErrCanceled
}

func convertSafely(ctx context.Context, converter Converter, source []byte, documentPath string, destination io.Writer) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: panic: %v", ErrConversion, recovered)
		}
	}()
	if err := converter.Convert(ctx, source, documentPath, destination); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrTimeout
		}
		if errors.Is(err, context.Canceled) {
			return ErrCanceled
		}
		if errors.Is(err, ErrSourceTooLarge) || errors.Is(err, ErrOutputTooLarge) || errors.Is(err, ErrTimeout) || errors.Is(err, ErrCanceled) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrConversion, err)
	}
	return nil
}

func (service *Service) readSource(filename string) (source []byte, returnErr error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConversion, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			source = nil
			returnErr = fmt.Errorf("%w: %w", ErrConversion, closeErr)
		}
	}()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, ErrConversion
	}
	if info.Size() > service.maxSource {
		return nil, ErrSourceTooLarge
	}
	source, err = io.ReadAll(io.LimitReader(file, service.maxSource+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConversion, err)
	}
	if int64(len(source)) > service.maxSource {
		return nil, ErrSourceTooLarge
	}
	return source, nil
}

type limitedBuffer struct {
	buffer  bytes.Buffer
	maximum int
	err     error
}

func (buffer *limitedBuffer) Write(data []byte) (int, error) {
	if buffer.err != nil {
		return 0, buffer.err
	}
	remaining := buffer.maximum - buffer.buffer.Len()
	if len(data) > remaining {
		if remaining > 0 {
			_, _ = buffer.buffer.Write(data[:remaining])
		}
		buffer.err = ErrOutputTooLarge
		return remaining, buffer.err
	}
	return buffer.buffer.Write(data)
}

func (buffer *limitedBuffer) Bytes() []byte {
	return buffer.buffer.Bytes()
}

func validate(format Format, output []byte) error {
	switch format {
	case PDF:
		if !bytes.HasPrefix(output, []byte("%PDF-")) {
			return ErrConversion
		}
	case DOCX:
		archive, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
		if err != nil {
			return ErrConversion
		}
		required := map[string]bool{"[Content_Types].xml": false, "word/document.xml": false}
		for _, file := range archive.File {
			if _, ok := required[file.Name]; ok {
				required[file.Name] = true
			}
		}
		for _, present := range required {
			if !present {
				return ErrConversion
			}
		}
	default:
		return ErrUnknownFormat
	}
	return nil
}
