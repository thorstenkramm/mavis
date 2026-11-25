package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-swiss/fonts"
	goldpdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/stephenafamo/goldmark-pdf/gopdf"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestConvertRealPDFOffline(t *testing.T) {
	var requests atomic.Int64
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, fmt.Errorf("unexpected network request for %s", request.URL)
	})
	oldTransport := http.DefaultTransport
	oldClient := http.DefaultClient
	http.DefaultTransport = transport
	http.DefaultClient = &http.Client{Transport: transport}
	t.Cleanup(func() {
		http.DefaultTransport = oldTransport
		http.DefaultClient = oldClient
	})

	converter, err := New(nil, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	var output bytes.Buffer
	source := []byte("# Grüße\n\nGerman: äöü ÄÖÜ ß. CJK 中文 and emoji 😀.\n\n" +
		"| Name | Wert |\n| --- | --- |\n| Käse | Müller |\n\n" +
		"```go\nfmt.Println(\"Grüße\")\n```\n\n" +
		"![remote](https://example.invalid/secret.png)\n")
	if err := converter.Convert(context.Background(), source, "guide.md", &output); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatalf("output does not start with a PDF signature: %q", output.Bytes()[:min(16, output.Len())])
	}
	if !bytes.Contains(output.Bytes(), []byte("%%EOF")) {
		t.Fatal("output has no PDF EOF marker")
	}
	if !bytes.Contains(output.Bytes(), []byte("Roboto")) {
		t.Fatal("output does not identify the embedded Roboto font")
	}
	if requests.Load() != 0 {
		t.Fatalf("conversion made %d network requests", requests.Load())
	}
	if converter.fonts.cacheHits.Load() == 0 {
		t.Fatal("conversion did not use the preloaded font cache")
	}
}

func TestConvertRecordingBackendGFMAndGlyphs(t *testing.T) {
	converter, err := New(nil, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	recorder := recordingConverter(t, converter)
	source := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`---
title: front-secret
---
# Überschrift

Grüße, **fett** und *kursiv* mit äöü ÄÖÜ ß.

- eins
- zwei

1. erstens
2. zweitens

| Name | Wert |
| ---- | ---- |
| Käse | Müller |

[Mavis](https://example.com/docs) and `+"`inline <code>`"+`.

`+"```go\nfmt.Println(\"Grüße\")\n```"+`

- [x] erledigt
- [ ] offen

CJK 中文 and emoji 😀.

Literal comparison: 2 < 3 & 4 > 1. Raw <span>HTML text</span>.
`)...)
	var output bytes.Buffer
	if err := converter.Convert(context.Background(), source, "docs/guide.md", &output); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	allText := strings.Join(append(append(append([]string{}, recorder.texts...), recorder.cells...), recorder.split...), "")
	for _, link := range recorder.externalLinks {
		allText += link.text
	}
	for _, link := range recorder.internalLinks {
		allText += link.text
	}
	for _, expected := range []string{
		"Überschrift", "Grüße", "fett", "kursiv", "äöü ÄÖÜ ß",
		"eins", "zwei", "erstens", "Käse", "Müller", "Mavis",
		"inline <code>", "fmt.Println", "erledigt", "offen",
		"2 < 3 & 4 > 1", "HTML text",
	} {
		if !strings.Contains(allText, expected) {
			t.Errorf("recorded PDF text does not contain %q\n%s", expected, allText)
		}
	}
	for _, forbidden := range []string{"front-secret", "中文", "😀", "\uFEFF", "<span>"} {
		if strings.Contains(allText, forbidden) {
			t.Errorf("recorded PDF text unexpectedly contains %q\n%s", forbidden, allText)
		}
	}
	if !strings.Contains(allText, "CJK �� and emoji �") {
		t.Errorf("unsupported glyphs were not visibly replaced: %q", allText)
	}
	if len(recorder.externalLinks) != 1 || recorder.externalLinks[0].destination != "https://example.com/docs" {
		t.Errorf("external links = %+v", recorder.externalLinks)
	}
	if output.String() != "%PDF-recorded\n%%EOF\n" {
		t.Fatalf("recorded output = %q", output.String())
	}
}

func TestConvertBOMWithoutFrontMatter(t *testing.T) {
	converter, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := recordingConverter(t, converter)
	source := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Plain body\n")...)
	if err := converter.Convert(context.Background(), source, "plain.md", io.Discard); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(recorder.texts, ""), "\uFEFF") {
		t.Fatal("BOM reached PDF text")
	}
}

