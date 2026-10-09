package tools

// JSON for render-config: parse a destination, tell leaves, write literals.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// parseJSON decodes numbers as json.Number, so a number is written back as
// in the file. json.Unmarshal checks the syntax first: the Decoder misses
// trailing data and reports truncated input without a position.
func parseJSON(data []byte) (any, error) {
	if err := json.Unmarshal(data, new(json.RawMessage)); err != nil {
		if se, ok := errors.AsType[*json.SyntaxError](err); ok {
			// encoding/json gives a byte offset; a line number helps more.
			line := 1 + bytes.Count(data[:se.Offset], []byte("\n"))
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}

// isJSONLeaf: what encoding/json decodes a JSON scalar to (numbers are
// json.Number with UseNumber).
func isJSONLeaf(v any) bool {
	switch v.(type) {
	case string, bool, json.Number:
		return true
	}
	return false
}

// jsonLiteral marshals a leaf, a flat array, or a template default, without
// escaping <, > and &.
func jsonLiteral(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}
