package dis

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mitchellh/go-homedir"
)

// SelfRepo is the implicit repo name for the git repository that contains the
// distro file.
const SelfRepo = "self"

var repoVarPattern = regexp.MustCompile(`\$\{repos\.([^}]*)\}`)

// ResolvedRepo is a repo from a distro's repos: map (or the implicit self repo)
// with its path resolved to an absolute directory.
type ResolvedRepo struct {
	Name string
	URL  string
	Ref  string
	// Path is the absolute directory of the local clone.
	Path string
	// Implicit is true for the self repo, which is not declared in repos:.
	Implicit bool
}

// ResolveRepos returns the distro's declared repos plus the implicit self repo,
// sorted by name with self first.
func ResolveRepos(cfg DistroConfig, distroDir string) ([]ResolvedRepo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}
	if _, ok := cfg.Repos[SelfRepo]; ok {
		return nil, fmt.Errorf("repos: %q is reserved for the repo that contains the distro file and cannot be declared", SelfRepo)
	}

	resolve := func(p string) (string, error) {
		p = strings.ReplaceAll(p, "${home}", home)
		p, err := homedir.Expand(p)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(home, p)
		}
		return filepath.Clean(p), nil
	}

	selfPath := gitToplevel(distroDir)
	repos := []ResolvedRepo{{
		Name:     SelfRepo,
		URL:      gitRemoteURL(selfPath),
		Path:     selfPath,
		Implicit: true,
	}}

	names := make([]string, 0, len(cfg.Repos))
	for name := range cfg.Repos {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		r := cfg.Repos[name]
		if !validRepoName(name) {
			return nil, fmt.Errorf("repos: invalid repo name %q (use letters, digits, - and _)", name)
		}
		if r.URL == "" {
			return nil, fmt.Errorf("repos.%s: url is required", name)
		}
		p := r.Path
		if p == "" {
			p = filepath.Join(home, name)
		}
		abs, err := resolve(p)
		if err != nil {
			return nil, fmt.Errorf("repos.%s: resolving path %q: %w", name, p, err)
		}
		repos = append(repos, ResolvedRepo{Name: name, URL: r.URL, Ref: r.Ref, Path: abs})
	}
	return repos, nil
}

func validRepoName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if !(c == '-' || c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// varExpander expands ${home} and ${repos.<name>} in distro values.
type varExpander struct {
	home  string
	repos map[string]string
}

func newVarExpander(home string, repos []ResolvedRepo) *varExpander {
	m := make(map[string]string, len(repos))
	for _, r := range repos {
		m[r.Name] = r.Path
	}
	return &varExpander{home: home, repos: m}
}

// expand replaces ${home} and every ${repos.<name>} in s. It fails when s
// references an undeclared repo or a repo that has not been cloned yet.
func (e *varExpander) expand(s string) (string, error) {
	s = strings.ReplaceAll(s, "${home}", e.home)
	var expandErr error
	out := repoVarPattern.ReplaceAllStringFunc(s, func(m string) string {
		name := repoVarPattern.FindStringSubmatch(m)[1]
		p, ok := e.repos[name]
		if !ok {
			if expandErr == nil {
				expandErr = fmt.Errorf("repo %q is not declared in repos:", name)
			}
			return m
		}
		if _, err := os.Stat(p); err != nil {
			if expandErr == nil {
				expandErr = fmt.Errorf("repo %q is not cloned (expected at %s): run 'dis pull'", name, p)
			}
			return m
		}
		return p
	})
	return out, expandErr
}

// gitToplevel returns the root of the git work tree containing dir: the nearest
// ancestor with a .git entry (a directory, or a file for worktrees and
// submodules). It returns dir itself when dir is not inside a git repo. It
// does not need the git CLI, which only 'dis pull' requires.
func gitToplevel(dir string) string {
	for d := dir; ; {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir
		}
		d = parent
	}
}

// gitRemoteURL returns the origin URL of the repo at dir, or "" if there is none.
func gitRemoteURL(dir string) string {
	u, err := git(dir, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return u
}
