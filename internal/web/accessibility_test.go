package web

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var cssColorPattern = regexp.MustCompile(`--([a-z-]+):\s*(#[0-9a-fA-F]{6});`)

func TestWCAGAAColorContrast(t *testing.T) {
	colors := make(map[string]string)
	for _, match := range cssColorPattern.FindAllStringSubmatch(readEmbeddedText(t, "static/app.css"), -1) {
		colors[match[1]] = match[2]
	}
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
		{name: "skip link", foreground: "background", background: "text", minimum: 4.5},
		{name: "focus on page", foreground: "#005fcc", background: "background", minimum: 3},
		{name: "focus on sidebar", foreground: "#005fcc", background: "sidebar", minimum: 3},
		{name: "focus on current page", foreground: "#005fcc", background: "accent-soft", minimum: 3},
		{name: "control border on page", foreground: "control-border", background: "background", minimum: 3},
		{name: "control border on sidebar", foreground: "control-border", background: "sidebar", minimum: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			foreground := test.foreground
			if !strings.HasPrefix(foreground, "#") {
				foreground = colors[foreground]
			}
			background := colors[test.background]
			if foreground == "" || background == "" {
				t.Fatalf("missing CSS colors for %q and %q", test.foreground, test.background)
			}
			if ratio := contrastRatio(t, foreground, background); ratio < test.minimum {
				t.Fatalf("contrast ratio = %.2f:1, want at least %.1f:1 (%s on %s)", ratio, test.minimum, foreground, background)
			}
		})
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
