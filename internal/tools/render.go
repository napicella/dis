package tools

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"unicode"
)

// How render-config works
//
// A config file that dis deploys can also be written by another tool (a theming
// tool sets the theme name or colors). Copying the file would reset those
// values, so the file in the configs folder is a Go template, and RenderConfig
// renders it to the destination, filling its keep and has actions from the
// destination's current content.
//
// Two rules shape the code:
//
//   - The destination is only read, to look values up. The output is always the
//     template's own text plus the values, never the destination re-encoded, so
//     the template's comments and layout survive, whatever the other tool did to
//     the file (yq, jq and friends drop comments and reformat).
//   - A value is written as a literal of the destination's format (JSON, TOML or
//     YAML), so it drops into the template exactly where a hand-written value
//     would go: `"theme": {{ keep "theme" "dark" }}` gives `"theme": "dark"`.
//     Only a leaf (string, number, bool, date/time) or a flat array of leaves
//     is kept: a table is kept key by key, its layout staying the template's.
//
// Each format's part (parsing a destination, which values are leaves, writing
// a literal) is in render_json.go, render_toml.go and render_yaml.go; the
// atomic write is in atomic.go.

// RenderConfig renders the text/template src to dest. The template can pull
// values from dest's current content with:
//
//   - keep PATH DEFAULT: the value at PATH in dest, or DEFAULT when dest does
//     not exist or has no value there, rendered as a literal in dest's format.
//     The value must be a leaf or a flat array of leaves, not a table.
//   - has PATH: whether dest has a value at PATH, for {{ if has PATH }}
//     blocks; false when dest does not exist or has no value there.
//
// PATH is dotted keys, a double-quoted key holding dots: see parsePath.
//
// The format comes from dest's extension (.json, .toml, .yaml, .yml). dest is
// parsed only when the template calls keep or has, so a template without
// them is a plain copy. dest is written atomically, keeping its file mode (0644
// for a new file); a dest that is a symlink, dangling or not, has its target
// written. On any error dest is left untouched.
func RenderConfig(src, dest string) error {
	text, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", src, err)
	}
	doc := &lazyDoc{load: sync.OnceValues(func() (*configDoc, error) { return parseConfigDoc(dest) })}
	// The template is named after src, so its errors read "src:LINE:COL: ...".
	tmpl, err := template.New(src).Funcs(template.FuncMap{
		"keep": doc.keep,
		"has":  doc.has,
	}).Parse(string(text))
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", src, err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, nil); err != nil {
		// text/template wraps a function's error in its own ("template: src:1:3:
		// executing ... error calling keep: ..."). A destination that can't be
		// parsed is not a template problem, so return that error on its own.
		if pe, ok := errors.AsType[*destParseError](err); ok {
			return pe
		}
		return fmt.Errorf("rendering %s to %s: %w", src, dest, err)
	}
	if err := WriteFileAtomic(dest, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}

// configDoc is the destination's content as generic values: map[string]any
// for a table, []any for an array, nil for no value (null included),
// anything else a leaf.
type configDoc struct {
	root any
	// isLeaf reports whether a decoded value is a leaf: each format's decoder
	// has its own leaf types (json.Number, int64, toml.LocalDate...).
	isLeaf func(any) bool
	// literal writes a leaf, a flat array of leaves, or a template default,
	// in the format, on one line.
	literal func(any) (string, error)
}

// get returns the value at path, nil when there is none.
func (d *configDoc) get(path []string) any {
	v := d.root
	for _, key := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[key]
	}
	return v
}

// lazyDoc is the destination file, parsed the first time a template function
// needs it.
type lazyDoc struct {
	load func() (*configDoc, error)
}

// get returns the document and the value at path, nil when there is none.
// The path is checked first, so a bad path fails even on a first install.
func (d *lazyDoc) get(path string) (*configDoc, any, error) {
	keys, err := parsePath(path)
	if err != nil {
		return nil, nil, err
	}
	doc, err := d.load()
	if err != nil {
		return nil, nil, err
	}
	return doc, doc.get(keys), nil
}

