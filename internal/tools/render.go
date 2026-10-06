package tools

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// RenderConfig renders the text/template src to dest. The template can pull
// values from dest's current content with:
//
//   - keep PATH DEFAULT: the value at PATH in dest, or DEFAULT when dest does
//     not exist or has no value there, rendered as a literal in dest's format.
//   - keepTable PATH: (TOML only) the table at PATH in dest, rendered with its
//     header, other keys and subtables, each sorted by key, and preceded by a
//     blank line ("\n\n" first); empty when there is none.
//
// PATH is dotted keys, a double-quoted key holding dots: see parsePath.
//
// The format comes from dest's extension (.json, .toml, .yaml, .yml). dest is
// parsed only when the template calls keep or keepTable, so a template without
// them is a plain copy. dest is written atomically, keeping its file mode (0644
// for a new file); a dest that is a symlink, dangling or not, has its target
// written. On any error dest is left untouched.
func RenderConfig(src, dest string) error {
	text, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", src, err)
	}
	doc := &lazyDoc{path: dest}
	tmpl, err := template.New(src).Funcs(template.FuncMap{
		"keep":      doc.keep,
		"keepTable": doc.keepTable,
	}).Parse(string(text))
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", src, err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, nil); err != nil {
		if pe, ok := errors.AsType[*destParseError](err); ok {
			return pe
		}
		return fmt.Errorf("rendering %s to %s: %w", src, dest, err)
	}
	if err := writeFileAtomic(dest, out.Bytes()); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}

// writeFileAtomic writes data to a temp file next to path and renames it over
// path, creating the parent dirs. The file keeps path's mode, or gets 0644.
// When path is a symlink, the link's final target is written instead, even
// when it does not exist yet.
func writeFileAtomic(path string, data []byte) error {
	path, err := resolveLink(path)
	if err != nil {
		return err
	}
	mode := fs.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		mode = fi.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// maxLinks bounds the symlinks resolveLink follows, as the kernel does.
const maxLinks = 40

// resolveLink follows path while it is a symlink and returns the first path
// that is not one, which may not exist (a dangling link). A relative target
// is relative to the directory holding the link.
func resolveLink(path string) (string, error) {
	orig := path
	for range maxLinks + 1 {
		fi, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) || (err == nil && fi.Mode()&fs.ModeSymlink == 0) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			// Resolve the link's dir first, so ".." in target goes up from
			// where the link really is, as the kernel does.
			dir := filepath.Dir(path)
			if real, err := filepath.EvalSymlinks(dir); err == nil {
				dir = real
			}
			target = filepath.Join(dir, target)
		}
		path = target
	}
	return "", fmt.Errorf("%s: too many levels of symbolic links", orig)
}

// configDoc is the parsed content of a destination file.
type configDoc interface {
	// lookup returns the value at path rendered as a literal of the format,
	// or found false when there is none.
	lookup(path []string) (literal string, found bool, err error)
	// literal renders a template default as a literal of the format.
	literal(v any) (string, error)
}

// lazyDoc parses the destination file the first time a template function
// needs it.
type lazyDoc struct {
	path   string
	doc    configDoc
	exists bool
	err    error
	done   bool
}

func (d *lazyDoc) load() error {
	if d.done {
		return d.err
	}
	d.done = true
	d.doc, d.exists, d.err = parseConfigDoc(d.path)
	return d.err
}

func (d *lazyDoc) keep(path string, def any) (string, error) {
	keys, err := parsePath(path)
	if err != nil {
		return "", err
	}
	if err := d.load(); err != nil {
		return "", err
	}
	if d.exists {
		lit, found, err := d.doc.lookup(keys)
		if err != nil || found {
			return lit, err
		}
	}
	return d.doc.literal(def)
}

// keepTable returns the table preceded by a blank line, so that
// "last line\n{{- keepTable PATH }}\n\n[next]" leaves exactly one blank line
// on each side of the table, and between the two lines when there is none.
func (d *lazyDoc) keepTable(path string) (string, error) {
	if !strings.EqualFold(filepath.Ext(d.path), ".toml") {
		return "", fmt.Errorf("keepTable %q: %s is not a .toml file", path, d.path)
	}
	keys, err := parsePath(path)
	if err != nil {
		return "", err
	}
	if err := d.load(); err != nil {
		return "", err
	}
	if !d.exists {
		return "", nil
	}
	table, err := d.doc.(*tomlDoc).table(keys)
	if err != nil || table == "" {
		return "", err
	}
	return "\n\n" + table, nil
}

// parsePath splits a keep or keepTable PATH into keys. "." separates keys; a
// key in double quotes is taken literally, dots included, with \" and \\ as
// escapes, as in TOML dotted keys: "[python]"."editor.tabSize". An unquoted
// key can't hold whitespace, so "x. y" fails rather than look up " y"; a
// quote anywhere but at the start of a key is a plain character.
func parsePath(path string) ([]string, error) {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("invalid path %q: %s", path, fmt.Sprintf(format, a...))
	}
	var keys []string
	for i := 0; ; {
		var key strings.Builder
		if i < len(path) && path[i] == '"' {
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
		i++ // the "."
		if i == len(path) {
			return nil, bad(`empty key (a leading, trailing or doubled ".")`)
		}
	}
}