func TestGlyphMeasurementAndDrawingUseIdenticalText(t *testing.T) {
	converter, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := &recordingPDF{}
	wrapped := newGlyphPDF(recorder, converter.fonts.coverage)
	regular := converter.fonts.values[fontAssets[0].url]
	if err := wrapped.AddFont("Roboto", "", regular); err != nil {
		t.Fatal(err)
	}
	if err := wrapped.SetFont("Roboto", "", 12); err != nil {
		t.Fatal(err)
	}
	input := "A中文😀B"
	expected := wrapped.normalize(input)
	wrapped.WriteText(12, input)
	wrapped.CellFormat(100, 12, input, "", 0, "", false, 0, "")
	wrapped.WriteExternalLink(12, input, "https://example.com")
	wrapped.WriteInternalLink(12, input, "anchor")
	wrapped.MeasureTextWidth(input)
	wrapped.SplitText(input, 100)

	for operation, values := range map[string][]string{
		"WriteText":         recorder.texts,
		"CellFormat":        recorder.cells,
		"WriteExternalLink": {recorder.externalLinks[0].text},
		"WriteInternalLink": {recorder.internalLinks[0].text},
		"MeasureTextWidth":  recorder.measured,
		"SplitText":         recorder.split,
	} {
		for _, value := range values {
			if value != expected {
				t.Errorf("%s received %q, want normalized %q", operation, value, expected)
			}
		}
	}
	if strings.Contains(expected, "中") || strings.Contains(expected, "😀") || !strings.Contains(expected, "�") {
		t.Errorf("normalized text = %q", expected)
	}
	if normalized := wrapped.normalize("Grüße ß"); normalized != "Grüße ß" {
		t.Errorf("German text changed to %q", normalized)
	}
}

func TestGlyphFallbackQuestionMark(t *testing.T) {
	recorder := &recordingPDF{}
	wrapped := newGlyphPDF(recorder, map[fontKey]runeCoverage{
		{}: {'A': 1, '?': 2, '\t': 3},
	})
	wrapped.current = fontKey{}
	if got := wrapped.normalize("A中😀\n\t"); got != "A??\n\t" {
		t.Fatalf("normalize() = %q, want replacements with preserved layout controls", got)
	}
	withoutCoverage := newGlyphPDF(recorder, nil)
	if got := withoutCoverage.normalize("unchanged 中文"); got != "unchanged 中文" {
		t.Fatalf("normalize() without coverage = %q", got)
	}
	if err := wrapped.AddFont("broken", "", []byte("not a font")); err == nil {
		t.Fatal("AddFont(invalid) error = nil")
	}
	if _, err := parseFontCoverage([]byte("not a font")); err == nil {
		t.Fatal("parseFontCoverage(invalid) error = nil")
	}
}

func TestOfflineCacheAndLogWriter(t *testing.T) {
	cache, err := loadOfflineFonts()
	if err != nil {
		t.Fatal(err)
	}
	cache.Set("https://example.invalid/font.ttf", []byte("unexpected"))
	if value, hit := cache.Get("https://example.invalid/font.ttf"); !hit || value != nil {
		t.Fatal("unknown cache key did not fail closed with a nil offline hit")
	}
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	if _, err := (standardLogWriter{logger: logger}).Write([]byte("warning\n")); err != nil {
		t.Fatal(err)
	}
	if logs.String() != "warning\n" {
		t.Fatalf("log output = %q", logs.String())
	}
}

type panicOnWritePDF struct {
	*recordingPDF
}

func (pdf *panicOnWritePDF) Write(io.Writer) error {
	panic("deliberate test panic")
}

func TestConvertErrorsAndPanicRecovery(t *testing.T) {
	t.Run("writer error", func(t *testing.T) {
		converter, err := New(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		recorder := recordingConverter(t, converter)
		writeErr := errors.New("write failed")
		recorder.writeErr = writeErr
		if err := converter.Convert(context.Background(), []byte("body"), "guide.md", io.Discard); !errors.Is(err, writeErr) {
			t.Fatalf("Convert() error = %v, want wrapped write error", err)
		}
	})

	t.Run("panic", func(t *testing.T) {
		converter, err := New(nil, log.New(io.Discard, "", 0))
		if err != nil {
			t.Fatal(err)
		}
		recorder := &panicOnWritePDF{recordingPDF: &recordingPDF{}}
		converter.newBackend = func(context.Context, gopdf.Config, fonts.Cache) goldpdf.PDF {
			return recorder
		}
		err = converter.Convert(context.Background(), []byte("body"), "guide.md", io.Discard)
		if err == nil || !strings.Contains(err.Error(), "deliberate test panic") {
			t.Fatalf("Convert() panic error = %v", err)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		converter, err := New(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := converter.Convert(ctx, []byte("body"), "guide.md", io.Discard); !errors.Is(err, context.Canceled) {
			t.Fatalf("Convert() error = %v", err)
		}
	})

	t.Run("nil destination", func(t *testing.T) {
		converter, err := New(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := converter.Convert(context.Background(), nil, "guide.md", nil); err == nil {
			t.Fatal("Convert(nil destination) error = nil")
		}
	})

	t.Run("invalid document path", func(t *testing.T) {
		converter, err := New(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, documentPath := range []string{"", "../guide.md", "./guide.md", "docs//guide.md", "//guide.md", `docs\guide.md`, "guide\x00.md"} {
			if err := converter.Convert(context.Background(), nil, documentPath, io.Discard); err == nil {
				t.Errorf("Convert(document path %q) error = nil", documentPath)
			}
		}
	})
}
