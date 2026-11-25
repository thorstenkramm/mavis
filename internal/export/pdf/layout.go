package pdf

import (
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/go-swiss/fonts"
	goldpdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/stephenafamo/goldmark-pdf/gopdf"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const (
	// Page geometry in points: 2 cm side and top margins, and a deeper bottom
	// margin that keeps body text clear of the footer.
	pageMargin       = 56.7
	pageMarginBottom = 70.9
	footerOffset     = 42.0
	footerSize       = 9
	footerLineHeight = 12.0
	pageNumberWidth  = 40.0

	bodySize       = 11.0
	bodySpacing    = 5.0
	bodyLineHeight = bodySize + bodySpacing
	// A heading needs room for itself and about three lines of what follows.
	headingKeepWithNext = 6 * bodyLineHeight

	bodyFamily = "Roboto"
	// boldFamily is an alias, not a font file: upstream styles cannot ask for
	// a weight, so glyphPDF.SetFont maps this family to Roboto Bold.
	boldFamily = "Roboto Bold"
)

var (
	inkColor    = color.RGBA{R: 21, G: 24, B: 30, A: 255}
	mutedColor  = color.RGBA{R: 95, G: 104, B: 117, A: 255}
	accentColor = color.RGBA{R: 11, G: 107, B: 107, A: 255}
	paperColor  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	headerFill  = color.RGBA{R: 228, G: 230, B: 234, A: 255}
	rowFill     = color.RGBA{R: 246, G: 247, B: 248, A: 255}

	boldFont = goldpdf.Font{CanUseForText: true, Category: "sans-serif", Family: boldFamily, Type: goldpdf.FontTypeCustom}
)

// codeTheme is a light syntax theme with explicit text and background colors.
// Upstream reads both from the theme and treats unset ones as white.
var codeTheme = buildCodeTheme()

func buildCodeTheme() *chroma.Style {
	theme, err := styles.Get("github").Builder().
		Add(chroma.Background, "bg:#f1f2f4").
		Add(chroma.Text, "#15181e bg:#f1f2f4").
		Build()
	if err != nil {
		return styles.Fallback
	}
	return theme
}

// documentStyles is Mavis's PDF style profile: a clear heading scale in bold,
// body text with generous leading, and the interface's ink and accent colors.
// Sizes are whole points because the upstream renderer truncates them.
func documentStyles() goldpdf.Styles {
	result := goldpdf.DefaultStyles()
	heading := func(size, spacing float64, text color.Color) *goldpdf.Style {
		return &goldpdf.Style{Font: boldFont, Size: size, Spacing: spacing, TextColor: text, FillColor: paperColor}
	}
	result.H1 = heading(22, 6, inkColor)
	result.H2 = heading(16, 6, inkColor)
	result.H3 = heading(13, 5, inkColor)
	result.H4 = heading(12, 5, inkColor)
	result.H5 = heading(bodySize, bodySpacing, inkColor)
	result.H6 = heading(bodySize, bodySpacing, mutedColor)
	result.Normal = &goldpdf.Style{Font: goldpdf.FontRoboto, Size: bodySize, Spacing: bodySpacing, TextColor: inkColor, FillColor: paperColor}
	result.Blockquote = &goldpdf.Style{Font: goldpdf.FontRoboto, Size: bodySize, Spacing: bodySpacing, TextColor: mutedColor, FillColor: paperColor}
	result.THeader = &goldpdf.Style{Font: boldFont, Size: 10, Spacing: 4, TextColor: inkColor, FillColor: headerFill}
	result.TBody = &goldpdf.Style{Font: goldpdf.FontRoboto, Size: 10, Spacing: 4, TextColor: inkColor, FillColor: rowFill}
	result.CodeFont = goldpdf.FontRobotoMono
	result.CodeBlockTheme = codeTheme
	result.LinkColor = accentColor
	return result
}

func withDocumentStyles() goldpdf.Option {
	return goldpdf.OptionFunc(func(config *goldpdf.Config) {
		config.Styles = documentStyles()
	})
}

