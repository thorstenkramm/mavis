package web

import (
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	// Front matter has no schema, so the tree is bounded by structure only.
	maxFrontMatterDepth = 32
	maxFrontMatterNodes = 2000
	frontMatterElided   = "…"
)

// frontMatterNode is one entry of the schema-free front-matter tree. Scalars
// carry Value; lists and maps carry Summary and Children.
type frontMatterNode struct {
	Key       string
	Value     string
	Summary   string
	Empty     bool
	Composite bool
	Children  []frontMatterNode
}

type frontMatterBuilder struct {
	remaining int
}

// parseFrontMatter turns YAML source into a display tree without assuming any
// keys or value types. It reports false when the source cannot be shown as a
// tree, in which case callers fall back to the original text.
func parseFrontMatter(source string) ([]frontMatterNode, bool) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(source), &document); err != nil {
		return nil, false
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return nil, false
	}
	builder := &frontMatterBuilder{remaining: maxFrontMatterNodes}
	root := builder.convert("", document.Content[0], 0)
	if root.Composite {
		return root.Children, true
	}
	if root.Empty {
		return nil, false
	}
	return []frontMatterNode{root}, true
}

func (builder *frontMatterBuilder) convert(key string, node *yaml.Node, depth int) frontMatterNode {
	builder.remaining--
	if builder.remaining < 0 || depth > maxFrontMatterDepth {
		return frontMatterNode{Key: key, Value: frontMatterElided, Empty: true}
	}
	switch node.Kind {
	case yaml.SequenceNode:
		if len(node.Content) == 0 {
			return frontMatterNode{Key: key, Value: "empty list", Empty: true}
		}
		children := make([]frontMatterNode, 0, len(node.Content))
		for index, item := range node.Content {
			children = append(children, builder.convert(strconv.Itoa(index+1), item, depth+1))
		}
		return frontMatterNode{Key: key, Summary: countLabel("list", len(children), "item", "items"), Composite: true, Children: children}
	case yaml.MappingNode:
		if len(node.Content) < 2 {
			return frontMatterNode{Key: key, Value: "empty map", Empty: true}
		}
		children := make([]frontMatterNode, 0, len(node.Content)/2)
		for index := 0; index+1 < len(node.Content); index += 2 {
			children = append(children, builder.convert(mappingKey(node.Content[index]), node.Content[index+1], depth+1))
		}
		return frontMatterNode{Key: key, Summary: countLabel("map", len(children), "key", "keys"), Composite: true, Children: children}
	case yaml.AliasNode:
		// Aliases are named, never expanded, so repeated anchors cannot grow the tree.
		return frontMatterNode{Key: key, Value: "*" + node.Value}
	default:
		if node.Tag == "!!null" || node.Value == "" {
			return frontMatterNode{Key: key, Value: "empty", Empty: true}
		}
		return frontMatterNode{Key: key, Value: node.Value}
	}
}

func mappingKey(node *yaml.Node) string {
	if node.Kind == yaml.ScalarNode {
		return node.Value
	}
	return "(complex key)"
}

func countLabel(kind string, count int, singular, plural string) string {
	unit := plural
	if count == 1 {
		unit = singular
	}
	return kind + " · " + strconv.Itoa(count) + " " + unit
}

// frontMatterCount labels the collapsed panel with its top-level entry count.
func frontMatterCount(nodes []frontMatterNode) string {
	if len(nodes) == 1 {
		return "1 field"
	}
	return strconv.Itoa(len(nodes)) + " fields"
}
