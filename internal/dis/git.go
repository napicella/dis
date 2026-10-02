package dis

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// PullAction is what CloneOrUpdate did with a repo.
type PullAction string

const (
	PullCloned   PullAction = "cloned"
	PullUpdated  PullAction = "updated"
	PullUpToDate PullAction = "up to date"
	PullSkipped  PullAction = "skipped"
)

// PullResult reports the outcome of CloneOrUpdate for one repo.
type PullResult struct {
	Action PullAction
	// Detail explains a skip, or adds context (e.g. "ahead by 2").
	Detail string
}

// CheckGit returns an error when the git CLI is not on PATH. dis shells out
// to git so that the user's SSH config, keys and credential helpers apply.
func CheckGit() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not on PATH: dis pull needs it to clone repos")
	}
	return nil
}

// CloneOrUpdate makes sure repo is present at repo.Path:
//
//   - missing: git clone [--branch ref] url path
//   - present: git fetch, then fast-forward only when the work tree is clean,
//     on a branch, tracking an upstream, and not diverged. Anything else is
//     skipped with a reason, so local work is never touched.
//
// It fails when the path exists but is not a git repo, or when its origin
// does not match repo.URL. git's own output (progress, auth prompts) goes to
// out, and stdin is inherited so credential prompts work.
func CloneOrUpdate(repo ResolvedRepo, out io.Writer) (PullResult, error) {
	if _, err := os.Stat(repo.Path); os.IsNotExist(err) {
		if repo.URL == "" {
			return PullResult{}, fmt.Errorf("%s does not exist and repo %q has no url to clone from", repo.Path, repo.Name)
		}
		if err := os.MkdirAll(filepath.Dir(repo.Path), 0o755); err != nil {
			return PullResult{}, err
		}
		args := []string{"clone"}
		if repo.Ref != "" {
			args = append(args, "--branch", repo.Ref)
		}
		args = append(args, repo.URL, repo.Path)
		if err := gitInteractive(out, "", args...); err != nil {
			return PullResult{}, fmt.Errorf("cloning %s: %w", repo.URL, err)
		}
		return PullResult{Action: PullCloned}, nil
	} else if err != nil {
		return PullResult{}, err
	}

	if top, err := git(repo.Path, "rev-parse", "--show-toplevel"); err != nil || !samePath(top, repo.Path) {
		return PullResult{}, fmt.Errorf("%s exists but is not the root of a git repo", repo.Path)
	}
	if repo.URL != "" {
		origin, err := git(repo.Path, "remote", "get-url", "origin")
		if err != nil {
			return PullResult{}, fmt.Errorf("%s has no origin remote (want %s)", repo.Path, repo.URL)
		}
		if normalizeGitURL(origin) != normalizeGitURL(repo.URL) {
			return PullResult{}, fmt.Errorf("%s is a clone of %s, not %s", repo.Path, origin, repo.URL)
		}
	}

	if err := gitInteractive(out, repo.Path, "fetch", "--quiet"); err != nil {
		return PullResult{}, fmt.Errorf("fetching %s: %w", repo.Path, err)
	}
	if status, err := git(repo.Path, "status", "--porcelain"); err != nil {
		return PullResult{}, err
	} else if status != "" {
		return PullResult{Action: PullSkipped, Detail: "uncommitted changes"}, nil
	}
	if _, err := git(repo.Path, "symbolic-ref", "-q", "HEAD"); err != nil {
		return PullResult{Action: PullSkipped, Detail: "detached HEAD"}, nil
	}
	if _, err := git(repo.Path, "rev-parse", "--abbrev-ref", "@{u}"); err != nil {
		return PullResult{Action: PullSkipped, Detail: "branch has no upstream"}, nil
	}
	counts, err := git(repo.Path, "rev-list", "--left-right", "--count", "HEAD...@{u}")
	if err != nil {
		return PullResult{}, err
	}
	ahead, behind, err := parseAheadBehind(counts)
	if err != nil {
		return PullResult{}, err
	}
	switch {
	case ahead > 0 && behind > 0:
		return PullResult{Action: PullSkipped, Detail: fmt.Sprintf("diverged from upstream (%d ahead, %d behind)", ahead, behind)}, nil
	case behind > 0:
		if _, err := git(repo.Path, "merge", "--ff-only", "--quiet", "@{u}"); err != nil {
			return PullResult{}, fmt.Errorf("fast-forwarding %s: %w", repo.Path, err)
		}
		return PullResult{Action: PullUpdated, Detail: fmt.Sprintf("%d new commits", behind)}, nil
	case ahead > 0:
		return PullResult{Action: PullUpToDate, Detail: fmt.Sprintf("ahead by %d, not pushed", ahead)}, nil
	default:
		return PullResult{Action: PullUpToDate}, nil
	}
}

