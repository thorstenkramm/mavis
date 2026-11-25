// Package pdf converts Markdown documents into self-contained PDF files.
package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"path"
	"runtime/debug"
	"strings"

	"github.com/go-swiss/fonts"
	goldpdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/stephenafamo/goldmark-pdf/gopdf"
	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type backendFactory func(context.Context, gopdf.Config, fonts.Cache) goldpdf.PDF

// Converter creates independent PDF backends for each conversion. A Converter
// is safe for concurrent use after New returns.
type Converter struct {
	resolver   *content.Resolver
	logger     *log.Logger
	fonts      *offlineFontCache
	newBackend backendFactory
	// observeOutline, when set, receives each conversion's bookmark outline.
	observeOutline func(*documentOutline)
}

// New constructs an offline Markdown-to-PDF converter. All font variants used
// by goldmark-pdf are parsed and preloaded before this function returns.
func New(resolver *content.Resolver, logger *log.Logger) (*Converter, error) {
	fontCache, err := loadOfflineFonts()
	if err != nil {
		return nil, err
	}
	converter := &Converter{
		resolver: resolver,
		logger:   logger,
		fonts:    fontCache,
	}
	converter.newBackend = func(ctx context.Context, config gopdf.Config, cache fonts.Cache) goldpdf.PDF {
		return gopdf.New(ctx, config, cache)
	}
	return converter, nil
}

// Convert writes a simplified GFM PDF generated from source. documentPath is
// the document's decoded, root-relative logical path and anchors local images.
// The library used for layout is not generally cancellable; cancellation is
// checked before and after conversion so the caller can discard late output.
func (converter *Converter) Convert(
	ctx context.Context,
	source []byte,
	documentPath string,
	destination io.Writer,
) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if destination == nil {
		return errors.New("pdf destination is nil")
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if converter.logger != nil {
				converter.logger.Printf("PDF conversion panic: %v\n%s", recovered, debug.Stack())
			}
			err = fmt.Errorf("pdf conversion panic: %v", recovered)
		}
	}()

	images, err := newResourceFS(converter.resolver, documentPath)
	if err != nil {
		return err
	}
	body := bytes.TrimPrefix(render.MarkdownBody(source), utf8BOM)
	title := documentTitle(documentPath)
	backend := converter.newBackend(ctx, gopdf.Config{
		Title:      title,
		FooterFunc: pageFooter(title, converter.fonts.coverage[fontKey{family: bodyFamily}]),
	}, converter.fonts)
	applyPageMargins(backend)
	normalized := newGlyphPDF(backend, converter.fonts.coverage)
	normalized.outline = newDocumentOutline(backend)
	if converter.observeOutline != nil {
		defer converter.observeOutline(normalized.outline)
	}

	renderer := goldpdf.New(
		goldpdf.WithContext(ctx),
		goldpdf.WithPDF(normalized),
		goldpdf.WithImageFS(images),
		goldpdf.WithFontsCache(converter.fonts),
		goldpdf.WithEscapeHTML(false),
		goldpdf.WithLogger(upstreamLogger(converter.logger)),
		withDocumentStyles(),
		goldpdf.WithNodeRenderers(util.Prioritized(&glyphTextRenderer{glyphs: normalized}, 500)),
	)
	markdown := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(&imageASTTransformer{images: images}, 1000),
				util.Prioritized(lineBreakTransformer{}, 1001),
				util.Prioritized(normalized.outline, 1002),
			),
		),
		goldmark.WithRenderer(renderer),
	)
	if err := markdown.Convert(body, destination); err != nil {
		return fmt.Errorf("convert PDF: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func documentTitle(documentPath string) string {
	base := path.Base(strings.TrimPrefix(documentPath, "/"))
	extension := path.Ext(base)
	return strings.TrimSuffix(base, extension)
}

type standardLogWriter struct {
	logger *log.Logger
}

func (writer standardLogWriter) Write(data []byte) (int, error) {
	writer.logger.Print(strings.TrimSuffix(string(data), "\n"))
	return len(data), nil
}

func upstreamLogger(logger *log.Logger) *slog.Logger {
	if logger == nil {
		return nil
	}
	return slog.New(slog.NewTextHandler(standardLogWriter{logger: logger}, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
}
