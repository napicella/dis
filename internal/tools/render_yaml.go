package tools

// YAML for render-config: parse a destination, tell leaves, write literals.

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// parseYAML decodes with yaml.v3, which resolves aliases (and fails on one
// that contains itself).
func parseYAML(data []byte) (any, error) {
	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return stringKeys(v), nil
}

// isYAMLLeaf: what yaml.v3 decodes a YAML scalar to (int64 and uint64 for
// integers that don't fit an int).
func isYAMLLeaf(v any) bool {
	switch v.(type) {
	case string, bool, int, int64, uint64, float64, time.Time:
		return true
	}
	return false
}

// stringKeys makes every mapping a map[string]any, so paths can look into it.
// yaml.v3 decodes a mapping to map[any]any as soon as one of its keys is not a
// string (1: x, true: y); its keys become text here: 1 -> "1", true -> "true".
func stringKeys(v any) any {
	switch v := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(v))
		for k, x := range v {
			m[fmt.Sprint(k)] = stringKeys(x)
		}
		return m
	case map[string]any:
		for k, x := range v {
			v[k] = stringKeys(x)
		}
	case []any:
		for i, x := range v {
			v[i] = stringKeys(x)
		}
	}
	return v
}

// yamlLiteral writes a leaf, a flat array, or a template default with
// yaml.v3, on one line: an array in flow style, a string holding a newline
// double-quoted. Two values would not read back as yaml.v3 writes them: a
// whole float (1, an int) and a timestamp in a flow array (quoted, a string).
func yamlLiteral(v any) (string, error) {
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		return "", err
	}
	values, nodes := []any{v}, []*yaml.Node{&n}
	if items, ok := v.([]any); ok {
		n.Style = yaml.FlowStyle
		values, nodes = items, n.Content
	}
	for i, s := range nodes {
		_, isFloat := values[i].(float64)
		switch {
		case strings.Contains(s.Value, "\n"):
			s.Style = yaml.DoubleQuotedStyle
		case isFloat && s.ShortTag() == "!!int":
			s.Tag, s.Value = "!!float", s.Value+".0"
		case s.ShortTag() == "!!timestamp" && n.Kind == yaml.SequenceNode:
			s.Style = yaml.TaggedStyle
		}
	}
	b, err := yaml.Marshal(&n)
	return strings.TrimSuffix(string(b), "\n"), err
}