// parseConfigDoc reads and parses path according to its extension. A missing
// file gives an empty document of the right format and exists false.
func parseConfigDoc(path string) (doc configDoc, exists bool, err error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".toml", ".yaml", ".yml":
	default:
		return nil, false, fmt.Errorf("can't keep values from %s: unsupported extension %q, want .json, .toml, .yaml or .yml", path, ext)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		err = nil
	} else {
		exists = true
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading %s: %w", path, err)
	}
	switch ext {
	case ".json":
		doc, err = parseJSONDoc(data, exists)
	case ".toml":
		doc, err = parseTOMLDoc(data)
	default:
		doc, err = parseYAMLDoc(data)
	}
	if err != nil {
		return nil, false, &destParseError{path: path, pkg: os.Getenv("DIS_PACKAGE"), err: err}
	}
	return doc, exists, nil
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

// defaultValue checks that a template default is a string, number or bool.
func defaultValue(v any) (any, error) {
	switch v.(type) {
	case string, bool, int, int64, float64:
		return v, nil
	}
	return nil, fmt.Errorf("DEFAULT must be a string, number or bool, not %T", v)
}

// ── JSON ─────────────────────────────────────────────────────────────────────

type jsonDoc struct{ data []byte }

func parseJSONDoc(data []byte, exists bool) (configDoc, error) {
	if exists {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			var se *json.SyntaxError
			if errors.As(err, &se) {
				line := 1 + bytes.Count(data[:se.Offset], []byte("\n"))
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			return nil, err
		}
	}
	return &jsonDoc{data: data}, nil
}

// lookup walks objects with RawMessage, so the value found keeps its key
// order and number format. null counts as no value.
func (d *jsonDoc) lookup(path []string) (string, bool, error) {
	raw := json.RawMessage(d.data)
	for _, key := range path {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return "", false, nil
		}
		var ok bool
		if raw, ok = obj[key]; !ok {
			return "", false, nil
		}
	}
	if string(bytes.TrimSpace(raw)) == "null" {
		return "", false, nil
	}
	var b bytes.Buffer
	if err := json.Compact(&b, raw); err != nil {
		return "", false, err
	}
	return b.String(), true, nil
}

func (d *jsonDoc) literal(v any) (string, error) {
	v, err := defaultValue(v)
	if err != nil {
		return "", err
	}
	return jsonLiteral(v)
}

// jsonLiteral marshals v without escaping <, > and &.
func jsonLiteral(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// ── YAML ─────────────────────────────────────────────────────────────────────

// yamlDoc renders values as flow YAML: strings double-quoted (JSON string
// syntax, which YAML accepts), other scalars as written, collections as flow
// collections in the file's order.
type yamlDoc struct{ root *yaml.Node }

func parseYAMLDoc(data []byte) (configDoc, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	d := &yamlDoc{}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		d.root = doc.Content[0]
	}
	return d, nil
}

func resolveAlias(n *yaml.Node) *yaml.Node {
	for n != nil && n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n
}

func (d *yamlDoc) lookup(path []string) (string, bool, error) {
	n := resolveAlias(d.root)
	for _, key := range path {
		if n == nil || n.Kind != yaml.MappingNode {
			return "", false, nil
		}
		var next *yaml.Node
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				next = n.Content[i+1]
			}
		}
		n = resolveAlias(next)
	}
	if n == nil || (n.Kind == yaml.ScalarNode && n.ShortTag() == "!!null") {
		return "", false, nil
	}
	lit, err := yamlLiteral(n)
	return lit, err == nil, err
}

func yamlLiteral(n *yaml.Node) (string, error) {
	n = resolveAlias(n)
	switch n.Kind {
	case yaml.ScalarNode:
		switch tag := n.ShortTag(); tag {
		case "!!int", "!!float", "!!bool", "!!null", "!!timestamp":
			return n.Value, nil
		case "!!str":
			return jsonLiteral(n.Value)
		default:
			s, err := jsonLiteral(n.Value)
			return n.Tag + " " + s, err
		}
	case yaml.SequenceNode:
		items := make([]string, 0, len(n.Content))
		for _, c := range n.Content {
			s, err := yamlLiteral(c)
			if err != nil {
				return "", err
			}
			items = append(items, s)
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	case yaml.MappingNode:
		items := make([]string, 0, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, err := yamlLiteral(n.Content[i])
			if err != nil {
				return "", err
			}
			v, err := yamlLiteral(n.Content[i+1])
			if err != nil {
				return "", err
			}
			items = append(items, k+": "+v)
		}
		return "{" + strings.Join(items, ", ") + "}", nil
	}
	return "", fmt.Errorf("unsupported YAML node kind %d", n.Kind)
}

func (d *yamlDoc) literal(v any) (string, error) {
	v, err := defaultValue(v)
	if err != nil {
		return "", err
	}
	return jsonLiteral(v)
}

// ── TOML ─────────────────────────────────────────────────────────────────────

// tomlDoc holds the decoded values. TOML tables are unordered and the decoded
// map does not say which tables were inline or arrays of tables, so values are
// written with sorted keys, tables as inline tables in keep and as [headers]
// in keepTable, arrays of tables as inline arrays.
type tomlDoc struct{ values map[string]any }

func parseTOMLDoc(data []byte) (configDoc, error) {
	d := &tomlDoc{values: map[string]any{}}
	if err := toml.Unmarshal(data, &d.values); err != nil {
		var de *toml.DecodeError
		if errors.As(err, &de) {
			row, col := de.Position()
			return nil, fmt.Errorf("line %d, column %d: %w", row, col, err)
		}
		return nil, err
	}
	return d, nil
}

func (d *tomlDoc) get(path []string) (any, bool) {
	var v any = d.values
	for _, key := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		if v, ok = m[key]; !ok {
			return nil, false
		}
	}
	return v, true
}