// applyPageMargins widens the backend's default margins and moves the cursor
// of the already opened first page to the new origin.
func applyPageMargins(backend goldpdf.PDF) {
	backend.SetMarginLeft(pageMargin)
	backend.SetMarginRight(pageMargin)
	backend.SetMarginTop(pageMargin)
	backend.SetMarginBottom(pageMarginBottom)
	backend.SetY(pageMargin)
	backend.SetX(pageMargin)
}

// canonicalFontStyle orders style letters as the backends register them.
func canonicalFontStyle(style string, bold bool) string {
	style = strings.ToUpper(style)
	var result strings.Builder
	if bold || strings.Contains(style, "B") {
		result.WriteByte('B')
	}
	for _, letter := range "IU" {
		if strings.ContainsRune(style, letter) {
			result.WriteRune(letter)
		}
	}
	return result.String()
}

// pageFooter draws the document title and page number in the bottom margin of
// every page. It talks to gopdf directly: the wrapper mirrors font and color
// for restoration after a page break, and the footer must not disturb that.
func pageFooter(title string, coverage runeCoverage) func(*gopdf.Impl, fonts.Cache) func() {
	return func(backend *gopdf.Impl, _ fonts.Cache) func() {
		return func() {
			document := backend.GoPdf
			if err := document.SetFont(bodyFamily, "", footerSize); err != nil {
				return
			}
			x, y := document.GetX(), document.GetY()
			width, height := backend.GetPageSize()
			baseline := height - footerOffset
			document.SetTextColor(mutedColor.R, mutedColor.G, mutedColor.B)

			// A footer that cannot be measured or drawn is left out rather
			// than failing the export.
			measure := func(candidate string) float64 {
				measured, err := document.MeasureTextWidth(candidate)
				if err != nil {
					return width
				}
				return measured
			}
			draw := func(left float64, value string) {
				if value == "" {
					return
				}
				document.SetXY(left, baseline)
				if err := document.Cell(nil, value); err != nil {
					return
				}
			}
			number := strconv.Itoa(document.GetNumberOfPages())
			if numberWidth := measure(number); numberWidth <= pageNumberWidth {
				draw(width-pageMargin-numberWidth, number)
			}
			draw(pageMargin, fitText(normalizeGlyphs(title, coverage), width-2*pageMargin-pageNumberWidth, measure))
			document.SetXY(x, y)
		}
	}
}

// fitText shortens text with an ellipsis until measure reports that it fits.
func fitText(value string, available float64, measure func(string) float64) string {
	if measure(value) <= available {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if candidate := strings.TrimRight(string(runes), " ") + "..."; measure(candidate) <= available {
			return candidate
		}
	}
	return ""
}

// Goldmark has already parsed this RawHTML node as a single tag, including
// its attributes. Match the element name without confusing <break> with <br>.
var lineBreakTag = regexp.MustCompile(`(?i)^<br[\s/>]`)

// lineBreakMarker is the String node value that stands for an explicit break.
const lineBreakMarker = "\n"

// lineBreakTransformer keeps <br> elements, which the PDF renderer otherwise
// drops along with all raw HTML, so the words around them no longer run
// together. Table cells read the marker as a space.
type lineBreakTransformer struct{}

func (lineBreakTransformer) Transform(document *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	var breaks []*ast.RawHTML
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		raw, ok := node.(*ast.RawHTML)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		var tag strings.Builder
		for index := 0; index < raw.Segments.Len(); index++ {
			segment := raw.Segments.At(index)
			tag.Write(segment.Value(source))
		}
		if lineBreakTag.MatchString(strings.TrimSpace(tag.String())) {
			breaks = append(breaks, raw)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		panic(err)
	}
	for _, raw := range breaks {
		if parent := raw.Parent(); parent != nil {
			parent.ReplaceChild(parent, raw, ast.NewString([]byte(lineBreakMarker)))
		}
	}
}
