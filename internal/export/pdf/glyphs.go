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
	coverage    map[fontKey]runeCoverage
	current     fontKey
	inHeading   bool
	deferMarker bool
	marker      *listBullet
	stroke      [3]uint8
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
	} else if pdf.inHeading && family == goldpdf.FontRobotoMono.Family {
		style = canonicalFontStyle(style, true)
	}
	pdf.current = fontKey{family: family, style: normalizeFontStyle(style)}
	return pdf.PDF.SetFont(family, style, size)
}

// AddInternalLink marks the start of a heading, which upstream anchors before
// drawing it. A heading stranded at the bottom of a page is moved to the next
// one together with the text it introduces.
func (pdf *glyphPDF) AddInternalLink(anchor string) {
	pdf.breakPageUnlessFits(headingKeepWithNext)
	pdf.drawMarker(pdf.GetY())
	if pdf.outline != nil {
		pdf.outline.mark(anchor)
	}
	pdf.PDF.AddInternalLink(anchor)
}

// Write links the collected bookmarks into their hierarchy before the
// document is serialized.
func (pdf *glyphPDF) Write(destination io.Writer) error {
	if pdf.outline != nil {
		return pdf.outline.write(pdf.PDF, destination)
	}
	return pdf.PDF.Write(destination)
}

// breakPageUnlessFits starts a new page when less than required points remain
// above the bottom margin, and reports how far the cursor moved vertically.
func (pdf *glyphPDF) breakPageUnlessFits(required float64) float64 {
	_, height := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	top := pdf.GetY()
	if limit := height - bottom; top+required <= limit {
		return 0
	}
	pdf.AddPage()
	return pdf.GetY() - top
}

// Circle draws a list bullet. Upstream draws it before the item's text, so a
// bullet at the very bottom of a page would be left behind when the text moves
// to the next page. Break the page first when a body line no longer fits.
func (pdf *glyphPDF) Circle(x, y, radius float64) {
	if pdf.deferMarker {
		pdf.marker = &listBullet{x: x, offset: y - pdf.GetY(), radius: radius, stroke: pdf.stroke}
		return
	}
	pdf.PDF.Circle(x, y+pdf.breakPageUnlessFits(bodyLineHeight), radius)
}

type listBullet struct {
	x, offset, radius float64
	stroke            [3]uint8
}

func (pdf *glyphPDF) SetDrawColor(r, g, b uint8) {
	pdf.stroke = [3]uint8{r, g, b}
	pdf.PDF.SetDrawColor(r, g, b)
}

// Images choose their own page after registration and scaling. Position the
// deferred marker at the top of the image on that page, using its original
// stroke (gopdf draws circles with a stroked oval).
func (pdf *glyphPDF) UseImage(id string, x, y, w, h float64) {
	pdf.PDF.UseImage(id, x, y, w, h)
	if pdf.marker == nil {
		return
	}
	if h == 0 {
		nw, nh := pdf.ImageNaturalSize(id)
		if nw > 0 {
			h = w * nh / nw
		}
	}
	pdf.drawMarker(pdf.GetY() - h)
}

func (pdf *glyphPDF) drawMarker(y float64) {
	if marker := pdf.marker; marker != nil {
		pdf.PDF.SetDrawColor(marker.stroke[0], marker.stroke[1], marker.stroke[2])
		pdf.PDF.Circle(marker.x, y+marker.offset, marker.radius)
		pdf.PDF.SetDrawColor(pdf.stroke[0], pdf.stroke[1], pdf.stroke[2])
		pdf.marker = nil
	}
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
	normalized := pdf.normalize(text)
	if text != "" && normalized == "" {
		return []string{""}
	}
	return pdf.PDF.SplitText(normalized, width)
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
	// Only whitespace immediately after a break is indentation. In particular,
	// an autolink or image rendered by another handler ends that run.
	trimNext ast.Node
}

func (renderer *glyphTextRenderer) RegisterFuncs(registerer goldpdf.NodeRendererFuncRegisterer) {
	registerer.Register(ast.KindText, renderer.renderText)
	registerer.Register(ast.KindString, renderer.renderText)
	// Decorate the pinned renderer's heading and list-item handlers: layout
	// must run before they emit the anchor or marker, respectively.
	c := goldpdf.DefaultConfig()
	c.AddDefaultNodeRenderers()
	for _, entry := range c.NodeRenderers {
		if defaults, ok := entry.Value.(goldpdf.NodeRenderer); ok {
			defaults.RegisterFuncs(blockRegistrar(func(kind ast.NodeKind, render goldpdf.NodeRendererFunc) {
				if kind == ast.KindHeading || kind == ast.KindListItem || kind == ast.KindImage {
					registerer.Register(kind, renderer.beforeBlock(render))
				}
			}))
		}
	}
}