func parseAheadBehind(s string) (int, int, error) {
	f := strings.Fields(s)
	if len(f) != 2 {
		return 0, 0, fmt.Errorf("unexpected git rev-list output %q", s)
	}
	a, err1 := strconv.Atoi(f[0])
	b, err2 := strconv.Atoi(f[1])
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("unexpected git rev-list output %q", s)
	}
	return a, b, nil
}

// samePath reports whether a and b name the same directory, following symlinks
// (e.g. /home -> /local/home on Cloud Desktops).
func samePath(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return ra == rb
}

// normalizeGitURL makes URLs comparable: trailing slashes and ".git" are ignored.
func normalizeGitURL(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimRight(u, "/")
	return strings.TrimSuffix(u, ".git")
}

// RepoNameFromURL returns the last path element of a git URL without ".git",
// e.g. "git@github.com:napicella/dotfiles.git" -> "dotfiles".
func RepoNameFromURL(u string) string {
	u = normalizeGitURL(u)
	if i := strings.LastIndexAny(u, "/:"); i >= 0 {
		u = u[i+1:]
	}
	return u
}

// git runs a non-interactive git command in dir and returns trimmed stdout.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(b)), nil
}

// gitInteractive runs a git command that may talk to a remote: its output goes
// to out and stdin is inherited, so SSH and credential prompts work.
func gitInteractive(out io.Writer, dir string, args ...string) error {
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// RepoStatus is the local state of a clone, as git knows it without fetching.
type RepoStatus struct {
	// Missing is true when the clone's directory does not exist.
	Missing bool
	// Dirty is true when the work tree has uncommitted or untracked changes.
	Dirty bool
	// Branch is the checked-out branch; empty on a detached HEAD.
	Branch string
	// HasUpstream is true when Branch tracks a remote branch.
	HasUpstream bool
	// Ahead and Behind count commits relative to the upstream, as of the last
	// fetch.
	Ahead, Behind int
}

// GetRepoStatus reports the state of the clone at path. It does not fetch, so
// Behind only counts commits fetched earlier (e.g. by 'dis pull').
func GetRepoStatus(path string) (RepoStatus, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return RepoStatus{Missing: true}, nil
	} else if err != nil {
		return RepoStatus{}, err
	}
	var st RepoStatus
	status, err := git(path, "status", "--porcelain")
	if err != nil {
		return RepoStatus{}, err
	}
	st.Dirty = status != ""
	if branch, err := git(path, "symbolic-ref", "-q", "--short", "HEAD"); err == nil {
		st.Branch = branch
	} else {
		return st, nil
	}
	if _, err := git(path, "rev-parse", "--abbrev-ref", "@{u}"); err != nil {
		return st, nil
	}
	st.HasUpstream = true
	counts, err := git(path, "rev-list", "--left-right", "--count", "HEAD...@{u}")
	if err != nil {
		return RepoStatus{}, err
	}
	st.Ahead, st.Behind, err = parseAheadBehind(counts)
	return st, err
}
