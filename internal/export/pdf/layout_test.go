package pdf

import (
	"bytes"
	"context"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/go-swiss/fonts"
	goldpdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/stephenafamo/goldmark-pdf/gopdf"
)

// layoutPDF records the layout calls that recordingPDF ignores and lets a
// test place the cursor near the bottom of the page.
type layoutPDF struct {
	recordingPDF
	y        float64
	pages    int
	fonts    []string
	circles  []float64
	anchors  []string
	margins  [4]float64
	position [2]float64
	outline  *documentOutline
}

// recordingFactory makes Convert use backend and captures its outline.
func recordingFactory(backend *layoutPDF) backendFactory {
	return func(context.Context, gopdf.Config, fonts.Cache) goldpdf.PDF {
		return backend
	}
}

func (pdf *layoutPDF) AddPage()                 { pdf.pages++; pdf.y = 56 }
func (pdf *layoutPDF) GetY() float64            { return pdf.y }
func (pdf *layoutPDF) SetX(x float64)           { pdf.position[0] = x }
func (pdf *layoutPDF) SetY(y float64)           { pdf.position[1] = y }
func (pdf *layoutPDF) SetMarginLeft(m float64)  { pdf.margins[0] = m }
func (pdf *layoutPDF) SetMarginTop(m float64)   { pdf.margins[1] = m }
func (pdf *layoutPDF) SetMarginRight(m float64) { pdf.margins[2] = m }
func (pdf *layoutPDF) SetMarginBottom(m float64) {
	pdf.margins[3] = m
}
func (pdf *layoutPDF) GetMargins() (float64, float64, float64, float64) { return 56, 56, 56, 70 }
func (pdf *layoutPDF) SetFont(family, style string, _ int) error {
	pdf.fonts = append(pdf.fonts, family+"/"+style)
	return nil
}
func (pdf *layoutPDF) Circle(_, y, _ float64)        { pdf.circles = append(pdf.circles, y) }
func (pdf *layoutPDF) AddInternalLink(anchor string) { pdf.anchors = append(pdf.anchors, anchor) }

func newLayoutGlyphs(t *testing.T, backend *layoutPDF) *glyphPDF {
	t.Helper()
	cache, err := loadOfflineFonts()
	if err != nil {
		t.Fatal(err)
	}
	return newGlyphPDF(backend, cache.coverage)
}

