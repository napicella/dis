package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// renderInDir writes the template, and dest unless it is nil, to a temp dir
// and renders them. It returns dest's path.
func renderInDir(t *testing.T, destName string, dest *string, tmpl string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "template")
	if err := os.WriteFile(src, []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}
	destPath := filepath.Join(dir, "out", destName)
	if dest != nil {
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destPath, []byte(*dest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return destPath, RenderConfig(src, destPath)
}

func ptr(s string) *string { return &s }

const starshipDest = `palette = "custom"

[palettes.custom]
main_color = "#ff0000"
"weird key" = 'it"s'
inline = { b = 1, a = 2 }
list = [1, 2]

[palettes.custom.zeta]
z = true

[palettes.custom.alpha]
a = 1.5

[palettes.custom.alpha.deep]
when = 2024-01-02

[other]
x = 1
`

// starshipTable is palettes.custom from starshipDest: keys sorted, then the
// subtables sorted, the inline table as one of them.
const starshipTable = `[palettes.custom]
list = [1, 2]
main_color = "#ff0000"
"weird key" = "it\"s"

[palettes.custom.alpha]
a = 1.5

[palettes.custom.alpha.deep]
when = 2024-01-02

[palettes.custom.inline]
a = 2
b = 1

[palettes.custom.zeta]
z = true`

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
		{"json present string", "s.json", ptr(`{"theme-name": "wal-picker"}`),
			`{"theme-name": {{ keep "theme-name" "dark" }}}`, `{"theme-name": "wal-picker"}`, ""},
		{"json missing dest", "s.json", nil,
			`{"theme-name": {{ keep "theme-name" "dark" }}}`, `{"theme-name": "dark"}`, ""},
		{"json missing key", "s.json", ptr(`{"other": 1}`),
			`{{ keep "theme-name" "dark" }}`, `"dark"`, ""},
		{"json null is no value", "s.json", ptr(`{"a": null}`),
			`{{ keep "a" "x" }}`, `"x"`, ""},
		{"json nested and kinds", "s.json", ptr(`{"a": {"b": 3, "c": true, "d": [1, "x"], "e": {"z": 1, "y": 2}}}`),
			`{{ keep "a.b" 0 }} {{ keep "a.c" false }} {{ keep "a.d" "" }} {{ keep "a.e" "" }}`,
			`3 true [1,"x"] {"z":1,"y":2}`, ""},
		{"json path through non-object", "s.json", ptr(`{"a": "s"}`),
			`{{ keep "a.b" 1 }}`, `1`, ""},
		{"json bool and number defaults", "s.json", nil,
			`{{ keep "a" true }} {{ keep "b" 3 }} {{ keep "c" 2.5 }}`, `true 3 2.5`, ""},
		{"json flat key with dots", "s.json", ptr(`{"workbench.colorTheme": "Nord"}`),
			"{{ keep \"workbench.colorTheme\" \"Default Dark+\" }} {{ keep `\"workbench.colorTheme\"` \"Default Dark+\" }} {{ keep \"\\\"workbench.colorTheme\\\"\" \"\" }}",
			`"Default Dark+" "Nord" "Nord"`, ""},
		{"json quoted key escapes", "s.json", ptr(`{"a\"b": {"c\\d": 1}}`),
			"{{ keep `\"a\\\"b\".\"c\\\\d\"` 0 }}", `1`, ""},
		{"json bad path", "s.json", ptr(`{}`),
			`{{ keep "a..b" 1 }}`, "", `invalid path "a..b": empty key`},
		{"json string escaping", "s.json", ptr(`{"a": "q\"b\\n\nü<"}`),
			`{{ keep "a" "" }} {{ keep "x" "q\"b\\n\nü<" }}`, `"q\"b\\n\nü<" "q\"b\\n\nü<"`, ""},

		// TOML
		{"toml present", "s.toml", ptr(starshipDest),
			`palette = {{ keep "palette" "default" }}`, `palette = "custom"`, ""},
		{"toml missing dest", "s.toml", nil,
			`palette = {{ keep "palette" "default" }}`, `palette = "default"`, ""},
		{"toml missing key", "s.toml", ptr(starshipDest),
			`{{ keep "palettes.custom.nope" "x" }}`, `"x"`, ""},
		{"toml kinds", "s.toml", ptr(starshipDest),
			`{{ keep "palettes.custom.alpha.a" 0 }} {{ keep "palettes.custom.zeta.z" false }} {{ keep "palettes.custom.list" "" }} {{ keep "palettes.custom.inline" "" }} {{ keep "palettes.custom.alpha.deep.when" "" }}`,
			`1.5 true [1, 2] { a = 2, b = 1 } 2024-01-02`, ""},
		{"toml literal string becomes basic", "s.toml", ptr(starshipDest),
			"{{ keep `palettes.custom.\"weird key\"` \"\" }}", `"it\"s"`, ""},
		{"toml string escaping", "s.toml", ptr("a = \"q\\\"b\\\\n\\nü\\u0001\"\n"),
			`{{ keep "a" "" }} {{ keep "x" "q\"b\\n\nü\u0001\t" }}`, `"q\"b\\n\nü\u0001" "q\"b\\n\nü\u0001\t"`, ""},
		{"toml bool and number defaults", "s.toml", nil,
			`{{ keep "a" true }} {{ keep "b" 3 }} {{ keep "c" 2.5 }} {{ keep "d" 2.0 }}`, `true 3 2.5 2.0`, ""},
		{"toml keepTable nested, sorted", "s.toml", ptr(starshipDest),
			"{{ keepTable \"palettes.custom\" }}\n", "\n\n" + starshipTable + "\n", ""},
		{"toml keepTable absent", "s.toml", ptr(starshipDest),
			`a{{ keepTable "palettes.nope" }}b`, `ab`, ""},
		{"toml keepTable missing dest", "s.toml", nil,
			`a{{ keepTable "palettes.custom" }}b`, `ab`, ""},
		{"toml keepTable header only for subtables", "s.toml", ptr("[a.b.c]\nx = 1\n[a.b.d]\ny = 2\n"),
			`{{ keepTable "a" }}`, "\n\n[a]\n\n[a.b.c]\nx = 1\n\n[a.b.d]\ny = 2", ""},
		{"toml keepTable dotted keys", "s.toml", ptr("[t]\nk = 1\nsub.x = 2\n"),
			`{{ keepTable "t" }}`, "\n\n[t]\nk = 1\n\n[t.sub]\nx = 2", ""},
		{"toml keepTable quoted keys", "s.toml", ptr("[palettes.\"my.palette\"]\ncolor = \"red\"\n\"a.b\".c = 1\n"),
			"{{ keepTable `palettes.\"my.palette\"` }}", "\n\n[palettes.\"my.palette\"]\ncolor = \"red\"\n\n[palettes.\"my.palette\".\"a.b\"]\nc = 1", ""},
		{"toml keepTable with else default absent", "s.toml", ptr("x = 1\n"),
			"{{ with keepTable \"t\" }}{{ . }}{{ else }}[t]\nk = 0{{ end }}", "[t]\nk = 0", ""},
		{"toml keepTable with else default present", "s.toml", ptr("[t]\nk = 1\n"),
			"{{ with keepTable \"t\" }}{{ . }}{{ else }}[t]\nk = 0{{ end }}", "\n\n[t]\nk = 1", ""},
		{"toml keepTable bad path", "s.toml", ptr("[t]\nk = 1\n"),
			`{{ keepTable "t." }}`, "", `invalid path "t.": empty key`},
		{"toml array map keys sorted", "s.toml", ptr("z = 0\nb = 0\nlist = [{ z = 2, b = 1, m = { y = 1, x = 2 } }]\n[t]\nitems = [{ z = 1, a = 2 }]\n"),
			`{{ keep "list" "" }} {{ keepTable "t" }}`, "[{ b = 1, m = { x = 2, y = 1 }, z = 2 }] \n\n[t]\nitems = [{ a = 2, z = 1 }]", ""},
		{"toml quoted path keys", "s.toml", ptr("[\"[python]\"]\n\"editor.tabSize\" = 4\n[palettes.\"my.palette\"]\ncolor = \"red\"\n"),
			"{{ keep `\"[python]\".\"editor.tabSize\"` 2 }} {{ keep `palettes.\"my.palette\".color` \"\" }} {{ keep \"palettes.my.palette.color\" \"none\" }}",
			`4 "red" "none"`, ""},
		{"toml keepTable not a table", "s.toml", ptr(starshipDest),
			`{{ keepTable "palette" }}`, "", `keepTable "palette": not a table`},
		{"toml keepTable array of tables is an inline array", "s.toml", ptr("[t]\nk = 1\n[[t.items]]\nn = 1\nm = { b = 1, a = 2 }\n[[t.items]]\nn = 2\n"),
			`{{ keepTable "t" }} {{ keep "t.items" "" }}`, "\n\n[t]\nitems = [{ m = { a = 2, b = 1 }, n = 1 }, { n = 2 }]\nk = 1 [{ m = { a = 2, b = 1 }, n = 1 }, { n = 2 }]", ""},
		{"toml keepTable on an array of tables", "s.toml", ptr("[[t]]\nn = 1\n"),
			`{{ keepTable "t" }}`, "", `keepTable "t": an array of tables, not a table; use keep`},
		{"toml keepTable sorted", "s.toml", ptr("[t]\nz = 1\nb = \"x\"\n\"A\" = 0\n\n[t.y]\nk = 1\n\n[t.c]\nk = 2\n\n[t.c.b]\nk = 3\n"),
			`{{ keepTable "t" }}`, "\n\n[t]\nA = 0\nb = \"x\"\nz = 1\n\n[t.c]\nk = 2\n\n[t.c.b]\nk = 3\n\n[t.y]\nk = 1", ""},
		{"toml keepTable inline table is a subtable", "s.toml", ptr("[t]\nz = 1\nin = { y = { q = 1 }, x = 2 }\nempty = {}\n"),
			`{{ keepTable "t" }}`, "\n\n[t]\nz = 1\n\n[t.empty]\n\n[t.in]\nx = 2\n\n[t.in.y]\nq = 1", ""},

		// YAML
		{"yaml present", "s.yaml", ptr("theme:\n  name: wal # comment\n"),
			`name: {{ keep "theme.name" "dark" }}`, `name: "wal"`, ""},
		{"yaml missing dest", "s.yml", nil,
			`name: {{ keep "theme.name" "dark" }}`, `name: "dark"`, ""},
		{"yaml missing key", "s.yml", ptr("theme: {}\n"),
			`{{ keep "theme.name" "dark" }}`, `"dark"`, ""},
		{"yaml kinds", "s.yml", ptr("a: 3\nb: yes\nc: [1, x]\nd:\n  z: 1\n  y: \"2\"\ne: |\n  two\n  lines\nf: ~\n"),
			`{{ keep "a" 0 }} {{ keep "b" false }} {{ keep "c" "" }} {{ keep "d" "" }} {{ keep "e" "" }} {{ keep "f" "def" }}`,
			`3 "yes" [1, "x"] {"z": 1, "y": "2"} "two\nlines\n" "def"`, ""},
		{"yaml quoted path key", "s.yaml", ptr("\"a.b\":\n  c: 1\n"),
			"{{ keep `\"a.b\".c` 0 }}", `1`, ""},
		{"yaml bool and number defaults", "s.yaml", nil,
			`{{ keep "a" true }} {{ keep "b" 3 }} {{ keep "c" 2.5 }}`, `true 3 2.5`, ""},

		// Any format
		{"no actions is a plain copy", "s.conf", ptr("old"),
			"line 1\n\"quoted\" \\ ü\n\n", "line 1\n\"quoted\" \\ ü\n\n", ""},
		{"keep on unsupported extension", "s.conf", ptr("old"),
			`{{ keep "a" "b" }}`, "", `unsupported extension ".conf"`},
		{"keepTable on JSON", "s.json", ptr(`{}`),
			`{{ keepTable "a" }}`, "", "is not a .toml file"},
		{"bad default", "s.json", nil,
			`{{ keep "a" nil }}`, "", "DEFAULT must be"},
		{"template syntax error", "s.json", ptr(`{}`),
			"ok\n{{ keep \"a\" ", "", "template:"},
		{"unknown function", "s.json", ptr(`{}`),
			`{{ nope }}`, "", `function "nope" not defined`},
		{"unparseable json", "s.json", ptr(`{"a": `),
			`{{ keep "a" 1 }}`, "", "cannot parse"},
		{"unparseable json says how to recover", "s.json", ptr(`{"a": `),
			`{{ keep "a" 1 }}`, "", "fix or delete it, then re-run dis config"},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest, err := renderInDir(t, tt.destName, tt.dest, tt.tmpl)
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
	dest, err := renderInDir(t, "s.json", ptr(`{"a": 1}`), "x\n  {{ keep \"a..b\" 1 }}")
	if err == nil {
		t.Fatal("want error")
	}
	src := filepath.Join(filepath.Dir(filepath.Dir(dest)), "template")
	for _, want := range []string{src + ":2:", dest} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err %q does not contain %q", err, want)
		}
	}
}

