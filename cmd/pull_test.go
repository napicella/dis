package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetConfigDistro(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".config", "dis", "config.yaml")

	// No config file yet: it is created.
	old, err := setConfigDistro("/a/distro.yml")
	if err != nil {
		t.Fatal(err)
	}
	if old != "" {
		t.Errorf("old = %q, want empty", old)
	}
	if b, _ := os.ReadFile(cfgPath); strings.TrimSpace(string(b)) != "distro: /a/distro.yml" {
		t.Errorf("config = %q", b)
	}

	// Existing file: distro is replaced, other keys and comments are kept.
	if err := os.WriteFile(cfgPath, []byte("# my config\ndistro: ~/old.yml # pinned\nother: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old, err = setConfigDistro("/b/distro.yml")
	if err != nil {
		t.Fatal(err)
	}
	if old != filepath.Join(home, "old.yml") {
		t.Errorf("old = %q, want the expanded previous value", old)
	}
	b, _ := os.ReadFile(cfgPath)
	for _, want := range []string{"# my config", "distro: /b/distro.yml", "other: 1"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("config %q does not contain %q", b, want)
		}
	}
}

func TestPickDistro(t *testing.T) {
	repo := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const distro = "os: ubuntu\npackages:\n  - a/b\n"
	write("distros/laptop/laptop.yml", distro)
	write("distros/laptop/dis.ws.yml", "packages:\n  - root: ./x\n") // workspace, not a distro
	write("distros/server/configs/app.yml", "key: value\n")          // arbitrary YAML
	write(".hidden/distro.yml", distro)                              // hidden dirs are skipped

	got, err := pickDistro(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(repo, "distros/laptop/laptop.yml"); got != want {
		t.Errorf("pickDistro = %q, want %q", got, want)
	}

	write("distros/server/server.yml", distro)
	if _, err := pickDistro(repo, ""); err == nil || !strings.Contains(err.Error(), "distros/server/server.yml") {
		t.Errorf("two distros: err = %v, want it to list the candidates", err)
	}
	if got, err := pickDistro(repo, "distros/server/server.yml"); err != nil || got != filepath.Join(repo, "distros/server/server.yml") {
		t.Errorf("explicit distro: got %q, err %v", got, err)
	}
	if _, err := pickDistro(repo, "nope.yml"); err == nil {
		t.Error("missing explicit distro: expected an error")
	}
}
