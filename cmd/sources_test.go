package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/napicella/dis/internal/dis"
)

func TestSourceRows(t *testing.T) {
	root := t.TempDir()
	dotfiles := filepath.Join(root, "dotfiles")
	for _, d := range []string{"tools", "tools-exp"} {
		if err := os.MkdirAll(filepath.Join(dotfiles, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	repos := []dis.ResolvedRepo{
		{Name: "self", Path: root},
		{Name: "dotfiles", Path: dotfiles},
	}
	sources := []dis.ResolvedSource{
		{Declared: "${repos.dotfiles}/tools", Path: filepath.Join(dotfiles, "tools")},
		{Declared: "${repos.dotfiles}/tools-exp", Path: filepath.Join(dotfiles, "tools-exp")},
		{Declared: "/opt/elsewhere", Path: "/opt/elsewhere"},
	}
	pkgs := []dis.PackageInfo{
		{Provides: "tools/a", InstallerPath: filepath.Join(dotfiles, "tools", "a", "install.sh")},
		{Provides: "tools/b", InstallerPath: filepath.Join(dotfiles, "tools", "b.sh")},
		// "tools-exp" shares a prefix with "tools" and must not count for it.
		{Provides: "tools/c", InstallerPath: filepath.Join(dotfiles, "tools-exp", "c.sh")},
	}

	got := sourceRows(sources, repos, pkgs)
	want := []sourceRow{
		// The deepest containing repo wins over self, which contains everything.
		{Declared: "${repos.dotfiles}/tools", Path: filepath.Join(dotfiles, "tools"), Repo: "dotfiles", Exists: true, Packages: 2},
		{Declared: "${repos.dotfiles}/tools-exp", Path: filepath.Join(dotfiles, "tools-exp"), Repo: "dotfiles", Exists: true, Packages: 1},
		{Declared: "/opt/elsewhere", Path: "/opt/elsewhere", Repo: "", Exists: false, Packages: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sourceRows =\n%+v\nwant\n%+v", got, want)
	}
}

func TestWithin(t *testing.T) {
	tests := []struct {
		path, dir string
		want      bool
	}{
		{"/a/b", "/a/b", true},
		{"/a/b/c.sh", "/a/b", true},
		{"/a/bc/d.sh", "/a/b", false},
		{"/a", "/a/b", false},
		{"/x/..y", "/x", true}, // a name starting with ".." is still inside
	}
	for _, tt := range tests {
		if got := within(tt.path, tt.dir); got != tt.want {
			t.Errorf("within(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
		}
	}
}

func TestSourcesCommand(t *testing.T) {
	distro := searchFixture(t)

	out, err := runCmd(t, "sources", "--json", "--distro", distro)
	_ = sourcesCmd.Flags().Set("json", "false") // cobra keeps flag values across Execute calls
	if err != nil {
		t.Fatal(err)
	}
	var rows []sourceRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if len(rows) != 1 || !rows[0].Exists || rows[0].Packages != 2 {
		t.Errorf("rows = %+v, want one existing source with 2 packages", rows)
	}

	// The sources used to be a mode of 'dis list'.
	if _, err := runCmd(t, "list", "--sources"); err == nil || !strings.Contains(err.Error(), "unknown flag: --sources") {
		t.Errorf("dis list --sources: error = %v, want unknown flag", err)
	}
}
