package web

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// Front matter has no schema, so the tree is bounded by structure only.
	maxFrontMatterDepth = 32
	maxFrontMatterNodes = 2000
	maxFrontMatterBytes = 1 << 20 // Larger blocks remain available as source.
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
// keys or value types. It reports a zero entry count when the source cannot be shown as a
// tree, in which case callers fall back to the original text.
func parseFrontMatter(source string) ([]frontMatterNode, int) {
	if len(source) > maxFrontMatterBytes {
		return nil, 0
	}
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(source), &document); err != nil {
		return nil, 0
	}
	keys := keySet{ids: make(map[string]int), memo: make(map[*yaml.Node]int)}
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 || !keys.uniqueMappingKeys(&document) {
		return nil, 0
	}
	builder := &frontMatterBuilder{remaining: maxFrontMatterNodes}
	root := builder.convert("", document.Content[0], 0, false)
	if root.Composite {
		count := len(document.Content[0].Content)
		if document.Content[0].Kind == yaml.MappingNode {
			count /= 2
		}
		return root.Children, count
	}
	if root.Empty {
		return nil, 0
	}
	return []frontMatterNode{root}, 1
}

func (builder *frontMatterBuilder) convert(key string, node *yaml.Node, depth int, siblingsPending bool) frontMatterNode {
	builder.remaining--
	if (builder.remaining == 0 && (siblingsPending || len(node.Content) > 0)) || depth > maxFrontMatterDepth {
		return frontMatterNode{Key: key, Value: frontMatterElided, Empty: true}
	}
	switch node.Kind {
	case yaml.SequenceNode:
		if len(node.Content) == 0 {
			return frontMatterNode{Key: key, Value: "empty list", Empty: true}
		}
		children := make([]frontMatterNode, 0, min(len(node.Content), builder.remaining, 32))
		for index, item := range node.Content {
			if builder.remaining == 0 {
				break
			}
			children = append(children, builder.convert(strconv.Itoa(index+1), item, depth+1, siblingsPending || index+1 < len(node.Content)))
		}
		return frontMatterNode{Key: key, Summary: countLabel("list", len(node.Content), "item", "items"), Composite: true, Children: children}
	case yaml.MappingNode:
		if len(node.Content) < 2 {
			return frontMatterNode{Key: key, Value: "empty map", Empty: true}
		}
		children := make([]frontMatterNode, 0, min(len(node.Content)/2, builder.remaining, 32))
		for index := 0; index+1 < len(node.Content); index += 2 {
			if builder.remaining == 0 {
				break
			}
			children = append(children, builder.convert(mappingKey(node.Content[index]), node.Content[index+1], depth+1, siblingsPending || index+2 < len(node.Content)))
		}
		return frontMatterNode{Key: key, Summary: countLabel("map", len(node.Content)/2, "key", "keys"), Composite: true, Children: children}
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

// yaml.Unmarshal into a Node preserves duplicate keys. Check every mapping,
// including ones outside the display budget, without decoding alias values.
func (keys keySet) uniqueMappingKeys(n *yaml.Node) bool {
	if n.Kind == yaml.MappingNode && len(n.Content) > 2 {
		seen := make(map[int]bool)
		for i := 0; i < len(n.Content); i += 2 {
			id := keys.id(n.Content[i])
			if id == 0 || seen[id] {
				return false
			}
			seen[id] = true
		}
	}
	for _, child := range n.Content {
		if !keys.uniqueMappingKeys(child) {
			return false
		}
	}
	return true
}

// Intern key structures, so aliases compare by target without repeatedly
// expanding them. Map order and scalar quoting don't distinguish YAML keys.
type keySet struct {
	ids  map[string]int
	memo map[*yaml.Node]int
}

func (s keySet) id(n *yaml.Node) int {
	if id, ok := s.memo[n]; ok {
		return id
	}
	s.memo[n] = 0 // Recursive key: use the source fallback.
	if n.Kind == yaml.AliasNode {
		id := s.id(n.Alias)
		s.memo[n] = id
		return id
	}
	var value string
	if n.Kind == yaml.ScalarNode {
		value = n.Value
		if n.ShortTag() != "!!str" {
			var scalar any
			if err := n.Decode(&scalar); err != nil {
				return 0
			}
			value = fmt.Sprint(scalar)
		}
	} else {
		parts := make([]string, 0, len(n.Content))
		for i := 0; i < len(n.Content); i++ {
			id := s.id(n.Content[i])
			if id == 0 {
				return 0
			}
			part := strconv.Itoa(id)
			if n.Kind == yaml.MappingNode {
				i++
				v := s.id(n.Content[i])
				if v == 0 {
					return 0
				}
				part += ":" + strconv.Itoa(v)
			}
			parts = append(parts, part)
		}
		if n.Kind == yaml.MappingNode {
			slices.Sort(parts)
		}
		value = strings.Join(parts, ",")
	}
	sig := strconv.Itoa(int(n.Kind)) + ":" + strconv.Quote(n.ShortTag()) + ":" + value
	id, ok := s.ids[sig]
	if !ok {
		id = len(s.ids) + 1
		s.ids[sig] = id
	}
	s.memo[n] = id
	return id
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
func frontMatterCount(count int) string {
	if count == 1 {
		return "1 field"
	}
	return strconv.Itoa(count) + " fields"
}