func TestRenderConfigFileMode(t *testing.T) {
	dest, err := renderInDir(t, "s.json", ptr(`{}`), "{}")
	if err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(dest); fi.Mode().Perm() != 0o600 {
		t.Errorf("existing file mode = %v, want 0600", fi.Mode().Perm())
	}
	dest, err = renderInDir(t, "new.json", nil, "{}")
	if err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(dest); fi.Mode().Perm() != 0o644 {
		t.Errorf("new file mode = %v, want 0644", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(dest))
	if len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
}

// A symlinked DEST keeps the link and has its target rendered.
func TestRenderConfigSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	if err := os.WriteFile(target, []byte(`{"a": "kept"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "template")
	if err := os.WriteFile(src, []byte(`{{ keep "a" "x" }}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RenderConfig(src, link); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link replaced: %v, %v", fi, err)
	}
	if got := readRC(t, target); got != `"kept"` {
		t.Errorf("target = %q", got)
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		path    string
		want    []string
		wantErr string
	}{
		{`theme-name`, []string{"theme-name"}, ""},
		{`palettes.custom.main_color`, []string{"palettes", "custom", "main_color"}, ""},
		{`"workbench.colorTheme"`, []string{"workbench.colorTheme"}, ""},
		{`"[python]"."editor.tabSize"`, []string{"[python]", "editor.tabSize"}, ""},
		{`palettes."my.palette".color`, []string{"palettes", "my.palette", "color"}, ""},
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

// "{{- keepTable }}" on the line after the last content line leaves one blank
// line on each side of the table, and between the lines when there is none.
func TestRenderConfigKeepTableSpacing(t *testing.T) {
	const tmpl = "[theme]\nname = \"x\"\n# comment\n{{- keepTable \"theme.custom\" }}\n\n[terminal]\nshell = \"bash\"\n"
	tests := []struct {
		name string
		dest *string
		want string
	}{
		{"no dest", nil,
			"[theme]\nname = \"x\"\n# comment\n\n[terminal]\nshell = \"bash\"\n"},
		{"no table", ptr("[theme]\nname = \"x\"\n"),
			"[theme]\nname = \"x\"\n# comment\n\n[terminal]\nshell = \"bash\"\n"},
		{"table", ptr("[theme.custom]\naccent = \"#ff0000\"\n\n[theme.custom.dark]\nbg = \"#000000\"\n"),
			"[theme]\nname = \"x\"\n# comment\n\n[theme.custom]\naccent = \"#ff0000\"\n\n[theme.custom.dark]\nbg = \"#000000\"\n\n[terminal]\nshell = \"bash\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest, err := renderInDir(t, "c.toml", tt.dest, tmpl)
			if err != nil {
				t.Fatal(err)
			}
			if got := readRC(t, dest); got != tt.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.want)
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
		{"common/herdr", "): fix or delete it, then re-run dis config for common/herdr"},
	} {
		t.Run(tt.pkg, func(t *testing.T) {
			t.Setenv("DIS_PACKAGE", tt.pkg)
			dest, err := renderInDir(t, "c.toml", ptr("a = = 1\n"), "x\n{{ keep \"a\" 1 }}")
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

// A DEST symlink stays a link and has its target written, even when the
// target does not exist yet.
func TestRenderConfigDanglingSymlink(t *testing.T) {
	tests := []struct {
		name string
		// setup creates the links under dir and returns DEST and the file
		// that should be written.
		setup   func(t *testing.T, dir string) (dest, target string)
		wantErr string
	}{
		{"absolute", func(t *testing.T, dir string) (string, string) {
			target := filepath.Join(dir, "new", "dir", "target.json")
			return symlink(t, target, filepath.Join(dir, "link.json")), target
		}, ""},
		{"relative", func(t *testing.T, dir string) (string, string) {
			return symlink(t, "real/target.json", filepath.Join(dir, "link.json")), filepath.Join(dir, "real", "target.json")
		}, ""},
		{"relative to the link's dir", func(t *testing.T, dir string) (string, string) {
			link := filepath.Join(dir, "links", "link.json")
			mkdir(t, filepath.Dir(link))
			return symlink(t, "../data/target.json", link), filepath.Join(dir, "data", "target.json")
		}, ""},
		{"relative through a symlinked dir", func(t *testing.T, dir string) (string, string) {
			mkdir(t, filepath.Join(dir, "a", "b"))
			symlink(t, filepath.Join(dir, "a", "b"), filepath.Join(dir, "short"))
			// ".." goes up from a/b, where the link really is, not from short.
			link := symlink(t, "../target.json", filepath.Join(dir, "a", "b", "link.json"))
			return filepath.Join(dir, "short", filepath.Base(link)), filepath.Join(dir, "a", "target.json")
		}, ""},
		{"chain", func(t *testing.T, dir string) (string, string) {
			symlink(t, "c.json", filepath.Join(dir, "b.json"))
			symlink(t, filepath.Join(dir, "sub", "d.json"), filepath.Join(dir, "c.json"))
			return symlink(t, "b.json", filepath.Join(dir, "a.json")), filepath.Join(dir, "sub", "d.json")
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			dest, target := tt.setup(t, dir)
			src := filepath.Join(dir, "template")
			if err := os.WriteFile(src, []byte(`{{ keep "a" "x" }}`), 0o644); err != nil {
				t.Fatal(err)
			}
			err := RenderConfig(src, dest)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
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

// Rendering over the result of a render changes nothing: what keep and
// keepTable write is read back as the same value.
func TestRenderConfigIdempotent(t *testing.T) {
	tests := []struct {
		name     string
		destName string
		dest     *string
		tmpl     string
	}{
		{"json", "s.json", ptr(`{"a": {"z": [1, "x\"y"], "b": 2.50}, "workbench.colorTheme": "Nord"}`),
			"{\"a\": {{ keep \"a\" \"\" }}, \"workbench.colorTheme\": {{ keep `\"workbench.colorTheme\"` \"Default\" }}}"},
		{"toml", "s.toml", ptr(starshipDest),
			"palette = {{ keep \"palette\" \"default\" }}\nlist = {{ keep \"palettes.custom.list\" \"\" }}\n# end\n{{- keepTable \"palettes.custom\" }}\n\n[other]\nx = 1\n"},
		{"toml no table", "s.toml", nil,
			"palette = {{ keep \"palette\" \"default\" }}\n# end\n{{- keepTable \"palettes.custom\" }}\n\n[other]\nx = 1\n"},
		{"yaml", "s.yaml", ptr("theme:\n  name: wal\n  list: [1, {b: 2, a: \"x\"}]\n"),
			"theme:\n  name: {{ keep \"theme.name\" \"dark\" }}\n  list: {{ keep \"theme.list\" \"\" }}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest, err := renderInDir(t, tt.destName, tt.dest, tt.tmpl)
			if err != nil {
				t.Fatal(err)
			}
			first := readRC(t, dest)
			src := filepath.Join(filepath.Dir(filepath.Dir(dest)), "template")
			if err := RenderConfig(src, dest); err != nil {
				t.Fatal(err)
			}
			if second := readRC(t, dest); second != first {
				t.Errorf("second render differs:\nfirst:\n%s\nsecond:\n%s", first, second)
			}
		})
	}
}

// The templates the packages ship render on a new host and keep what
// wal-picker wrote on a themed one, and re-rendering changes nothing.
func TestShippedTemplates(t *testing.T) {
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
		{"ulauncher wal-picker", "ubuntu/configs/ulauncher.json.tmpl", "settings.json",
			ptr(`{"show-indicator-icon": false, "theme-name": "wal-picker"}`),
			[]string{`"theme-name": "wal-picker"`}, checkJSON},
		{"starship new", "all/configs/starship/starship.toml.tmpl", "starship.toml", nil,
			[]string{`palette = "custom"`, "[palettes.custom]\nmain_color = \"#A08AE2\"\nsecondary_color = \"#A9E9B7\"\n"}, checkTOML},
		{"starship wal-picker", "all/configs/starship/starship.toml.tmpl", "starship.toml",
			ptr("palette = \"custom\"\n\n[palettes.custom]\nmain_color = \"#112233\"\nsecondary_color = \"#445566\"\n"),
			[]string{`palette = "custom"`, "[palettes.custom]\nmain_color = \"#112233\"\nsecondary_color = \"#445566\"\n"}, checkTOML},
		{"herdr new", "all/configs/herdr/config.toml.tmpl", "config.toml", nil,
			[]string{`name = "gruvbox"`, "# text = \"#cdd6f4\"\n\n[terminal]\n"}, checkHerdr(false)},
		{"herdr wal-picker", "all/configs/herdr/config.toml.tmpl", "config.toml",
			ptr("onboarding = false\n\n[theme]\nname = \"terminal\"\nauto_switch = false\n\n" +
				"[theme.custom]\naccent = \"#ff79c6\"\nred = \"#ff5555\"\n\n" +
				"[theme.custom.light]\npanel_bg = \"#eff1f5\"\n\n[theme.custom.dark]\npanel_bg = \"#1e1e2e\"\n\n[terminal]\ndefault_shell = \"/bin/zsh\"\n"),
			[]string{`name = "terminal"`, "# text = \"#cdd6f4\"\n\n[theme.custom]\naccent = \"#ff79c6\"\nred = \"#ff5555\"\n\n" +
				"[theme.custom.dark]\npanel_bg = \"#1e1e2e\"\n\n[theme.custom.light]\npanel_bg = \"#eff1f5\"\n\n[terminal]\n"},
			checkHerdr(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := os.ReadFile(filepath.Join(pkgs, tt.src))
			if err != nil {
				t.Fatal(err)
			}
			dest, err := renderInDir(t, tt.destName, tt.dest, string(tmpl))
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
			src := filepath.Join(filepath.Dir(filepath.Dir(dest)), "template")
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

// checkHerdr parses the herdr config and checks the theme: wal-picker's
// colors when themed, no custom colors otherwise.
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
		if !themed {
			if cfg.Theme.Custom != nil {
				t.Errorf("theme.custom = %v, want none", cfg.Theme.Custom)
			}
			return
		}
		dark, _ := cfg.Theme.Custom["dark"].(map[string]any)
		if cfg.Theme.Custom["accent"] != "#ff79c6" || dark["panel_bg"] != "#1e1e2e" {
			t.Errorf("theme.custom = %v", cfg.Theme.Custom)
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
	dest, err := renderInDir(t, "c.toml", ptr("[t]\n"+values), "{{ keep \"t.odt\" \"\" }} {{ keep \"t.odtz\" \"\" }} {{ keep \"t.ldt\" \"\" }} {{ keep \"t.ld\" \"\" }} {{ keep \"t.lt\" \"\" }}\n{{- keepTable \"t\" }}\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "1979-05-27T00:32:00-07:00 1979-05-27T23:32:00.5Z 1979-05-27T00:32:00.100 1979-05-27 00:32:00.999999\n\n" +
		"[t]\nld = 1979-05-27\nldt = 1979-05-27T00:32:00.100\nlt = 00:32:00.999999\nodt = 1979-05-27T00:32:00-07:00\nodtz = 1979-05-27T23:32:00.5Z\n"
	if got := readRC(t, dest); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
