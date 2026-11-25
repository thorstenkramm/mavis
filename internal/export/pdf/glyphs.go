package pdf

import (
	"fmt"
	"strings"

	goldpdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/yuin/goldmark/ast"
)

// glyphPDF normalizes text at every public measurement and drawing entry
// point. In particular, table layout measures through SplitText while ordinary
// paragraphs measure through MeasureTextWidth; both must see the same text that
// is eventually drawn.
type glyphPDF struct {
	goldpdf.PDF
	coverage map[fontKey]runeCoverage
	current  fontKey
}

func newGlyphPDF(backend goldpdf.PDF, coverage map[fontKey]runeCoverage) *glyphPDF {
	fonts := make(map[fontKey]runeCoverage, len(coverage))
	for key, glyphs := range coverage {
		fonts[key] = glyphs
	}
	return &glyphPDF{
		PDF:      backend,
		coverage: fonts,
		current:  fontKey{family: "Roboto"},
	}
}

func (pdf *glyphPDF) AddFont(family string, style string, data []byte) error {
	coverage, err := parseFontCoverage(data)
	if err != nil {
		return fmt.Errorf("inspect font coverage: %w", err)
	}
	pdf.coverage[fontKey{family: family, style: normalizeFontStyle(style)}] = coverage
	return pdf.PDF.AddFont(family, style, data)
}

func (pdf *glyphPDF) SetFont(family string, style string, size int) error {
	pdf.current = fontKey{family: family, style: normalizeFontStyle(style)}
	return pdf.PDF.SetFont(family, style, size)
}

func (pdf *glyphPDF) WriteText(height float64, text string) {
	pdf.PDF.WriteText(height, pdf.normalize(text))
}

func (pdf *glyphPDF) CellFormat(
	width, height float64,
	text, border string,
	line int,
	align string,
	fill bool,
	link int,
	linkTarget string,
) {
	pdf.PDF.CellFormat(width, height, pdf.normalize(text), border, line, align, fill, link, linkTarget)
}

func (pdf *glyphPDF) WriteInternalLink(height float64, text string, anchor string) {
	pdf.PDF.WriteInternalLink(height, pdf.normalize(text), anchor)
}

func (pdf *glyphPDF) WriteExternalLink(height float64, text string, destination string) {
	pdf.PDF.WriteExternalLink(height, pdf.normalize(text), destination)
}

func (pdf *glyphPDF) MeasureTextWidth(text string) float64 {
	return pdf.PDF.MeasureTextWidth(pdf.normalize(text))
}

func (pdf *glyphPDF) SplitText(text string, width float64) []string {
	return pdf.PDF.SplitText(pdf.normalize(text), width)
}

func (pdf *glyphPDF) normalize(text string) string {
	coverage, ok := pdf.coverage[pdf.current]
	if !ok {
		coverage = pdf.coverage[fontKey{family: "Roboto"}]
	}
	if coverage == nil {
		return text
	}

	var normalized strings.Builder
	normalized.Grow(len(text))
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' {
			normalized.WriteRune(r)
			continue
		}
		if _, supported := coverage[int(r)]; supported {
			normalized.WriteRune(r)
			continue
		}
		if _, supported := coverage[int('\uFFFD')]; supported {
			normalized.WriteRune('\uFFFD')
		} else {
			normalized.WriteByte('?')
		}
	}
	return normalized.String()
}

// glyphTextRenderer compensates for goldmark-pdf's Writer.Text, which removes
// non-BMP runes before calling PDF.WriteText. Normalizing while the original
// text node is still available makes emoji and other supplementary runes
// visible replacements instead of silently disappearing.
type glyphTextRenderer struct {
	glyphs *glyphPDF
}

func (renderer *glyphTextRenderer) RegisterFuncs(registerer goldpdf.NodeRendererFuncRegisterer) {
	registerer.Register(ast.KindText, renderer.renderText)
	registerer.Register(ast.KindString, renderer.renderText)
}

func (renderer *glyphTextRenderer) renderText(
	writer *goldpdf.Writer,
	source []byte,
	node ast.Node,
	entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	switch node := node.(type) {
	case *ast.Text:
		text := string(node.Segment.Value(source))
		if node.SoftLineBreak() {
			text += " "
		}
		writer.WriteText(renderer.glyphs.normalize(text))
	case *ast.String:
		writer.WriteText(renderer.glyphs.normalize(string(node.Value)))
	}
	return ast.WalkContinue, nil
}