func TestLineBreaksSurviveAsBreaksAndNeverJoinWords(t *testing.T) {
	converter, err := New(nil, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	recorder := recordingConverter(t, converter)
	source := []byte("**Role**: Team<br>\n**Duty**: Operates systems<BR />and reviews.\n\n" +
		"Hard  \nbreak and soft\nbreak.\n\n" +
		"Inline <span>kept text</span> stays.\n\n" +
		"| A | B |\n| --- | --- |\n| one<br>two | three |\n")
	var output bytes.Buffer
	if err := converter.Convert(context.Background(), source, "guide.md", &output); err != nil {
		t.Fatal(err)
	}
	flow := strings.Join(recorder.texts, "")
	for _, expected := range []string{"Team\nDuty", "systems\nand reviews.", "Hard\nbreak and soft break.", "kept text"} {
		if !strings.Contains(flow, expected) {
			t.Errorf("flowing text does not contain %q in %q", expected, flow)
		}
	}
	for _, joined := range []string{"TeamDuty", "systemsand", "Hardbreak", "softbreak", "<br", "<BR"} {
		if strings.Contains(flow, joined) {
			t.Errorf("flowing text contains %q: %q", joined, flow)
		}
	}
	if cells := strings.Join(recorder.split, "|"); !strings.Contains(cells, "one two") || strings.Contains(cells, "onetwo") {
		t.Errorf("table cell lost its break: %q", cells)
	}
}

func TestLineBreakIsNotFollowedByAnIndent(t *testing.T) {
	converter, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := recordingConverter(t, converter)
	if err := converter.Convert(context.Background(), []byte("first<br>\n   second\n\nnext paragraph\n"), "a.md", io.Discard); err != nil {
		t.Fatal(err)
	}
	for index, text := range recorder.texts {
		if text == "\n" && index+1 < len(recorder.texts) && strings.HasPrefix(recorder.texts[index+1], " ") {
			t.Fatalf("text after a break starts with a space: %q", recorder.texts)
		}
	}
	if flow := strings.Join(recorder.texts, ""); !strings.Contains(flow, "first\nsecond") || !strings.Contains(flow, "next paragraph") {
		t.Fatalf("flow = %q", flow)
	}
}

func TestUnsupportedSpacesAndHyphensUseCoveredEquivalents(t *testing.T) {
	cache, err := loadOfflineFonts()
	if err != nil {
		t.Fatal(err)
	}
	roboto := cache.coverage[fontKey{family: bodyFamily}]
	if _, covered := roboto[0x202F]; covered {
		t.Skip("the embedded font now covers the narrow no-break space")
	}
	got := normalizeGlyphs("z.\u202fB. a\u2009b soft\u00adhyphen non\u2011breaking 中", roboto)
	if strings.ContainsRune(got, '\u202f') || strings.Contains(got, "z.\ufffdB") {
		t.Fatalf("narrow no-break space was not substituted: %q", got)
	}
	if !strings.Contains(got, "softhyphen") || !strings.HasSuffix(got, "\ufffd") {
		t.Fatalf("soft hyphen or unknown glyph handling changed: %q", got)
	}
	for _, r := range got {
		if _, covered := roboto[int(r)]; !covered {
			t.Errorf("result still contains unsupported rune %U in %q", r, got)
		}
	}

	onlySpace := runeCoverage{int(' '): 1, int('a'): 1}
	if got := normalizeGlyphs("a\u202fa\u00a0a\u3000a\u2011a", onlySpace); got != "a a a a?a" {
		t.Fatalf("fallback chain = %q", got)
	}
	if got := normalizeGlyphs("a\u202f", nil); got != "a\u202f" {
		t.Fatalf("missing coverage must leave text unchanged: %q", got)
	}
}

func TestBoldAliasSelectsRobotoBold(t *testing.T) {
	backend := &layoutPDF{}
	glyphs := newLayoutGlyphs(t, backend)
	for _, style := range []string{"", "i", "B", "IU"} {
		if err := glyphs.SetFont(boldFamily, style, 12); err != nil {
			t.Fatal(err)
		}
	}
	if err := glyphs.SetFont(bodyFamily, "I", 11); err != nil {
		t.Fatal(err)
	}
	want := []string{"Roboto/B", "Roboto/BI", "Roboto/B", "Roboto/BIU", "Roboto/I"}
	if strings.Join(backend.fonts, ",") != strings.Join(want, ",") {
		t.Fatalf("fonts = %v, want %v", backend.fonts, want)
	}
	if glyphs.current != (fontKey{family: bodyFamily, style: normalizeFontStyle("I")}) {
		t.Fatalf("current font = %+v", glyphs.current)
	}

	styles := documentStyles()
	for name, style := range map[string]string{
		"H1": styles.H1.Font.Family, "H6": styles.H6.Font.Family, "table header": styles.THeader.Font.Family,
	} {
		if style != boldFamily {
			t.Errorf("%s font family = %q", name, style)
		}
	}
	if styles.Normal.Font.Family != bodyFamily || styles.H1.Size <= styles.H2.Size || styles.H2.Size <= styles.H3.Size {
		t.Fatalf("heading scale or body font is wrong: %+v %+v %+v", styles.H1, styles.H2, styles.H3)
	}
	text := styles.CodeBlockTheme.Get(chroma.Text)
	if !text.Colour.IsSet() || !text.Background.IsSet() { //nolint:misspell // chroma's field name
		t.Fatalf("code theme has no explicit colors: %+v", text)
	}
}

func TestHeadingsAndBulletsMoveToTheNextPageTogetherWithTheirText(t *testing.T) {
	backend := &layoutPDF{}
	glyphs := newLayoutGlyphs(t, backend)
	limit := 842.0 - 70

	backend.y = 300
	glyphs.AddInternalLink("fits")
	glyphs.Circle(10, 308, 2)
	if backend.pages != 0 || backend.circles[0] != 308 {
		t.Fatalf("content that fits triggered a page break: pages=%d circles=%v", backend.pages, backend.circles)
	}

	backend.y = limit - bodyLineHeight/2
	glyphs.Circle(10, backend.y+8, 2)
	if backend.pages != 1 || backend.circles[1] != 56+8 {
		t.Fatalf("stranded bullet: pages=%d circles=%v", backend.pages, backend.circles)
	}

	backend.y = limit - headingKeepWithNext/2
	glyphs.AddInternalLink("stranded")
	if backend.pages != 2 || backend.y != 56 || strings.Join(backend.anchors, ",") != "fits,stranded" {
		t.Fatalf("stranded heading: pages=%d y=%v anchors=%v", backend.pages, backend.y, backend.anchors)
	}

	// A cursor already in the bottom margin belongs to the footer and must
	// never start another page.
	backend.y = limit + 5
	glyphs.AddInternalLink("footer")
	glyphs.Circle(10, backend.y, 2)
	if backend.pages != 2 {
		t.Fatalf("drawing in the bottom margin added a page: %d", backend.pages)
	}
}

func TestPageMarginsAndFooterHelpers(t *testing.T) {
	backend := &layoutPDF{}
	applyPageMargins(backend)
	if backend.margins != [4]float64{pageMargin, pageMargin, pageMargin, pageMarginBottom} || backend.position != [2]float64{pageMargin, pageMargin} {
		t.Fatalf("margins = %v, cursor = %v", backend.margins, backend.position)
	}
	if pageMarginBottom-footerOffset < footerLineHeight {
		t.Fatal("the footer does not fit into the bottom margin")
	}

	measure := func(value string) float64 { return float64(len([]rune(value))) }
	for _, test := range []struct {
		value     string
		available float64
		want      string
	}{
		{"short", 10, "short"},
		{"a long title", 9, "a long..."},
		{"abcdef", 4, "a..."},
		{"abcdef", 2, ""},
	} {
		if got := fitText(test.value, test.available, measure); got != test.want {
			t.Errorf("fitText(%q, %v) = %q, want %q", test.value, test.available, got, test.want)
		}
	}
	for style, want := range map[string]string{"": "", "i": "I", "ib": "BI", "UB": "BU", "iu": "IU"} {
		if got := canonicalFontStyle(style, false); got != want {
			t.Errorf("canonicalFontStyle(%q) = %q, want %q", style, got, want)
		}
	}
}

func TestMultiPageDocumentHasAFooterOnEveryPage(t *testing.T) {
	converter, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var source strings.Builder
	source.WriteString("# A long document\n\n")
	for index := 0; index < 60; index++ {
		source.WriteString("## Section\n\nParagraph with enough words to fill a line of the page and wrap once or twice.\n\n- item one\n- item two\n\n")
	}
	var output bytes.Buffer
	if err := converter.Convert(context.Background(), []byte(source.String()), "docs/A very long file name that the footer must shorten instead of overlapping the page number at all.md", &output); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	pages := bytes.Count(output.Bytes(), []byte("/Type /Page\n")) + bytes.Count(output.Bytes(), []byte("/Type /Page "))
	if pages < 3 {
		t.Fatalf("document has %d pages, want several; size %d", pages, output.Len())
	}
}
