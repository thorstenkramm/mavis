package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestIsMarkdown(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "guide.md", want: true},
		{name: "guide.markdown", want: true},
		{name: "GUIDE.MD", want: true},
		{name: "GUIDE.MARKDOWN", want: true},
		{name: "Guide.Markdown", want: true},
		{name: "guide.md.txt", want: false},
		{name: "markdown", want: false},
		{name: "image.png", want: false},
		{name: "", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsMarkdown(test.name); got != test.want {
				t.Fatalf("IsMarkdown(%q) = %v, want %v", test.name, got, test.want)
			}
		})
	}
}

func TestCoreGFMRendering(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "table",
			source: "| Feature | Status |\n| --- | --- |\n| Tables | Core |\n",
			want:   []string{"<table>", "<th>Feature</th>", "<td>Tables</td>", "</table>"},
		},
		{
			name:   "task lists",
			source: "- [ ] Pending\n- [x] Complete\n",
			want:   []string{`type="checkbox"`, `disabled=""`, `checked=""`, "Pending", "Complete"},
		},
		{
			name:   "strikethrough",
			source: "Keep ~~removed~~ this.\n",
			want:   []string{"Keep <del>removed</del> this."},
		},
		{
			name:   "linkification",
			source: "Visit https://example.com/docs for details.\n",
			want:   []string{`<a href="https://example.com/docs">https://example.com/docs</a>`},
		},
		{
			name:   "fenced code",
			source: "```go\nfmt.Println(\"hello\")\n```\n",
			want:   []string{`<pre><code class="language-go">`, "fmt.Println(&quot;hello&quot;)", "</code></pre>"},
		},
		{
			name:   "standard Markdown",
			source: "# Heading\n\n- one\n- two\n\n[Guide](guide.md) and **bold** text.\n",
			want:   []string{`<h1 id="heading">Heading</h1>`, "<ul>", "<li>one</li>", `<a href="guide.md">Guide</a>`, "<strong>bold</strong>"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := convertMarkdown(t, test.source)
			for _, want := range test.want {
				if !strings.Contains(output, want) {
					t.Errorf("output does not contain %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestRawHTMLAndDangerousURLsAreSafeByDefault(t *testing.T) {
	output := convertMarkdown(t, "<script>alert(1)</script>\n\nClick <span onclick=\"alert(2)\">here</span> or [bad](javascript:alert(3)).\n")
	for _, unwanted := range []string{"<script", "alert(1)", "onclick", "javascript:"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("output contains unsafe markup %q:\n%s", unwanted, output)
		}
	}
	if !strings.Contains(output, "raw HTML omitted") {
		t.Fatalf("raw HTML was not explicitly omitted:\n%s", output)
	}
}

func convertMarkdown(t *testing.T, source string) string {
	t.Helper()
	var output bytes.Buffer
	if err := NewMarkdown().Convert([]byte(source), &output); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	return output.String()
}