func (d *tomlDoc) lookup(path []string) (string, bool, error) {
	v, ok := d.get(path)
	if !ok {
		return "", false, nil
	}
	lit, err := tomlLiteral(v)
	return lit, err == nil, err
}

func (d *tomlDoc) literal(v any) (string, error) {
	v, err := defaultValue(v)
	if err != nil {
		return "", err
	}
	return tomlLiteral(v)
}

// tomlLiteral renders v as a TOML value; maps become inline tables with
// sorted keys.
func tomlLiteral(v any) (string, error) {
	switch v := v.(type) {
	case string:
		return tomlString(v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return tomlFloat(v), nil
	case time.Time:
		return v.Format(time.RFC3339Nano), nil
	case encoding.TextMarshaler: // toml.LocalDate, LocalTime, LocalDateTime
		b, err := v.MarshalText()
		return string(b), err
	case []any:
		items := make([]string, 0, len(v))
		for _, item := range v {
			s, err := tomlLiteral(item)
			if err != nil {
				return "", err
			}
			items = append(items, s)
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	case map[string]any:
		if len(v) == 0 {
			return "{}", nil
		}
		items := make([]string, 0, len(v))
		for _, k := range slices.Sorted(maps.Keys(v)) {
			s, err := tomlLiteral(v[k])
			if err != nil {
				return "", err
			}
			items = append(items, tomlKey(k)+" = "+s)
		}
		return "{ " + strings.Join(items, ", ") + " }", nil
	}
	return "", fmt.Errorf("unsupported TOML value %T", v)
}

// isTableArray reports whether v is an array of tables ([[x]] or [{...}, ...]):
// the stable decoder gives the same value for both forms.
func isTableArray(v any) bool {
	switch a := v.(type) {
	case []map[string]any:
		return len(a) > 0
	case []any:
		for _, e := range a {
			if _, ok := e.(map[string]any); !ok {
				return false
			}
		}
		return len(a) > 0
	}
	return false
}

// table renders the table at path: its header and its other values, then its
// subtables, recursively, each in sorted key order.
func (d *tomlDoc) table(path []string) (string, error) {
	v, ok := d.get(path)
	if !ok {
		return "", nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		if isTableArray(v) {
			return "", fmt.Errorf("keepTable %q: an array of tables, not a table; use keep", dottedKey(path))
		}
		return "", fmt.Errorf("keepTable %q: not a table", dottedKey(path))
	}
	var blocks []string
	if err := writeTable(&blocks, path, m, true); err != nil {
		return "", fmt.Errorf("keepTable %q: %w", dottedKey(path), err)
	}
	return strings.Join(blocks, "\n\n"), nil
}

func writeTable(blocks *[]string, path []string, m map[string]any, top bool) error {
	var lines, subtables []string
	for _, k := range slices.Sorted(maps.Keys(m)) {
		if _, isMap := m[k].(map[string]any); isMap {
			subtables = append(subtables, k)
			continue
		}
		lit, err := tomlLiteral(m[k])
		if err != nil {
			return err
		}
		lines = append(lines, tomlKey(k)+" = "+lit)
	}
	// A table holding only subtables needs no header of its own, as in the
	// usual TOML style; the requested table always gets one.
	if top || len(lines) > 0 || len(subtables) == 0 {
		*blocks = append(*blocks, strings.Join(append([]string{"[" + dottedKey(path) + "]"}, lines...), "\n"))
	}
	for _, k := range subtables {
		if err := writeTable(blocks, append(slices.Clone(path), k), m[k].(map[string]any), false); err != nil {
			return err
		}
	}
	return nil
}

var bareKey = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// dottedKey renders path as a TOML dotted key, quoting the keys that need it.
func dottedKey(path []string) string {
	keys := make([]string, len(path))
	for i, k := range path {
		keys[i] = tomlKey(k)
	}
	return strings.Join(keys, ".")
}

func tomlKey(k string) string {
	if bareKey.MatchString(k) {
		return k
	}
	return tomlString(k)
}

// tomlString renders s as a TOML basic string.
func tomlString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func tomlFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}
