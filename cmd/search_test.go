package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/napicella/dis/internal/dis"
)

func TestSearchCollectors(t *testing.T) {
	dir := t.TempDir()
	configs := filepath.Join(dir, "configs")
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	appConf := filepath.Join(configs, "app.conf")
	if err := os.WriteFile(appConf, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	gitSh := filepath.Join(dir, "git.sh")
	appSh := filepath.Join(dir, "app.sh")
	app2Sh := filepath.Join(dir, "app2.sh")
	if err := os.WriteFile(gitSh, []byte("alias status='git status'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	appBody := []byte("cp $DIS_CONFIG_FOLDER/app.conf ~/\ncp $DIS_CONFIG_FOLDER/app.conf ~/.config/\n")
	for _, p := range []string{appSh, app2Sh} {
		if err := os.WriteFile(p, appBody, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pkgs := []dis.PackageInfo{
		{Provides: "common/app", InstallerPath: appSh, ConfigsDir: configs},
		{Provides: "common/app2", InstallerPath: app2Sh, ConfigsDir: configs},
		{Provides: "common/git", InstallerPath: gitSh},
	}

	t.Run("packages", func(t *testing.T) {
		got := collectPackages(pkgs[2:])
		want := []searchResult{{Package: "common/git", Path: gitSh}}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("content", func(t *testing.T) {
		got, err := collectContent(pkgs, regexp.MustCompile(`status`))
		if err != nil {
			t.Fatal(err)
		}
		want := []searchResult{{Package: "common/git", Path: gitSh, Line: 1, Text: "alias status='git status'"}}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("configs are deduplicated per package", func(t *testing.T) {
		got, err := collectConfigs(pkgs)
		if err != nil {
			t.Fatal(err)
		}
		want := []searchResult{
			{Package: "common/app", Path: appConf},
			{Package: "common/app2", Path: appConf},
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("nothing found", func(t *testing.T) {
		if got, err := collectContent(pkgs, regexp.MustCompile(`nope`)); err != nil || len(got) != 0 {
			t.Errorf("content: got %v, %v; want no results", got, err)
		}
		if got, err := collectConfigs(pkgs[2:]); err != nil || len(got) != 0 {
			t.Errorf("configs: got %v, %v; want no results", got, err)
		}
	})
}

func TestSearchRenderers(t *testing.T) {
	results := []searchResult{
		{Package: "common/git", Path: "/p/git.sh", Line: 29, Text: "alias status='git status'"},
		{Package: "common/starship", Path: "/p/starship.toml"},
	}

	t.Run("rows", func(t *testing.T) {
		var out bytes.Buffer
		if err := renderRows(&out, results); err != nil {
			t.Fatal(err)
		}
		want := "common/git       /p/git.sh:29  alias status='git status'\n" +
			"common/starship  /p/starship.toml\n"
		if out.String() != want {
			t.Errorf("got\n%s\nwant\n%s", out.String(), want)
		}
	})

	t.Run("json omits line and text outside content matches", func(t *testing.T) {
		var out bytes.Buffer
		if err := renderJSON(&out, results); err != nil {
			t.Fatal(err)
		}
		want := `[
  {
    "package": "common/git",
    "path": "/p/git.sh",
    "line": 29,
    "text": "alias status='git status'"
  },
  {
    "package": "common/starship",
    "path": "/p/starship.toml"
  }
]
`
		if out.String() != want {
			t.Errorf("got\n%s\nwant\n%s", out.String(), want)
		}
	})

	t.Run("empty json is an empty array", func(t *testing.T) {
		var out bytes.Buffer
		if err := renderJSON(&out, nil); err != nil {
			t.Fatal(err)
		}
		if out.String() != "[]\n" {
			t.Errorf("got %q, want %q", out.String(), "[]\n")
		}
	})
}
