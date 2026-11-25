package render

import (
	"strings"
	"testing"
)

func TestTaskCheckboxesHaveAccessibleLabels(t *testing.T) {
	document, err := Render(
		[]byte("- [ ] Serve **documentation**\n- [x] Complete `setup`\n- [ ] <span>Unsafe</span>\n- [ ]\n"),
		"tasks.md",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<input disabled="" type="checkbox" aria-label="Serve documentation">`,
		`<input checked="" disabled="" type="checkbox" aria-label="Complete setup">`,
		`<input disabled="" type="checkbox" aria-label="Unsafe">`,
		`<input disabled="" type="checkbox" aria-label="Task">`,
	} {
		if !strings.Contains(document.HTML, want) {
			t.Errorf("rendered task list does not contain %q:\n%s", want, document.HTML)
		}
	}
	if strings.Contains(document.HTML, "<span") {
		t.Fatalf("raw HTML in a task label was retained:\n%s", document.HTML)
	}
}

func TestSanitizerPreservesGeneratedTaskLabels(t *testing.T) {
	input := `<input disabled="" type="checkbox" aria-label="Read the &quot;guide&quot;">`
	output := NewSanitizer().Sanitize(input)
	if !strings.Contains(output, `aria-label="Read the &#34;guide&#34;"`) {
		t.Fatalf("Sanitize() removed the accessible task label: %s", output)
	}
}
