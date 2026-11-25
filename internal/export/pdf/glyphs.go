package pdf

import (
	"fmt"
	"io"
	"strings"
	"unicode"

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
	// outline receives a bookmark for every heading; nil disables bookmarks.
	outline *documentOutline
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

// SetFont resolves the bold alias that heading and table-header styles use,
// because upstream styles carry a font family but no weight.
func (pdf *glyphPDF) SetFont(family string, style string, size int) error {
	if family == boldFamily {
		family, style = bodyFamily, canonicalFontStyle(style, true)
	}
	pdf.current = fontKey{family: family, style: normalizeFontStyle(style)}
	return pdf.PDF.SetFont(family, style, size)
}

// AddInternalLink marks the start of a heading, which upstream anchors before
// drawing it. A heading stranded at the bottom of a page is moved to the next
// one together with the text it introduces.
func (pdf *glyphPDF) AddInternalLink(anchor string) {
	pdf.breakPageUnlessFits(headingKeepWithNext)
	if pdf.outline != nil {
		pdf.outline.mark(anchor)
	}
	pdf.PDF.AddInternalLink(anchor)
}

// Write links the collected bookmarks into their hierarchy before the
// document is serialized.
func (pdf *glyphPDF) Write(destination io.Writer) error {
	if pdf.outline != nil {
		pdf.outline.link()
	}
	return pdf.PDF.Write(destination)
}

// breakPageUnlessFits starts a new page when less than required points remain
// above the bottom margin, and reports how far the cursor moved vertically.
func (pdf *glyphPDF) breakPageUnlessFits(required float64) float64 {
	_, height := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	top := pdf.GetY()
	if limit := height - bottom; top > limit || top+required <= limit {
		return 0
	}
	pdf.AddPage()
	return pdf.GetY() - top
}

// Circle draws a list bullet. Upstream draws it before the item's text, so a
// bullet at the very bottom of a page would be left behind when the text moves
// to the next page. Break the page first when a body line no longer fits.
func (pdf *glyphPDF) Circle(x, y, radius float64) {
	pdf.PDF.Circle(x, y+pdf.breakPageUnlessFits(bodyLineHeight), radius)
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
		coverage = pdf.coverage[fontKey{family: bodyFamily}]
	}
	return normalizeGlyphs(text, coverage)
}

// spaceFallbacks maps spaces and hyphens that the embedded fonts lack to
// covered characters with the same meaning, tried in order. Without it a
// narrow no-break space, as in "z. B.", becomes a visible replacement glyph.
var spaceFallbacks = map[rune][]rune{
	'\u00A0': {' '},
	'\u2007': {'\u00A0', ' '},
	'\u202F': {'\u00A0', ' '},
	'\u2011': {'-'},
}

// normalizeGlyphs replaces every rune the font cannot draw. Unsupported
// spaces fall back to a plain or no-break space; anything else becomes a
// visible replacement so missing content is never silent.
func normalizeGlyphs(text string, coverage runeCoverage) string {
	if coverage == nil {
		return text
	}
	supported := func(r rune) bool {
		_, ok := coverage[int(r)]
		return ok
	}

	var normalized strings.Builder
	normalized.Grow(len(text))
	for _, r := range text {
		switch {
		case r == '\u00AD':
			// A soft hyphen is only a break opportunity; fonts that cover it
			// would draw it as a visible hyphen in the middle of a word.
		case r == '\n' || r == '\r' || r == '\t' || supported(r):
			normalized.WriteRune(r)
		case substitute(r, supported) != 0:
			normalized.WriteRune(substitute(r, supported))
		case supported('\uFFFD'):
			normalized.WriteRune('\uFFFD')
		default:
			normalized.WriteByte('?')
		}
	}
	return normalized.String()
}

func substitute(r rune, supported func(rune) bool) rune {
	candidates, known := spaceFallbacks[r]
	if !known && unicode.Is(unicode.Zs, r) {
		candidates = []rune{' '}
	}
	for _, candidate := range candidates {
		if supported(candidate) {
			return candidate
		}
	}
	return 0
}

// glyphTextRenderer compensates for goldmark-pdf's Writer.Text, which removes
// non-BMP runes before calling PDF.WriteText. Normalizing while the original
// text node is still available makes emoji and other supplementary runes
// visible replacements instead of silently disappearing.
type glyphTextRenderer struct {
	glyphs *glyphPDF
	// afterBreak is set once an explicit line break was written, so the
	// whitespace that follows it in the source does not indent the new line.
	afterBreak bool
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
		renderer.write(writer, string(node.Segment.Value(source)))
		switch {
		case node.HardLineBreak():
			renderer.lineBreak(writer)
		case node.SoftLineBreak():
			renderer.write(writer, " ")
		}
	case *ast.String:
		if string(node.Value) == lineBreakMarker {
			renderer.lineBreak(writer)
		} else {
			renderer.write(writer, string(node.Value))
		}
	}
	return ast.WalkContinue, nil
}

func (renderer *glyphTextRenderer) write(writer *goldpdf.Writer, text string) {
	if renderer.afterBreak {
		text = strings.TrimLeft(text, " \t")
	}
	if text == "" {
		return
	}
	renderer.afterBreak = false
	writer.WriteText(renderer.glyphs.normalize(text))
}

// lineBreak ends the current line. Writer.WriteText flattens newlines to
// spaces, so the break goes to the backend, which returns to the left margin
// of the enclosing block.
func (renderer *glyphTextRenderer) lineBreak(writer *goldpdf.Writer) {
	writer.Pdf.WriteText(bodyLineHeight, "\n")
	renderer.afterBreak = true
}