// keep implements the template's keep: the value at path in the destination,
// a leaf or a flat array of leaves, or def, written in the destination's
// format.
func (d *lazyDoc) keep(path string, def any) (string, error) {
	doc, v, err := d.get(path)
	if err != nil {
		return "", err
	}
	if v == nil {
		// Nothing at path: write the default given in the template.
		switch def.(type) {
		case string, bool, int, float64: // a template number constant is an int or a float64
			return doc.literal(def)
		default:
			return "", fmt.Errorf("DEFAULT must be a string, number or bool, not %T", def)
		}
	}

	// A value at path: write it, if it's a leaf or a flat array of leaves.
	if doc.isLeaf(v) {
		return doc.literal(v)
	}
	switch v := v.(type) {
	case map[string]any:
		return "", fmt.Errorf("keep %q: a table, not a value: keep each of its keys instead (%s.KEY)", path, path)
	case []any:
		for _, item := range v {
			if item != nil && !doc.isLeaf(item) { // null items are kept
				return "", fmt.Errorf("keep %q: an array holding a table or an array, not a flat array of values: write the array in the template instead", path)
			}
		}
		return doc.literal(v) // a flat array of leaves
	default:
		return "", fmt.Errorf("keep %q: an unsupported value (%T)", path, v)
	}
}

// has implements the template's has: whether there is a value at path, of
// any kind, a table included.
func (d *lazyDoc) has(path string) (bool, error) {
	_, v, err := d.get(path)
	return v != nil, err
}

// parsePath splits a keep or has PATH into keys. "." separates keys; a
// key in double quotes is taken literally, dots included, with \" and \\ as
// escapes, as in TOML dotted keys: "[python]"."editor.tabSize". An unquoted
// key can't hold whitespace, so "x. y" fails rather than look up " y"; a
// quote anywhere but at the start of a key is a plain character.
func parsePath(path string) ([]string, error) {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("invalid path %q: %s", path, fmt.Sprintf(format, a...))
	}
	// Each iteration reads one key, starting at path[i], and the "." after it.
	var keys []string
	for i := 0; ; {
		var key strings.Builder
		if i < len(path) && path[i] == '"' {
			// Quoted key: up to the closing quote, unescaping \" and \\.
			i++
			for {
				if i >= len(path) {
					return nil, bad("unterminated quoted key")
				}
				c := path[i]
				if c == '"' {
					i++
					break
				}
				if c == '\\' {
					if i+1 >= len(path) {
						return nil, bad("unterminated quoted key")
					}
					if e := path[i+1]; e != '"' && e != '\\' {
						return nil, bad(`unsupported escape \%c in a quoted key: only \" and \\ are escapes`, e)
					}
					c = path[i+1]
					i++
				}
				key.WriteByte(c)
				i++
			}
			if i < len(path) && path[i] != '.' {
				return nil, bad(`%q after a quoted key, want "." or the end of the path`, path[i:])
			}
		} else {
			// Bare key: up to the next "." or the end.
			end := strings.IndexByte(path[i:], '.')
			if end < 0 {
				end = len(path) - i
			}
			if end == 0 {
				return nil, bad(`empty key (a leading, trailing or doubled ".")`)
			}
			if k := path[i : i+end]; strings.IndexFunc(k, unicode.IsSpace) >= 0 {
				return nil, bad(`whitespace in unquoted key %q (put the key in double quotes if the whitespace is part of it)`, k)
			}
			key.WriteString(path[i : i+end])
			i += end
		}
		keys = append(keys, key.String())
		if i == len(path) {
			return keys, nil
		}
		i++ // the "."; after a trailing one, the next key fails as empty
	}
}

// parseConfigDoc reads and parses path according to its extension. A missing
// file gives an empty document, whose defaults are still written in the
// format. A parse error comes back as a destParseError, which says how to
// recover.
func parseConfigDoc(path string) (*configDoc, error) {
	var parse func([]byte) (any, error)
	doc := &configDoc{}
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".json":
		parse, doc.isLeaf, doc.literal = parseJSON, isJSONLeaf, jsonLiteral
	case ".toml":
		parse, doc.isLeaf, doc.literal = parseTOML, isTOMLLeaf, tomlLiteral
	case ".yaml", ".yml":
		parse, doc.isLeaf, doc.literal = parseYAML, isYAMLLeaf, yamlLiteral
	default:
		return nil, fmt.Errorf("can't keep values from %s: unsupported extension %q, want .json, .toml, .yaml or .yml", path, ext)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if doc.root, err = parse(data); err != nil {
		return nil, &destParseError{path: path, pkg: os.Getenv("DIS_PACKAGE"), err: err}
	}
	return doc, nil
}

// destParseError is a DEST that can't be parsed. RenderConfig returns it as
// is, not inside the template's error, so the fix comes first.
type destParseError struct {
	path, pkg string // pkg: the installer's package, if any
	err       error
}

func (e *destParseError) Error() string {
	rerun := "dis config"
	if e.pkg != "" {
		rerun += " for " + e.pkg
	}
	return fmt.Sprintf("cannot parse %s (%v): fix or delete it, then re-run %s", e.path, e.err, rerun)
}

func (e *destParseError) Unwrap() error { return e.err }
