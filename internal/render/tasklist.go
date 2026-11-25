package render

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type taskCheckBoxRenderer struct{}

func newTaskCheckBoxRenderer() renderer.NodeRenderer {
	return &taskCheckBoxRenderer{}
}

func (renderer *taskCheckBoxRenderer) RegisterFuncs(registrar renderer.NodeRendererFuncRegisterer) {
	registrar.Register(extensionast.KindTaskCheckBox, renderer.render)
}

func (renderer *taskCheckBoxRenderer) render(
	writer util.BufWriter,
	source []byte,
	node ast.Node,
	entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	checkbox, ok := node.(*extensionast.TaskCheckBox)
	if !ok {
		return ast.WalkStop, fmt.Errorf("unexpected task checkbox node %T", node)
	}
	var output strings.Builder
	if checkbox.IsChecked {
		output.WriteString(`<input checked="" disabled="" type="checkbox"`)
	} else {
		output.WriteString(`<input disabled="" type="checkbox"`)
	}
	label := strings.Join(strings.Fields(plainText(node.Parent(), source)), " ")
	if chars := []rune(label); len(chars) > 500 {
		label = string(chars[:499]) + "…"
	}
	if label == "" {
		label = "Task"
	}
	output.WriteString(` aria-label="`)
	_, _ = output.Write(util.EscapeHTML([]byte(label)))
	output.WriteString(`"> `)
	if _, err := writer.WriteString(output.String()); err != nil {
		return ast.WalkStop, fmt.Errorf("write task checkbox: %w", err)
	}
	return ast.WalkContinue, nil
}

func plainText(node ast.Node, source []byte) string {
	var output strings.Builder
	appendInlineText(&output, node, source)
	return strings.TrimSpace(output.String())
}
