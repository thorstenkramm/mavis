package web

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var cssColorPattern = regexp.MustCompile(`--([a-z-]+):\s*(#[0-9a-fA-F]{6});`)

const darkThemeSelector = `:root[data-theme="dark"] {`

// themeColors returns the custom properties of the light and dark palettes.
func themeColors(t *testing.T) map[string]map[string]string {
	t.Helper()
	stylesheet := readEmbeddedText(t, "static/app.css")
	darkStart := strings.Index(stylesheet, darkThemeSelector)
	if darkStart < 0 {
		t.Fatal("stylesheet has no dark theme")
	}
	darkEnd := darkStart + strings.Index(stylesheet[darkStart:], "}")
	palettes := map[string]map[string]string{"light": {}, "dark": {}}
	for name, block := range map[string]string{"light": stylesheet[:darkStart], "dark": stylesheet[darkStart:darkEnd]} {
		for _, match := range cssColorPattern.FindAllStringSubmatch(block, -1) {
			palettes[name][match[1]] = match[2]
		}
	}
	return palettes
}

func TestWCAGAAColorContrast(t *testing.T) {
	tests := []struct {
		name       string
		foreground string
		background string
		minimum    float64
	}{
		{name: "body text on page", foreground: "text", background: "background", minimum: 4.5},
		{name: "body text on sidebar", foreground: "text", background: "sidebar", minimum: 4.5},
		{name: "muted text on page", foreground: "muted", background: "background", minimum: 4.5},
		{name: "muted text on sidebar", foreground: "muted", background: "sidebar", minimum: 4.5},
		{name: "links on page", foreground: "accent", background: "background", minimum: 4.5},
		{name: "links on sidebar", foreground: "accent", background: "sidebar", minimum: 4.5},
		{name: "current page", foreground: "accent-dark", background: "accent-soft", minimum: 4.5},
		{name: "search highlight", foreground: "mark-text", background: "mark", minimum: 4.5},
		{name: "hovered row", foreground: "text", background: "hover", minimum: 4.5},
		{name: "code", foreground: "text", background: "code", minimum: 4.5},
		{name: "skip link", foreground: "background", background: "text", minimum: 4.5},
		{name: "focus on page", foreground: "focus", background: "background", minimum: 3},
		{name: "focus on sidebar", foreground: "focus", background: "sidebar", minimum: 3},
		{name: "focus on current page", foreground: "focus", background: "accent-soft", minimum: 3},
		{name: "control border on page", foreground: "control-border", background: "background", minimum: 3},
		{name: "control border on sidebar", foreground: "control-border", background: "sidebar", minimum: 3},
	}
	for theme, colors := range themeColors(t) {
		for _, test := range tests {
			t.Run(theme+"/"+test.name, func(t *testing.T) {
				foreground, background := colors[test.foreground], colors[test.background]
				if foreground == "" || background == "" {
					t.Fatalf("missing CSS colors for %q and %q", test.foreground, test.background)
				}
				if ratio := contrastRatio(t, foreground, background); ratio < test.minimum {
					t.Fatalf("contrast ratio = %.2f:1, want at least %.1f:1 (%s on %s)", ratio, test.minimum, foreground, background)
				}
			})
		}
	}
}

func contrastRatio(t *testing.T, foreground, background string) float64 {
	t.Helper()
	first := relativeLuminance(t, foreground)
	second := relativeLuminance(t, background)
	if first < second {
		first, second = second, first
	}
	return (first + 0.05) / (second + 0.05)
}

func relativeLuminance(t *testing.T, color string) float64 {
	t.Helper()
	value, err := strconv.ParseUint(color[1:], 16, 32)
	if err != nil {
		t.Fatalf("parse CSS color %q: %v", color, err)
	}
	channels := []float64{
		float64((value>>16)&0xff) / 255,
		float64((value>>8)&0xff) / 255,
		float64(value&0xff) / 255,
	}
	for index, channel := range channels {
		if channel <= 0.04045 {
			channels[index] = channel / 12.92
		} else {
			channels[index] = math.Pow((channel+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}
