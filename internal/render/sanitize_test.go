package render

import (
	"strings"
	"testing"
)

func TestRenderSanitizerPreservesSafeGFMMarkup(t *testing.T) {
	source := []byte(`# Safe Heading

| Feature | Status |
| --- | --- |
| Tables | Safe |

- [x] Complete
- [ ] Pending

![Diagram](images/diagram.png "Architecture")

[Guide](../guide.md)

` + "```go\nfmt.Println(\"safe\")\n```\n")
	document, err := Render(source, "docs/current.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<h1 id="safe-heading">Safe Heading</h1>`,
		"<table>", "<th>Feature</th>", "<td>Tables</td>",
		`type="checkbox"`, `disabled=""`, `checked=""`,
		`src="/docs/images/diagram.png"`, `alt="Diagram"`, `title="Architecture"`,
		`href="/guide.md"`,
		`<code class="language-go">`,
	} {
		if !strings.Contains(document.HTML, want) {
			t.Errorf("sanitized HTML does not preserve %q:\n%s", want, document.HTML)
		}
	}
}

func TestRenderSanitizerRemovesActiveContent(t *testing.T) {
	source := []byte(`# Security

<script>alert(1)</script>

<img src="x" onerror="alert(2)">

<iframe src="https://example.com"></iframe>

[JavaScript](javascript:alert(3))

[Data](data:text/html;base64,PHNjcmlwdD5hbGVydCg0KTwvc2NyaXB0Pg==)

![Unsafe image](javascript:alert(5))
`)
	document, err := Render(source, "security.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{
		"<script", "alert(1)", "onerror", "alert(2)", "<iframe",
		"javascript:", "alert(3)", "data:text/html", "PHNjcmlwdD", "alert(5)",
	} {
		if strings.Contains(document.HTML, unwanted) {
			t.Errorf("sanitized HTML contains active content %q:\n%s", unwanted, document.HTML)
		}
	}
	if !strings.Contains(document.HTML, `<h1 id="security">Security</h1>`) {
		t.Fatalf("safe heading was not retained:\n%s", document.HTML)
	}
}

func TestSanitizerRejectsNonCheckboxInputsAndEventHandlers(t *testing.T) {
	input := `<input type="text" value="secret"><input type="checkbox" disabled="" onclick="alert(1)"><button>Go</button>`
	output := NewSanitizer().Sanitize(input)
	for _, unwanted := range []string{"text", "secret", "onclick", "alert", "<button"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("Sanitize() output contains %q: %s", unwanted, output)
		}
	}
	if !strings.Contains(output, `<input type="checkbox" disabled="">`) {
		t.Fatalf("disabled checkbox was not preserved: %s", output)
	}
}