type blockRegistrar func(ast.NodeKind, goldpdf.NodeRendererFunc)

func (f blockRegistrar) Register(kind ast.NodeKind, render goldpdf.NodeRendererFunc) { f(kind, render) }

func (renderer *glyphTextRenderer) beforeBlock(render goldpdf.NodeRendererFunc) goldpdf.NodeRendererFunc {
	return func(w *goldpdf.Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := node.(type) {
		case *ast.ListItem:
			first := node.FirstChild()
			if first != nil && (first.Kind() == ast.KindParagraph || first.Kind() == ast.KindTextBlock) {
				first = first.FirstChild()
			}
			if entering && first != nil && (first.Kind() == ast.KindHeading || first.Kind() == ast.KindImage) {
				renderer.glyphs.deferMarker = true
				defer func() { renderer.glyphs.deferMarker = false }()
			}
		case *ast.Image:
			// If the authorized file disappeared before rendering, no UseImage
			// call consumes the marker. Keep it on the item's original line.
			if entering && renderer.glyphs.marker != nil {
				defer renderer.glyphs.drawMarker(w.Pdf.GetY())
			}
		case *ast.Heading:
			if entering {
				renderer.glyphs.breakPageUnlessFits(renderer.headingRoom(w, source, node) + bodySize + 2*bodySpacing)
			}
			renderer.glyphs.inHeading = entering
		}
		return render(w, source, node, entering)
	}
}

func (renderer *glyphTextRenderer) headingRoom(w *goldpdf.Writer, source []byte, h *ast.Heading) float64 {
	styles := w.Styles
	style := [...]*goldpdf.Style{styles.H1, styles.H2, styles.H3, styles.H4, styles.H5, styles.H6}[h.Level-1]
	pdf := renderer.glyphs
	goldpdf.SetStyle(pdf, *style)
	defer goldpdf.SetStyle(pdf, *styles.Normal)
	wid, height := pdf.GetPageSize()
	left, top, right, bottom := pdf.GetMargins()
	available := max(1, wid-left-right)
	lines, used := 1, 0.0
	space := pdf.MeasureTextWidth(" ")
	for i, line := range strings.Split(headingText(h, source), "\n") {
		if i > 0 {
			lines++
			used = 0
		}
		for word := range strings.FieldsSeq(line) {
			width := pdf.MeasureTextWidth(word)
			if used > 0 && used+width > available {
				lines++
				used = 0
			}
			for width > available {
				lines++
				width -= available
			}
			used += width + space
		}
	}
	return min(height-top-bottom-bodySize-2*bodySpacing, max(headingKeepWithNext, float64(lines)*(style.Size+style.Spacing)+2*bodyLineHeight))
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
		value := string(node.Segment.Value(source))
		if node.SoftLineBreak() {
			value += " "
		}
		renderer.write(writer, node, value)
		if node.HardLineBreak() {
			renderer.lineBreak(writer, node)
		}
	case *ast.String:
		if string(node.Value) == lineBreakMarker {
			renderer.lineBreak(writer, node)
		} else {
			renderer.write(writer, node, string(node.Value))
		}
	}
	return ast.WalkContinue, nil
}

func (renderer *glyphTextRenderer) write(writer *goldpdf.Writer, node ast.Node, text string) {
	if renderer.trimNext == node {
		text = strings.TrimLeft(text, " \t")
		if text == "" {
			renderer.trimNext = node.NextSibling()
			return
		}
	}
	if text == "" {
		return
	}
	renderer.trimNext = nil
	writer.WriteText(renderer.glyphs.normalize(text))
}

// lineBreak ends the current line. Writer.WriteText flattens newlines to
// spaces, so the break goes to the backend, which returns to the left margin
// of the enclosing block.
func (renderer *glyphTextRenderer) lineBreak(writer *goldpdf.Writer, node ast.Node) {
	// GetBacktickStyle copies the active style, changing only its font.
	style := writer.GetBacktickStyle()
	advance := style.Size + style.Spacing
	if renderer.glyphs.breakPageUnlessFits(advance) == 0 {
		writer.Pdf.WriteText(advance, "\n")
	}
	renderer.trimNext = node.NextSibling()
}
