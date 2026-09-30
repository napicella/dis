package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/napicella/dis/internal/dis"
)

func TestSearchPrinters(t *testing.T) {
	dir := t.TempDir()
	configs := filepath.Join(dir, "configs")
	if err := os.MkdirAll(configs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configs, "app.conf"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	gitSh := filepath.Join(dir, "git.sh")
	appSh := filepath.Join(dir, "app.sh")
	if err := os.WriteFile(gitSh, []byte("alias status='git status'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appSh, []byte("cp $DIS_CONFIG_FOLDER/app.conf ~/\ncp $DIS_CONFIG_FOLDER/app.conf ~/.config/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgs := []dis.PackageInfo{
		{Provides: "common/app", InstallerPath: appSh, ConfigsDir: configs},
		{Provides: "common/git", InstallerPath: gitSh},
	}

	t.Run("content", func(t *testing.T) {
		var out bytes.Buffer
		if err := printContentMatches(&out, pkgs, regexp.MustCompile(`status`)); err != nil {
			t.Fatal(err)
		}
		want := "common/git  " + gitSh + ":1  alias status='git status'\n"
		if out.String() != want {
			t.Errorf("got %q, want %q", out.String(), want)
		}
	})

	t.Run("configs are deduplicated", func(t *testing.T) {
		var out bytes.Buffer
		if err := printConfigs(&out, pkgs); err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(configs, "app.conf") + "\n"; out.String() != want {
			t.Errorf("got %q, want %q", out.String(), want)
		}
	})

	t.Run("nothing found", func(t *testing.T) {
		var out bytes.Buffer
		if err := printContentMatches(&out, pkgs, regexp.MustCompile(`nope`)); !errors.Is(err, errNoMatches) {
			t.Errorf("content: got %v, want errNoMatches", err)
		}
		if err := printConfigs(&out, pkgs[1:]); !errors.Is(err, errNoMatches) {
			t.Errorf("configs: got %v, want errNoMatches", err)
		}
		if err := printPackages(&out, nil); !errors.Is(err, errNoMatches) {
			t.Errorf("packages: got %v, want errNoMatches", err)
		}
	})
}
