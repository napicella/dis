package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
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

// searchFixture writes a distro whose single source provides common/git and
// common/app (which references app.conf through $DIS_CONFIG_FOLDER) and returns
// the distro file.
func searchFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	src := t.TempDir()
	files := map[string]string{
		"dis.ws.yml":       "packages:\n  - root: ./installers\n    configs: ./configs\n",
		"configs/app.conf": "",
		"installers/git.sh": "### -- Manifest\n### provides: common/git\n### depends_on: []\n### distro: [all]\n### -- End\n" +
			"alias status='git status'\n",
		"installers/app.sh": "### -- Manifest\n### provides: common/app\n### depends_on: []\n### distro: [all]\n### -- End\n" +
			"cp $DIS_CONFIG_FOLDER/app.conf ~/\n",
	}
	for name, body := range files {
		path := filepath.Join(src, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	distro := filepath.Join(t.TempDir(), "distro.yml")
	if err := os.WriteFile(distro, []byte("os: ubuntu\nsources:\n  - "+src+"\npackages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return distro
}

// runCmd executes the root command with args and returns its stdout and error.
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs(args)
	defer rootCmd.SetArgs(nil)
	err := rootCmd.Execute()
	return out.String(), err
}

func TestSearchSubcommands(t *testing.T) {
	distro := searchFixture(t)

	tests := []struct {
		name    string
		args    []string
		want    []string // packages in the output, in order
		wantErr string
	}{
		{"packages", []string{"search", "packages", "git"}, []string{"common/git"}, ""},
		{"packages regex", []string{"search", "packages", "^common/"}, []string{"common/app", "common/git"}, ""},
		{"installers", []string{"search", "installers", "status", "--package", ""}, []string{"common/git"}, ""},
		{"installers narrowed by package", []string{"search", "installers", "status", "--package", "app"}, nil, "no matches"},
		{"configs by path", []string{"search", "configs", `app\.conf`, "--package", ""}, []string{"common/app"}, ""},
		{"configs narrowed by package", []string{"search", "configs", ".", "--package", "git"}, nil, "no matches"},
		{"no subcommand", []string{"search"}, nil, "say what to search"},
		{"a regex is not a subcommand", []string{"search", "git"}, nil, `unknown command "git"`},
		{"missing regex", []string{"search", "packages"}, nil, "accepts 1 arg(s), received 0"},
		{"extra argument", []string{"search", "installers", "a", "b"}, nil, "accepts 1 arg(s), received 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			if len(args) > 2 { // the parent command takes no flags
				args = append(args, "--distro", distro)
			}
			out, err := runCmd(t, args...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var got []string
			for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
				got = append(got, strings.Fields(line)[0])
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("packages = %v, want %v\noutput:\n%s", got, tt.want, out)
			}
		})
	}

	t.Run("json", func(t *testing.T) {
		out, err := runCmd(t, "search", "installers", "status", "--package", "", "--json", "--distro", distro)
		if err != nil {
			t.Fatal(err)
		}
		var got []searchResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
		// Line counts from the top of the file, manifest included.
		if len(got) != 1 || got[0].Package != "common/git" || got[0].Line != 6 {
			t.Errorf("got %+v, want common/git line 6", got)
		}
		// Reset the flag: cobra keeps flag values across Execute calls.
		_ = searchInstallersCmd.Flags().Set("json", "false")
	})
}
