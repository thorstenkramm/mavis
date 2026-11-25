package pdf

import (
	"io"
	"testing"

	"context"
	"github.com/go-swiss/fonts"
	goldpdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/stephenafamo/goldmark-pdf/gopdf"
)

type recordedImage struct {
	id     string
	format string
	data   []byte
}

type recordedLink struct {
	text        string
	destination string
}

type recordingPDF struct {
	texts         []string
	cells         []string
	measured      []string
	split         []string
	images        []recordedImage
	externalLinks []recordedLink
	internalLinks []recordedLink
	writeErr      error
}

var _ goldpdf.PDF = (*recordingPDF)(nil)

func (pdf *recordingPDF) AddPage()                        {}
func (pdf *recordingPDF) GetX() float64                   { return 0 }
func (pdf *recordingPDF) GetY() float64                   { return 0 }
func (pdf *recordingPDF) SetX(float64)                    {}
func (pdf *recordingPDF) SetY(float64)                    {}
func (pdf *recordingPDF) GetPageSize() (float64, float64) { return 595, 842 }
func (pdf *recordingPDF) SetMarginLeft(float64)           {}
func (pdf *recordingPDF) SetMarginRight(float64)          {}
func (pdf *recordingPDF) SetMarginTop(float64)            {}
func (pdf *recordingPDF) SetMarginBottom(float64)         {}
func (pdf *recordingPDF) GetMargins() (float64, float64, float64, float64) {
	return 28, 28, 28, 56
}
func (pdf *recordingPDF) AddFont(string, string, []byte) error { return nil }
func (pdf *recordingPDF) SetFont(string, string, int) error    { return nil }
func (pdf *recordingPDF) WriteText(_ float64, text string) {
	pdf.texts = append(pdf.texts, text)
}
func (pdf *recordingPDF) CellFormat(
	_, _ float64,
	text, _ string,
	_ int,
	_ string,
	_ bool,
	_ int,
	_ string,
) {
	if text != "" {
		pdf.cells = append(pdf.cells, text)
	}
}
func (pdf *recordingPDF) BR(float64)             {}
func (pdf *recordingPDF) AddInternalLink(string) {}
func (pdf *recordingPDF) WriteInternalLink(_ float64, text, anchor string) {
	pdf.internalLinks = append(pdf.internalLinks, recordedLink{text: text, destination: anchor})
}
func (pdf *recordingPDF) WriteExternalLink(_ float64, text, destination string) {
	pdf.externalLinks = append(pdf.externalLinks, recordedLink{text: text, destination: destination})
}
func (pdf *recordingPDF) RegisterImage(id, format string, source io.Reader) {
	data, err := io.ReadAll(source)
	if err != nil {
		panic(err)
	}
	pdf.images = append(pdf.images, recordedImage{id: id, format: format, data: data})
}
func (pdf *recordingPDF) UseImage(string, float64, float64, float64, float64) {}
func (pdf *recordingPDF) ImageNaturalSize(string) (float64, float64)          { return 100, 100 }
func (pdf *recordingPDF) MeasureTextWidth(text string) float64 {
	pdf.measured = append(pdf.measured, text)
	return float64(len([]rune(text)) * 10)
}
func (pdf *recordingPDF) SplitText(text string, _ float64) []string {
	pdf.split = append(pdf.split, text)
	return []string{text}
}
func (pdf *recordingPDF) SetDrawColor(uint8, uint8, uint8)        {}
func (pdf *recordingPDF) SetFillColor(uint8, uint8, uint8)        {}
func (pdf *recordingPDF) SetTextColor(uint8, uint8, uint8)        {}
func (pdf *recordingPDF) SetLineWidth(float64)                    {}
func (pdf *recordingPDF) Line(float64, float64, float64, float64) {}
func (pdf *recordingPDF) Circle(float64, float64, float64)        {}
func (pdf *recordingPDF) Write(destination io.Writer) error {
	if pdf.writeErr != nil {
		return pdf.writeErr
	}
	_, err := io.WriteString(destination, "%PDF-recorded\n%%EOF\n")
	return err
}

func recordingConverter(t *testing.T, converter *Converter) *recordingPDF {
	t.Helper()
	recorder := &recordingPDF{}
	converter.newBackend = func(context.Context, gopdf.Config, fonts.Cache) goldpdf.PDF {
		return recorder
	}
	return recorder
}
