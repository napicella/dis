package dis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRepos(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	distroDir := t.TempDir() // not a git repo: self falls back to it

	cfg := DistroConfig{Repos: map[string]Repo{
		"dotfiles": {URL: "git@github.com:me/dotfiles.git"},
		"dis":      {URL: "https://github.com/me/dis.git", Path: "~/github/dis", Ref: "v1"},
		"abs":      {URL: "u", Path: "/opt/abs"},
		"rel":      {URL: "u", Path: "src/rel"},
		"vars":     {URL: "u", Path: "${home}/v"},
	}}
	repos, err := ResolveRepos(cfg, distroDir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]ResolvedRepo{}
	var order []string
	for _, r := range repos {
		got[r.Name] = r
		order = append(order, r.Name)
	}
	if want := "self,abs,dis,dotfiles,rel,vars"; strings.Join(order, ",") != want {
		t.Errorf("order = %s, want %s", strings.Join(order, ","), want)
	}
	for name, want := range map[string]string{
		"self":     distroDir,
		"dotfiles": filepath.Join(home, "dotfiles"),
		"dis":      filepath.Join(home, "github/dis"),
		"abs":      "/opt/abs",
		"rel":      filepath.Join(home, "src/rel"),
		"vars":     filepath.Join(home, "v"),
	} {
		if got[name].Path != want {
			t.Errorf("%s path = %q, want %q", name, got[name].Path, want)
		}
	}
	if !got["self"].Implicit || got["dis"].Implicit {
		t.Errorf("only self should be implicit")
	}
	if got["dis"].Ref != "v1" || got["dis"].URL != "https://github.com/me/dis.git" {
		t.Errorf("dis = %+v", got["dis"])
	}
}

func TestResolveReposErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for name, repos := range map[string]map[string]Repo{
		"self declared": {"self": {URL: "u"}},
		"missing url":   {"dotfiles": {}},
		"invalid name":  {"a/b": {URL: "u"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ResolveRepos(DistroConfig{Repos: repos}, t.TempDir()); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestResolveReposSelfIsGitToplevel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "remote", "add", "origin", "git@example.com:me/distro.git")
	distroDir := filepath.Join(repo, "distros", "laptop")
	if err := os.MkdirAll(distroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repos, err := ResolveRepos(DistroConfig{}, distroDir)
	if err != nil {
		t.Fatal(err)
	}
	self := repos[0]
	if !samePath(self.Path, repo) {
		t.Errorf("self path = %q, want %q", self.Path, repo)
	}
	if self.URL != "git@example.com:me/distro.git" {
		t.Errorf("self url = %q", self.URL)
	}
}

func TestVarExpander(t *testing.T) {
	home := t.TempDir()
	cloned := filepath.Join(home, "dotfiles")
	if err := os.MkdirAll(cloned, 0o755); err != nil {
		t.Fatal(err)
	}
	e := newVarExpander(home, []ResolvedRepo{
		{Name: "dotfiles", Path: cloned},
		{Name: "missing", Path: filepath.Join(home, "missing")},
	})

	got, err := e.expand("${repos.dotfiles}/bashrc:${home}/x")
	if err != nil {
		t.Fatal(err)
	}
	if want := cloned + "/bashrc:" + home + "/x"; got != want {
		t.Errorf("expand = %q, want %q", got, want)
	}

	for in, wantErr := range map[string]string{
		"${repos.nope}/x":    `repo "nope" is not declared`,
		"${repos.missing}/x": `repo "missing" is not cloned`,
	} {
		if _, err := e.expand(in); err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("expand(%q) error = %v, want it to contain %q", in, err, wantErr)
		}
	}
}

func TestNewInstallContextExpandsRepos(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	src := filepath.Join(home, "dotfiles", "tools")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	installer := "### -- Manifest\n### provides: tools/hello\n### depends_on: []\n### distro: [all]\n### -- End\necho hi\n"
	if err := os.WriteFile(filepath.Join(src, "hello.sh"), []byte(installer), 0o644); err != nil {
		t.Fatal(err)
	}
	distro := filepath.Join(t.TempDir(), "distro.yml")
	yml := `os: ubuntu
repos:
  dotfiles: { url: git@github.com:me/dotfiles.git }
sources:
  - ${repos.dotfiles}/tools
parameters:
  TOOLS_DIR: ${repos.dotfiles}/tools
packages:
  - tools/hello
`
	if err := os.WriteFile(distro, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	ic, err := NewInstallContext(distro)
	if err != nil {
		t.Fatal(err)
	}
	if ic.parameters["TOOLS_DIR"] != src {
		t.Errorf("TOOLS_DIR = %q, want %q", ic.parameters["TOOLS_DIR"], src)
	}
	order, err := ic.ResolveInstallOrder()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 1 || order[0].Provides != "tools/hello" {
		t.Errorf("install order = %+v", order)
	}
}
