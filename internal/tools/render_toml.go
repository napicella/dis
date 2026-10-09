package tools

// TOML for render-config: parse a destination, tell leaves, write literals.

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// parseTOML decodes with go-toml's stable API. Values come out as string,
// bool, int64, float64, time.Time (offset date-times), toml.LocalDate,
// LocalTime and LocalDateTime, []any, and map[string]any for every table.
func parseTOML(data []byte) (any, error) {
	var v map[string]any
	if err := toml.Unmarshal(data, &v); err != nil {
		if de, ok := errors.AsType[*toml.DecodeError](err); ok {
			row, col := de.Position()
			return nil, fmt.Errorf("line %d, column %d: %w", row, col, err)
		}
		return nil, err
	}
	return v, nil
}

// isTOMLLeaf: what go-toml decodes a TOML scalar to.
func isTOMLLeaf(v any) bool {
	switch v.(type) {
	case string, bool, int64, float64, time.Time, toml.LocalDate, toml.LocalTime, toml.LocalDateTime:
		return true
	}
	return false
}

// tomlLiteral writes a leaf, a flat array, or a template default, with
// go-toml. It only marshals whole documents, so v goes in as "v = ..." and
// comes back out.
func tomlLiteral(v any) (string, error) {
	b, err := toml.Marshal(map[string]any{"v": v})
	if err != nil {
		return "", err
	}
	lit, ok := strings.CutPrefix(string(b), "v = ")
	if !ok {
		return "", fmt.Errorf("unexpected TOML for %T: %q", v, b)
	}
	return strings.TrimSuffix(lit, "\n"), nil
}
