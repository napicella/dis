package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/napicella/dis/internal/dis"
)

func TestResolveEditor(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"DIS_EDITOR wins", map[string]string{"DIS_EDITOR": "code --wait", "VISUAL": "vim", "EDITOR": "nano"}, "code --wait"},
		{"VISUAL before EDITOR", map[string]string{"VISUAL": "vim", "EDITOR": "nano"}, "vim"},
		{"EDITOR alone", map[string]string{"EDITOR": "nano"}, "nano"},
		{"empty values are skipped", map[string]string{"DIS_EDITOR": "", "VISUAL": "", "EDITOR": "nano"}, "nano"},
		{"vi fallback", map[string]string{}, "vi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(k string) string { return tt.env[k] }
			if got := resolveEditor(getenv); got != tt.want {
				t.Errorf("resolveEditor = %q, want %q", got, tt.want)
			}
		})
	}
}

// runEditor must split the editor value into words (so it can carry flags)
// while passing each file as a single argument, even with spaces in it.
func TestRunEditorArgs(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "args")
	script := filepath.Join(dir, "fake-editor")
	body := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + out + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	files := []string{"/tmp/plain.toml", "/tmp/with space.conf"}
	if err := runEditor(context.Background(), script+" --wait", files, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runEditor: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--wait", "/tmp/plain.toml", "/tmp/with space.conf"}
	if lines := strings.Split(strings.TrimSuffix(string(got), "\n"), "\n"); !reflect.DeepEqual(lines, want) {
		t.Errorf("editor args = %q, want %q", lines, want)
	}
}

func TestRunEditorFailure(t *testing.T) {
	if err := runEditor(context.Background(), "false", []string{"/tmp/x"}, nil, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("runEditor with a failing editor: want error, got nil")
	}
}

func TestExpandConfigFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(rel string, data []byte) string {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	top := write("starship.toml", []byte("format = \"$all\"\n"))
	a := write("conf.d/a.conf", []byte("a\n"))
	b := write("conf.d/nested/b.conf", []byte("b\n"))
	write("conf.d/wallpaper.png", []byte{0x89, 'P', 'N', 'G', 0, 0, 0})
	empty := write("conf.d/empty", nil)

	got, err := expandConfigFiles([]string{top, filepath.Join(dir, "conf.d"), top})
	if err != nil {
		t.Fatalf("expandConfigFiles: %v", err)
	}
	want := []string{top, a, empty, b}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expandConfigFiles =\n%q\nwant\n%q", got, want)
	}
}

func TestFilesToEdit(t *testing.T) {
	dir := t.TempDir()
	installer := filepath.Join(dir, "00_thing.sh")
	if err := os.WriteFile(installer, []byte("cp \"$DIS_CONFIG_FOLDER/thing.conf\" ~/.thing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configs := filepath.Join(dir, "configs")
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := filepath.Join(configs, "thing.conf")
	if err := os.WriteFile(conf, []byte("k = v\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := dis.PackageInfo{Provides: "test/thing", InstallerPath: installer, ConfigsDir: configs}

	tests := []struct {
		name                   string
		installerOnly, cfgOnly bool
		want                   []string
	}{
		{"default opens installer then configs", false, false, []string{installer, conf}},
		{"installer only", true, false, []string{installer}},
		{"configs only", false, true, []string{conf}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filesToEdit(pkg, tt.installerOnly, tt.cfgOnly)
			if err != nil {
				t.Fatalf("filesToEdit: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filesToEdit = %q, want %q", got, tt.want)
			}
		})
	}
}

// A package with no configs is still editable: the installer always exists.
func TestFilesToEditNoConfigs(t *testing.T) {
	dir := t.TempDir()
	installer := filepath.Join(dir, "00_bare.sh")
	if err := os.WriteFile(installer, []byte("echo hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := dis.PackageInfo{Provides: "test/bare", InstallerPath: installer}

	got, err := filesToEdit(pkg, false, false)
	if err != nil {
		t.Fatalf("filesToEdit: %v", err)
	}
	if want := []string{installer}; !reflect.DeepEqual(got, want) {
		t.Errorf("filesToEdit = %q, want %q", got, want)
	}

	if _, err := filesToEdit(pkg, false, true); err == nil {
		t.Error("filesToEdit with --configs and no configs: want error, got nil")
	}
}

func TestExpandConfigFilesMissing(t *testing.T) {
	if _, err := expandConfigFiles([]string{filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Fatal("expandConfigFiles with a missing path: want error, got nil")
	}
}
