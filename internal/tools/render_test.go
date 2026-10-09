package tools

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// renderInDir writes the template, and dest unless it is nil, to a temp dir
// and renders them. It returns the template's and dest's paths.
func renderInDir(t *testing.T, destName string, dest *string, tmpl string) (src, destPath string, err error) {
	t.Helper()
	dir := t.TempDir()
	src = filepath.Join(dir, "template")
	if err := os.WriteFile(src, []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}
	destPath = filepath.Join(dir, "out", destName)
	if dest != nil {
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destPath, []byte(*dest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return src, destPath, RenderConfig(src, destPath)
}

func ptr(s string) *string { return &s }

const tomlDest = `mode = "custom"

[section]
key = "#ff0000"
"weird key" = 'it"s'
inline = { b = 1, a = 2 }
list = [1, 2]
z = true
a = 1.5

[section.deep]
when = 2024-01-02
`

func TestRenderConfig(t *testing.T) {
	tests := []struct {
		name     string
		destName string
		dest     *string // nil: DEST does not exist
		tmpl     string
		want     string
		wantErr  string
	}{
		// JSON
		{"json present string", "s.json", ptr(`{"mode": "custom"}`),
			`{"mode": {{ keep "mode" "dark" }}}`, `{"mode": "custom"}`, ""},
		{"json missing dest", "s.json", nil,
			`{"mode": {{ keep "mode" "dark" }}}`, `{"mode": "dark"}`, ""},
		{"json missing key", "s.json", ptr(`{"other": 1}`),
			`{{ keep "mode" "dark" }}`, `"dark"`, ""},
		{"json null is no value", "s.json", ptr(`{"a": null}`),
			`{{ keep "a" "x" }}`, `"x"`, ""},
		{"json kinds", "s.json", ptr(`{"a": {"b": 3, "c": true, "d": [1, "x" , null,2.50], "e": []}}`),
			`{{ keep "a.b" 0 }} {{ keep "a.c" false }} {{ keep "a.d" "" }} {{ keep "a.e" "" }}`,
			`3 true [1,"x",null,2.50] []`, ""},
		{"json object is not a value", "s.json", ptr(`{"a": {"b": 1}}`),
			`{{ keep "a" "" }}`, "", `keep "a": a table, not a value: keep each of its keys instead (a.KEY)`},
		{"json array of objects", "s.json", ptr(`{"a": [1, {"b": 1}]}`),
			`{{ keep "a" "" }}`, "", `keep "a": an array holding a table or an array, not a flat array of values: write the array in the template instead`},
		{"json nested array", "s.json", ptr(`{"a": [[1]]}`),
			`{{ keep "a" "" }}`, "", `an array holding a table or an array`},
		{"json path through non-object", "s.json", ptr(`{"a": "s"}`),
			`{{ keep "a.b" 1 }}`, `1`, ""},
		{"json bool and number defaults", "s.json", nil,
			`{{ keep "a" true }} {{ keep "b" 3 }} {{ keep "c" 2.5 }}`, `true 3 2.5`, ""},
		{"json flat key with dots", "s.json", ptr(`{"app.mode": "dark"}`),
			"{{ keep \"app.mode\" \"light\" }} {{ keep `\"app.mode\"` \"light\" }} {{ keep \"\\\"app.mode\\\"\" \"\" }}",
			`"light" "dark" "dark"`, ""},
		{"json quoted key escapes", "s.json", ptr(`{"a\"b": {"c\\d": 1}}`),
			"{{ keep `\"a\\\"b\".\"c\\\\d\"` 0 }}", `1`, ""},
		{"json bad path", "s.json", ptr(`{}`),
			`{{ keep "a..b" 1 }}`, "", `invalid path "a..b": empty key`},
		{"json string escaping", "s.json", ptr(`{"a": "q\"b\\n\nü<"}`),
			`{{ keep "a" "" }} {{ keep "x" "q\"b\\n\nü<" }}`, `"q\"b\\n\nü<" "q\"b\\n\nü<"`, ""},

		// TOML
		{"toml present", "s.toml", ptr(tomlDest),
			`mode = {{ keep "mode" "default" }}`, `mode = 'custom'`, ""},
		{"toml missing dest", "s.toml", nil,
			`mode = {{ keep "mode" "default" }}`, `mode = 'default'`, ""},
		{"toml missing key", "s.toml", ptr(tomlDest),
			`{{ keep "section.nope" "x" }}`, `'x'`, ""},
		{"toml kinds", "s.toml", ptr(tomlDest),
			`{{ keep "section.a" 0 }} {{ keep "section.z" false }} {{ keep "section.list" "" }} {{ keep "section.deep.when" "" }}`,
			`1.5 true [1, 2] 2024-01-02`, ""},
		{"toml flat arrays", "s.toml", ptr("a = [\"x\", 2.0, true, 1979-05-27]\nb = []\n"),
			`{{ keep "a" "" }} {{ keep "b" "" }}`, `['x', 2.0, true, 1979-05-27] []`, ""},
		{"toml table is not a value", "s.toml", ptr(tomlDest),
			`{{ keep "section" "" }}`, "", `keep "section": a table, not a value`},
		{"toml inline table is not a value", "s.toml", ptr(tomlDest),
			`{{ keep "section.inline" "" }}`, "", `keep "section.inline": a table, not a value`},
		{"toml array of tables", "s.toml", ptr("[[t]]\nn = 1\n[[t]]\nn = 2\n"),
			`{{ keep "t" "" }}`, "", `keep "t": an array holding a table or an array`},
		{"toml array of inline tables", "s.toml", ptr("list = [{ a = 1 }]\n"),
			`{{ keep "list" "" }}`, "", `an array holding a table or an array`},
		{"toml nested array", "s.toml", ptr("list = [[1, 2], [3]]\n"),
			`{{ keep "list" "" }}`, "", `an array holding a table or an array`},
		{"toml literal string stays literal", "s.toml", ptr(tomlDest),
			"{{ keep `section.\"weird key\"` \"\" }}", `'it"s'`, ""},
		{"toml string escaping", "s.toml", ptr("a = \"q\\\"b\\\\n\\nü\\u0001\"\n"),
			`{{ keep "a" "" }} {{ keep "x" "q\"b\\n\nü\u0001\t" }}`, `"q\"b\\n\nü\u0001" "q\"b\\n\nü\u0001\t"`, ""},
		{"toml bool and number defaults", "s.toml", nil,
			`{{ keep "a" true }} {{ keep "b" 3 }} {{ keep "c" 2.5 }} {{ keep "d" 2.0 }}`, `true 3 2.5 2.0`, ""},
		{"toml quoted path keys", "s.toml", ptr("[\"[lang]\"]\n\"editor.size\" = 4\n[section.\"my.sub\"]\nkey = \"red\"\n"),
			"{{ keep `\"[lang]\".\"editor.size\"` 2 }} {{ keep `section.\"my.sub\".key` \"\" }} {{ keep \"section.my.sub.key\" \"none\" }}",
			`4 'red' 'none'`, ""},
		{"toml empty file is an empty document", "s.toml", ptr(""),
			`{{ keep "a" 1 }} {{ has "a" }}`, `1 false`, ""},

		// YAML
		{"yaml present", "s.yaml", ptr("section:\n  key: value # comment\n"),
			`key: {{ keep "section.key" "default" }}`, `key: value`, ""},
		{"yaml missing dest", "s.yml", nil,
			`key: {{ keep "section.key" "default" }}`, `key: default`, ""},
		{"yaml missing key", "s.yml", ptr("section: {}\n"),
			`{{ keep "section.key" "default" }}`, `default`, ""},
		{"yaml kinds", "s.yml", ptr("a: 3\nb: yes\nc: [1, x, 0x1F, ~, !Ref r]\ne: |\n  two\n  lines\nf: ~\ng: []\n"),
			`{{ keep "a" 0 }} {{ keep "b" false }} {{ keep "c" "" }} {{ keep "e" "" }} {{ keep "f" "def" }} {{ keep "g" "" }}`,
			`3 "yes" [1, x, 31, null, r] "two\nlines\n" def []`, ""},
		{"yaml aliases are resolved", "s.yaml", ptr("base: &b {k: 1}\nuse: *b\nx: &v [1, 2]\ny: *v\n"),
			`{{ keep "use.k" 0 }} {{ keep "y" "" }} {{ has "use" }}`, `1 [1, 2] true`, ""},
		{"yaml self-referencing alias", "s.yaml", ptr("a: &x [1, *x]\n"),
			`{{ keep "a" "" }}`, "", `cannot parse`},
		{"yaml non-string keys", "s.yaml", ptr("1: a\ntrue: b\nx: {2: c, k: d}\n"),
			`{{ keep "1" "" }} {{ keep "true" "" }} {{ keep "x.2" "" }} {{ keep "x.k" "" }}`, `a b c d`, ""},
		{"yaml mapping is not a value", "s.yaml", ptr("d:\n  z: 1\n"),
			`{{ keep "d" "" }}`, "", `keep "d": a table, not a value`},
		{"yaml sequence of mappings", "s.yaml", ptr("c:\n  - a: 1\n"),
			`{{ keep "c" "" }}`, "", `keep "c": an array holding a table or an array`},
		{"yaml nested sequence", "s.yaml", ptr("c: [1, [2]]\n"),
			`{{ keep "c" "" }}`, "", `an array holding a table or an array`},
		{"yaml quoted path key", "s.yaml", ptr("\"a.b\":\n  c: 1\n"),
			"{{ keep `\"a.b\".c` 0 }}", `1`, ""},
		{"yaml bool and number defaults", "s.yaml", nil,
			`{{ keep "a" true }} {{ keep "b" 3 }} {{ keep "c" 2.5 }}`, `true 3 2.5`, ""},
		{"yaml string defaults", "s.yaml", nil,
			`{{ keep "a" "yes" }} {{ keep "b" "1.0" }} {{ keep "c" "a: b" }} {{ keep "d" "two\nlines" }} {{ keep "e" "" }}`,
			`"yes" "1.0" 'a: b' "two\nlines" ""`, ""},
		{"yaml block scalars stay on one line", "s.yaml", ptr("a: >-\n  folded\n  text\nb: |-\n  one\n"),
			`{{ keep "a" "" }} {{ keep "b" "" }}`, `folded text one`, ""},

		// Any format
		{"no actions is a plain copy", "s.conf", ptr("old"),
			"line 1\n\"quoted\" \\ ü\n\n", "line 1\n\"quoted\" \\ ü\n\n", ""},
		{"keep on unsupported extension", "s.conf", ptr("old"),
			`{{ keep "a" "b" }}`, "", `unsupported extension ".conf"`},
		{"bad default", "s.json", nil,
			`{{ keep "a" nil }}`, "", "DEFAULT must be"},
		{"template syntax error", "s.json", ptr(`{}`),
			"ok\n{{ keep \"a\" ", "", "template:"},
		{"unknown function", "s.json", ptr(`{}`),
			`{{ nope }}`, "", `function "nope" not defined`},
		{"unparseable json", "s.json", ptr(`{"a": `),
			`{{ keep "a" 1 }}`, "", "cannot parse"},
		{"whitespace in path", "s.json", ptr(`{"x": {"y": 1}}`),
			`{{ keep "x. y" 0 }}`, "", `invalid path "x. y": whitespace in unquoted key " y"`},
		{"unparseable json has line", "s.json", ptr("{\n\"a\": }"),
			`{{ keep "a" 1 }}`, "", "line 2:"},
		{"unparseable toml", "s.toml", ptr("a = = 1\n"),
			`{{ keep "a" 1 }}`, "", "cannot parse"},
		{"unparseable toml has position", "s.toml", ptr("a = 1\nb = = 1\n"),
			`{{ keep "a" 1 }}`, "", "line 2, column 5"},
		{"unparseable yaml", "s.yaml", ptr("a: [1\n"),
			`{{ keep "a" 1 }}`, "", "cannot parse"},
		{"empty json file is unparseable", "s.json", ptr(""),
			`{{ keep "a" 1 }}`, "", "cannot parse"},

		// has
		{"has table", "s.toml", ptr(tomlDest),
			`{{ has "section" }} {{ has "section.deep" }} {{ has "section.inline" }}`, `true true true`, ""},
		{"has value", "s.toml", ptr(tomlDest),
			`{{ has "mode" }} {{ has "section.list" }}`, `true true`, ""},
		{"has absent", "s.toml", ptr(tomlDest),
			`{{ has "section.nope" }} {{ has "mode.x" }} {{ has "nope" }}`, `false false false`, ""},
		{"has missing dest toml", "s.toml", nil,
			`{{ has "section" }}`, `false`, ""},
		{"has missing dest json", "s.json", nil,
			`{{ has "a" }}`, `false`, ""},
		{"has missing dest yaml", "s.yaml", nil,
			`{{ has "a" }}`, `false`, ""},
		{"has json", "s.json", ptr(`{"a": null, "b": {}, "c": {"d": 0}}`),
			`{{ has "a" }} {{ has "b" }} {{ has "c.d" }} {{ has "c.e" }}`, `false true true false`, ""},
		{"has yaml", "s.yaml", ptr("a: ~\nb: null\nc: {}\nd:\n  e: 0\n"),
			`{{ has "a" }} {{ has "b" }} {{ has "c" }} {{ has "d.e" }} {{ has "d.f" }}`, `false false true true false`, ""},
		{"has block present", "s.toml", ptr("[t]\nk = 1\n"),
			"x = 0\n{{- if has \"t\" }}\n\n[t]\nk = {{ keep \"t.k\" 0 }}\n{{- end }}\n", "x = 0\n\n[t]\nk = 1\n", ""},
		{"has block absent", "s.toml", ptr("x = 0\n"),
			"x = 0\n{{- if has \"t\" }}\n\n[t]\nk = {{ keep \"t.k\" 0 }}\n{{- end }}\n", "x = 0\n", ""},
		{"has bad path", "s.toml", ptr("[t]\nk = 1\n"),
			`{{ has "t." }}`, "", `invalid path "t.": empty key`},
		{"has bad path missing dest", "s.toml", nil,
			`{{ has "a..b" }}`, "", `invalid path "a..b": empty key`},
		{"has unsupported extension", "s.conf", ptr("old"),
			`{{ has "a" }}`, "", `unsupported extension ".conf"`},
		{"has unparseable dest", "s.toml", ptr("a = = 1\n"),
			`{{ has "a" }}`, "", "cannot parse"},
		{"has unparseable json dest", "s.json", ptr(`{"a": `),
			`{{ has "a" }}`, "", "fix or delete it, then re-run dis config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, dest, err := renderInDir(t, tt.destName, tt.dest, tt.tmpl)
			got, readErr := os.ReadFile(dest)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				// DEST is untouched.
				if tt.dest == nil {
					if readErr == nil {
						t.Errorf("DEST was created: %q", got)
					}
				} else if string(got) != *tt.dest {
					t.Errorf("DEST changed to %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// Template errors name SRC, DEST and the template location.
func TestRenderConfigErrorsNamePaths(t *testing.T) {
	src, dest, err := renderInDir(t, "s.json", ptr(`{"a": 1}`), "x\n  {{ keep \"a..b\" 1 }}")
	if err == nil {
		t.Fatal("want error")
	}
	for _, want := range []string{src + ":2:", dest} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err %q does not contain %q", err, want)
		}
	}
}

// Rendering leaves no temp file next to DEST.
func TestRenderConfigNoTempFiles(t *testing.T) {
	_, dest, err := renderInDir(t, "s.json", ptr(`{}`), "{}")
	if err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(filepath.Dir(dest)); len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		path    string
		want    []string
		wantErr string
	}{
		{`key-name`, []string{"key-name"}, ""},
		{`section.sub.key_name`, []string{"section", "sub", "key_name"}, ""},
		{`"app.mode"`, []string{"app.mode"}, ""},
		{`"[lang]"."editor.size"`, []string{"[lang]", "editor.size"}, ""},
		{`section."my.sub".key`, []string{"section", "my.sub", "key"}, ""},
		{`"a\"b"`, []string{`a"b`}, ""},
		{`"a\\b"`, []string{`a\b`}, ""},
		{`"a\\"."\""`, []string{`a\`, `"`}, ""},
		{`""`, []string{""}, ""},
		{`a."".b`, []string{"a", "", "b"}, ""},
		{`a"b`, []string{`a"b`}, ""}, // a quote inside an unquoted key is a plain character
		{`"a b". c d `, nil, `invalid path "\"a b\". c d ": whitespace in unquoted key " c d " (put the key in double quotes if the whitespace is part of it)`},
		{`"a b"."c d "`, []string{"a b", "c d "}, ""},
		{`x. y`, nil, `invalid path "x. y": whitespace in unquoted key " y"`},
		{`x .y`, nil, `invalid path "x .y": whitespace in unquoted key "x "`},
		{`a b.c`, nil, `invalid path "a b.c": whitespace in unquoted key "a b"`},
		{"a\tb", nil, `whitespace in unquoted key "a\tb"`},

		{`"a`, nil, `invalid path "\"a": unterminated quoted key`},
		{`a."b.c`, nil, `invalid path "a.\"b.c": unterminated quoted key`},
		{`"a\`, nil, `invalid path "\"a\\": unterminated quoted key`},
		{`"a\"`, nil, `invalid path "\"a\\\"": unterminated quoted key`},
		{`a..b`, nil, `invalid path "a..b": empty key (a leading, trailing or doubled ".")`},
		{`.a`, nil, `invalid path ".a": empty key`},
		{`a.`, nil, `invalid path "a.": empty key`},
		{`"a".`, nil, `invalid path "\"a\".": empty key`},
		{``, nil, `invalid path "": empty key`},
		{`"a"b`, nil, `invalid path "\"a\"b": "b" after a quoted key, want "." or the end of the path`},
		{`"a" .b`, nil, `invalid path "\"a\" .b": " .b" after a quoted key`},
		{`"a\n"`, nil, `invalid path "\"a\\n\"": unsupported escape \n in a quoted key: only \" and \\ are escapes`},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := parsePath(tt.path)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// An unparseable DEST fails with how to recover first, not inside the
// template's error, naming the package to re-configure when run from an
// installer.
func TestRenderConfigParseErrorHint(t *testing.T) {
	for _, tt := range []struct{ pkg, want string }{
		{"", "): fix or delete it, then re-run dis config"},
		{"common/app", "): fix or delete it, then re-run dis config for common/app"},
	} {
		t.Run(tt.pkg, func(t *testing.T) {
			t.Setenv("DIS_PACKAGE", tt.pkg)
			_, dest, err := renderInDir(t, "c.toml", ptr("a = = 1\n"), "x\n{{ keep \"a\" 1 }}")
			if err == nil {
				t.Fatal("want error")
			}
			want := "cannot parse " + dest + " (line 1, column 5: toml: "
			if !strings.HasPrefix(err.Error(), want) {
				t.Errorf("err %q does not start with %q", err, want)
			}
			if !strings.HasSuffix(err.Error(), tt.want) {
				t.Errorf("err %q does not end with %q", err, tt.want)
			}
		})
	}
}

// A DEST symlink stays a link and has its target written, whether the target
// exists or not.
func TestRenderConfigDanglingSymlink(t *testing.T) {
	tests := []struct {
		name string
		// setup creates the links under dir and returns DEST and the file
		// that should be written.
		setup func(t *testing.T, dir string) (dest, target string)
	}{
		{"absolute", func(t *testing.T, dir string) (string, string) {
			target := filepath.Join(dir, "new", "dir", "target.json")
			return symlink(t, target, filepath.Join(dir, "link.json")), target
		}},
		{"relative", func(t *testing.T, dir string) (string, string) {
			return symlink(t, "real/target.json", filepath.Join(dir, "link.json")), filepath.Join(dir, "real", "target.json")
		}},
		{"relative to the link's dir", func(t *testing.T, dir string) (string, string) {
			link := filepath.Join(dir, "links", "link.json")
			mkdir(t, filepath.Dir(link))
			return symlink(t, "../data/target.json", link), filepath.Join(dir, "data", "target.json")
		}},
		{"relative through a symlinked dir", func(t *testing.T, dir string) (string, string) {
			mkdir(t, filepath.Join(dir, "a", "b"))
			symlink(t, filepath.Join(dir, "a", "b"), filepath.Join(dir, "short"))
			// ".." goes up from a/b, where the link really is, not from short.
			link := symlink(t, "../target.json", filepath.Join(dir, "a", "b", "link.json"))
			return filepath.Join(dir, "short", filepath.Base(link)), filepath.Join(dir, "a", "target.json")
		}},
		{"chain", func(t *testing.T, dir string) (string, string) {
			symlink(t, "c.json", filepath.Join(dir, "b.json"))
			symlink(t, filepath.Join(dir, "sub", "d.json"), filepath.Join(dir, "c.json"))
			return symlink(t, "b.json", filepath.Join(dir, "a.json")), filepath.Join(dir, "sub", "d.json")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			dest, target := tt.setup(t, dir)
			src := filepath.Join(dir, "template")
			if err := os.WriteFile(src, []byte(`{{ keep "a" "x" }}`), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := RenderConfig(src, dest); err != nil {
				t.Fatal(err)
			}
			if fi, err := os.Lstat(dest); err != nil || fi.Mode()&os.ModeSymlink == 0 {
				t.Errorf("link replaced: %v, %v", fi, err)
			}
			if got := readRC(t, target); got != `"x"` {
				t.Errorf("target = %q", got)
			}
			// Rendering again reads the target back through the link.
			if err := os.WriteFile(target, []byte(`{"a": "kept"}`), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := RenderConfig(src, dest); err != nil {
				t.Fatal(err)
			}
			if got := readRC(t, target); got != `"kept"` {
				t.Errorf("target after re-render = %q", got)
			}
			if fi, err := os.Lstat(dest); err != nil || fi.Mode()&os.ModeSymlink == 0 {
				t.Errorf("link replaced on re-render: %v, %v", fi, err)
			}
		})
	}
}

// A symlink loop fails rather than hang.
func TestRenderConfigSymlinkLoop(t *testing.T) {
	dir := t.TempDir()
	symlink(t, "a.json", filepath.Join(dir, "b.json"))
	dest := symlink(t, "b.json", filepath.Join(dir, "a.json"))
	src := filepath.Join(dir, "template")
	if err := os.WriteFile(src, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- RenderConfig(src, dest) }()
	select {
	case err := <-done:
		if want := dest + ": too many levels of symbolic links"; err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %v, want it to contain %q", err, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RenderConfig hangs on a symlink loop")
	}
}

func symlink(t *testing.T, target, link string) string {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	return link
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

// Rendering over the result of a render changes nothing: what keep writes is
// read back as the same value.
func TestRenderConfigIdempotent(t *testing.T) {
	tests := []struct {
		name     string
		destName string
		dest     *string
		tmpl     string
	}{
		{"json", "s.json", ptr(`{"a": {"z": [1, "x\"y"], "b": 2.50}, "app.mode": "dark"}`),
			"{\"a\": {\"z\": {{ keep \"a.z\" \"\" }}, \"b\": {{ keep \"a.b\" 0 }}}, \"app.mode\": {{ keep `\"app.mode\"` \"light\" }}}"},
		{"toml", "s.toml", ptr(tomlDest),
			"mode = {{ keep \"mode\" \"default\" }}\n# end\n{{- if has \"section\" }}\n\n[section]\nkey = {{ keep \"section.key\" \"\" }}\nlist = {{ keep \"section.list\" \"\" }}\n{{- end }}\n\n[other]\nx = 1\n"},
		{"toml no table", "s.toml", nil,
			"mode = {{ keep \"mode\" \"default\" }}\n# end\n{{- if has \"section\" }}\n\n[section]\nkey = {{ keep \"section.key\" \"\" }}\nlist = {{ keep \"section.list\" \"\" }}\n{{- end }}\n\n[other]\nx = 1\n"},
		{"yaml", "s.yaml", ptr("section:\n  key: value\n  list: [1, \"x\", yes, 0x1F]\n"),
			"section:\n  key: {{ keep \"section.key\" \"default\" }}\n  list: {{ keep \"section.list\" \"\" }}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dest, err := renderInDir(t, tt.destName, tt.dest, tt.tmpl)
			if err != nil {
				t.Fatal(err)
			}
			first := readRC(t, dest)
			if err := RenderConfig(src, dest); err != nil {
				t.Fatal(err)
			}
			if second := readRC(t, dest); second != first {
				t.Errorf("second render differs:\nfirst:\n%s\nsecond:\n%s", first, second)
			}
		})
	}
}

// The config templates dis's packages ship render on a new host, keep what
// another tool (a theme engine) wrote on a themed one, and re-rendering
// changes nothing.
func TestPackageConfigTemplates(t *testing.T) {
	const pkgs = "../../packages"
	tests := []struct {
		name     string
		src      string
		destName string
		dest     *string  // nil: a new host
		want     []string // in the output
		check    func(t *testing.T, out []byte)
	}{
		{"ulauncher new", "ubuntu/configs/ulauncher.json.tmpl", "settings.json", nil,
			[]string{`"theme-name": "dark"`}, checkJSON},
		{"ulauncher themed", "ubuntu/configs/ulauncher.json.tmpl", "settings.json",
			ptr(`{"show-indicator-icon": false, "theme-name": "my-theme"}`),
			[]string{`"theme-name": "my-theme"`}, checkJSON},
		{"starship new", "all/configs/starship/starship.toml.tmpl", "starship.toml", nil,
			[]string{`palette = 'custom'`, "[palettes.custom]\nmain_color = '#A08AE2'\nsecondary_color = '#A9E9B7'\n"}, checkTOML},
		{"starship themed", "all/configs/starship/starship.toml.tmpl", "starship.toml",
			ptr("palette = \"custom\"\n\n[palettes.custom]\nmain_color = \"#112233\"\nsecondary_color = \"#445566\"\n"),
			[]string{`palette = 'custom'`, "[palettes.custom]\nmain_color = '#112233'\nsecondary_color = '#445566'\n"}, checkTOML},
		{"herdr new", "all/configs/herdr/config.toml.tmpl", "config.toml", nil,
			[]string{`name = 'gruvbox'`, "# text = \"#cdd6f4\"\n\n[terminal]\n"}, checkHerdr(false)},
		{"herdr themed", "all/configs/herdr/config.toml.tmpl", "config.toml", ptr(herdrThemed),
			[]string{`name = 'terminal'`, "# text = \"#cdd6f4\"\n\n[theme.custom]\naccent = '#ff79c6'\nred = '#ff5555'\n" +
				"green = '#50fa7b'\nblue = '#8be9fd'\nyellow = '#f1fa8c'\nmauve = '#bd93f9'\nteal = '#8be9fe'\n\n[terminal]\n"},
			checkHerdr(true)},
		{"herdr theme colors cleared", "all/configs/herdr/config.toml.tmpl", "config.toml",
			ptr("onboarding = false\n\n[theme]\nname = \"nord\"\nauto_switch = false\n\n[terminal]\ndefault_shell = \"/bin/zsh\"\n"),
			[]string{`name = 'nord'`, "# text = \"#cdd6f4\"\n\n[terminal]\n"}, checkHerdr(false)},
		{"herdr partial theme.custom", "all/configs/herdr/config.toml.tmpl", "config.toml",
			ptr("[theme.custom]\naccent = \"#ff79c6\"\n"),
			[]string{"[theme.custom]\naccent = '#ff79c6'\nred = 'red'\n"}, checkHerdr(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := os.ReadFile(filepath.Join(pkgs, tt.src))
			if err != nil {
				t.Fatal(err)
			}
			src, dest, err := renderInDir(t, tt.destName, tt.dest, string(tmpl))
			if err != nil {
				t.Fatal(err)
			}
			out, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.want {
				if !strings.Contains(string(out), want) {
					t.Errorf("output does not contain %q", want)
				}
			}
			tt.check(t, out)
			if err := RenderConfig(src, dest); err != nil {
				t.Fatal(err)
			}
			if again := readRC(t, dest); again != string(out) {
				t.Errorf("second render differs")
			}
		})
	}
}

func checkJSON(t *testing.T, out []byte) {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Errorf("output is not JSON: %v", err)
	}
}

func checkTOML(t *testing.T, out []byte) {
	t.Helper()
	var v map[string]any
	if err := toml.Unmarshal(out, &v); err != nil {
		t.Errorf("output is not TOML: %v", err)
	}
}

// herdrThemed is a herdr config after a theme engine set custom colors with
// yq, which rewrites the file and drops its comments.
const herdrThemed = `onboarding = false

[theme]
name = "terminal"
auto_switch = false

[theme.custom]
accent = "#ff79c6"
red = "#ff5555"
green = "#50fa7b"
blue = "#8be9fd"
yellow = "#f1fa8c"
mauve = "#bd93f9"
teal = "#8be9fe"

[terminal]
default_shell = "/bin/zsh"
`

// checkHerdr parses the herdr config and checks the theme: the seven custom
// colors, the theme engine's accent among them, when themed, none otherwise.
func checkHerdr(themed bool) func(t *testing.T, out []byte) {
	return func(t *testing.T, out []byte) {
		t.Helper()
		var cfg struct {
			Theme struct {
				Name   string
				Custom map[string]any
			}
			Terminal struct {
				DefaultShell string `toml:"default_shell"`
			}
		}
		if err := toml.Unmarshal(out, &cfg); err != nil {
			t.Fatalf("output is not TOML: %v", err)
		}
		if cfg.Terminal.DefaultShell != "/bin/bash" {
			t.Errorf("default_shell = %q, want the template's", cfg.Terminal.DefaultShell)
		}
		if strings.Contains(string(out), "\n\n\n") {
			t.Error("output has two blank lines in a row")
		}
		if themed {
			if len(cfg.Theme.Custom) != 7 || cfg.Theme.Custom["accent"] != "#ff79c6" {
				t.Errorf("theme.custom = %v", cfg.Theme.Custom)
			}
		} else if cfg.Theme.Custom != nil {
			t.Errorf("theme.custom = %v, want none", cfg.Theme.Custom)
		}
	}
}

// Dates and times of all four TOML kinds come out as written, on a host far
// from UTC too: local ones are not moved to any zone, offset ones keep theirs.
func TestRenderConfigTOMLDateTimesNonUTC(t *testing.T) {
	old := time.Local
	time.Local = time.FixedZone("UTC+14", 14*60*60)
	t.Cleanup(func() { time.Local = old })
	const values = "odt = 1979-05-27T00:32:00-07:00\nodtz = 1979-05-27T23:32:00.5Z\nldt = 1979-05-27T00:32:00.100\nld = 1979-05-27\nlt = 00:32:00.999999\n"
	_, dest, err := renderInDir(t, "c.toml", ptr("[t]\n"+values), "{{ keep \"t.odt\" \"\" }} {{ keep \"t.odtz\" \"\" }} {{ keep \"t.ldt\" \"\" }} {{ keep \"t.ld\" \"\" }} {{ keep \"t.lt\" \"\" }}\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "1979-05-27T00:32:00-07:00 1979-05-27T23:32:00.5Z 1979-05-27T00:32:00.100 1979-05-27 00:32:00.999999\n"
	if got := readRC(t, dest); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderConfigTOMLRoundTrip(t *testing.T) {
	const dest = `s1 = "plain"
s2 = "it's \"q\" \\ back"
s3 = "ctl \u0001 tab\t nl\n cr\r del\u007f"
s4 = "ünï 日本 😀"
s5 = ''
i1 = 9223372036854775807
i2 = -9223372036854775808
f1 = 2.0
f2 = 1.5e300
f3 = inf
f4 = -inf
f5 = -0.0
f6 = 5e-324
b = true
odt = 1979-05-27T00:32:00.999999-07:00
ldt = 1979-05-27T00:32:00.100
ld = 1979-05-27
lt = 00:32:00.999999
a = ["x", 'y', 1, 2.0, true, 1979-05-27, 00:32:00]
nan = nan
`
	var want map[string]any
	if err := toml.Unmarshal([]byte(dest), &want); err != nil {
		t.Fatal(err)
	}
	var tmpl strings.Builder
	for k := range want {
		fmt.Fprintf(&tmpl, "%s = {{ keep %q \"\" }}\n", k, k)
	}
	_, out, err := renderInDir(t, "c.toml", ptr(dest), tmpl.String())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal([]byte(readRC(t, out)), &got); err != nil {
		t.Fatalf("output is not TOML: %v\n%s", err, readRC(t, out))
	}
	if f, _ := got["nan"].(float64); !math.IsNaN(f) {
		t.Errorf("nan = %v", got["nan"])
	}
	delete(got, "nan")
	delete(want, "nan")
	if f, _ := got["f5"].(float64); !math.Signbit(f) {
		t.Errorf("f5 = %v, want -0.0", got["f5"])
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v\noutput:\n%s", got, want, readRC(t, out))
	}
}

// Every YAML value keep writes is on one line and reads back as the same
// value: array items that need quoting in a flow sequence, block and
// multi-line strings, strings that look like other types, whole floats,
// timestamps.
func TestRenderConfigYAMLRoundTrip(t *testing.T) {
	const dest = `flow:
  - a, b
  - c]
  - "{x}"
  - "#c"
  - "- d"
  - ": e"
  - plain
long: [` + "w0, w1, w2, w3, w4, w5, w6, w7, w8, w9, w10, w11, w12, w13, w14, w15, w16, w17, w18, w19, w20, w21, w22, w23, w24, w25, w26, w27, w28, w29" + `]
mixed: [1, 2.5, 1.0, true, ~, 0x1F, "yes", "1.0", "~", !Ref r]
stamp: 2024-01-02T10:00:00.5+02:00
stamps: [2024-01-02, 2024-01-02T10:00:00+02:00]
whole: 1.0
binary: !!binary |
  R0lGODlhDAAMAIQAAP//9/X17unp5WZmZgAAAOfn515eXvPz7Y6OjuDg4J+fn5
  OTk6enp56enmlpaWNjY6Ojo4SEhP/++f/++f/++f/++f/++f/++f/++f/++f/+
tagged: !Ref |
  two
  lines
strip: |-
  one
  two
folded: >
  a
  b
single: 'first

  second'
anchored: &x value # comment
alias: *x
strs: ["yes", "1.0", "~", "null", "a: b", "[x]"]
`
	keys := []string{"flow", "long", "mixed", "stamp", "stamps", "whole", "binary", "tagged", "strip", "folded", "single", "anchored", "alias", "strs"}
	var tmpl strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&tmpl, "%s: {{ keep %q \"\" }}\n", k, k)
	}
	_, out, err := renderInDir(t, "c.yaml", ptr(dest), tmpl.String())
	if err != nil {
		t.Fatal(err)
	}
	text := readRC(t, out)
	if n := strings.Count(text, "\n"); n != len(keys) {
		t.Errorf("output has %d lines, want %d:\n%s", n, len(keys), text)
	}
	if got, want := yamlValues(t, text), yamlValues(t, dest); !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v\noutput:\n%s", got, want, text)
	}
}

// yamlValues decodes data, each timestamp as its instant, so the two sides
// compare with reflect.DeepEqual whatever their time.Location.
func yamlValues(t *testing.T, data string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("not YAML: %v\n%s", err, data)
	}
	instant := func(v any) any {
		if tm, ok := v.(time.Time); ok {
			return tm.UnixNano()
		}
		return v
	}
	for k, v := range m {
		if items, ok := v.([]any); ok {
			for i := range items {
				items[i] = instant(items[i])
			}
		} else {
			m[k] = instant(v)
		}
	}
	return m
}

// JSON leaves and flat arrays read back as the same values, numbers as
// written.
func TestRenderConfigJSONRoundTrip(t *testing.T) {
	const dest = `{"s": "q\"b\\ <&> \u00fc \n", "n": 1.50, "big": 12345678901234567890, "e": 1e3, "b": false, "a": ["x", 2.0, null, true, "]"]}`
	_, out, err := renderInDir(t, "c.json", ptr(dest),
		`{"s": {{ keep "s" "" }}, "n": {{ keep "n" 0 }}, "big": {{ keep "big" 0 }}, "e": {{ keep "e" 0 }}, "b": {{ keep "b" true }}, "a": {{ keep "a" "" }}}`)
	if err != nil {
		t.Fatal(err)
	}
	text := readRC(t, out)
	for _, want := range []string{`"n": 1.50`, `"big": 12345678901234567890`, `"e": 1e3`, `"a": ["x",2.0,null,true,"]"]`} {
		if !strings.Contains(text, want) {
			t.Errorf("output does not contain %s:\n%s", want, text)
		}
	}
	var got, want any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, text)
	}
	json.Unmarshal([]byte(dest), &want) //nolint:errcheck
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
