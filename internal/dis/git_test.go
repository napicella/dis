package dis

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// newUpstream creates a bare repo with one commit on main and returns its path
// plus a scratch clone used to push more commits to it.
func newUpstream(t *testing.T) (bare, pusher string) {
	t.Helper()
	root := t.TempDir()
	bare = filepath.Join(root, "upstream.git")
	pusher = filepath.Join(root, "pusher")
	runGit(t, root, "init", "-q", "--bare", "-b", "main", bare)
	runGit(t, root, "clone", "-q", bare, pusher)
	commit(t, pusher, "a.txt")
	runGit(t, pusher, "push", "-q", "origin", "HEAD:main")
	return bare, pusher
}

func commit(t *testing.T, dir, file string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", file)
	runGit(t, dir, "commit", "-q", "-m", file)
}

func pushNew(t *testing.T, pusher, file string) {
	t.Helper()
	commit(t, pusher, file)
	runGit(t, pusher, "push", "-q", "origin", "HEAD:main")
}

func TestCloneOrUpdate(t *testing.T) {
	if err := CheckGit(); err != nil {
		t.Skip(err)
	}
	bare, pusher := newUpstream(t)
	path := filepath.Join(t.TempDir(), "nested", "clone")
	repo := ResolvedRepo{Name: "r", URL: bare, Path: path}

	pull := func() PullResult {
		t.Helper()
		res, err := CloneOrUpdate(repo, io.Discard)
		if err != nil {
			t.Fatalf("CloneOrUpdate: %v", err)
		}
		return res
	}
	expect := func(res PullResult, action PullAction, detail string) {
		t.Helper()
		if res.Action != action || !strings.Contains(res.Detail, detail) {
			t.Errorf("got %s (%s), want %s (%s)", res.Action, res.Detail, action, detail)
		}
	}

	expect(pull(), PullCloned, "")
	expect(pull(), PullUpToDate, "")

	pushNew(t, pusher, "b.txt")
	expect(pull(), PullUpdated, "1 new commits")
	if _, err := os.Stat(filepath.Join(path, "b.txt")); err != nil {
		t.Errorf("fast-forward did not bring b.txt: %v", err)
	}

	// Local, unpushed commit: reported, not touched.
	commit(t, path, "local.txt")
	expect(pull(), PullUpToDate, "ahead by 1")

	// Diverged: local commit plus a new upstream commit.
	pushNew(t, pusher, "c.txt")
	expect(pull(), PullSkipped, "diverged")
	runGit(t, path, "reset", "-q", "--hard", "origin/main")

	// Dirty work tree: skipped even when upstream has new commits.
	pushNew(t, pusher, "d.txt")
	if err := os.WriteFile(filepath.Join(path, "a.txt"), []byte("edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	expect(pull(), PullSkipped, "uncommitted changes")
	if b, _ := os.ReadFile(filepath.Join(path, "a.txt")); string(b) != "edited" {
		t.Errorf("local edit was lost: %q", b)
	}
	runGit(t, path, "checkout", "-q", "--", "a.txt")

	// Detached HEAD.
	runGit(t, path, "checkout", "-q", "--detach")
	expect(pull(), PullSkipped, "detached HEAD")
	runGit(t, path, "checkout", "-q", "main")

	// Branch without upstream.
	runGit(t, path, "checkout", "-q", "-b", "local-only")
	expect(pull(), PullSkipped, "no upstream")
}

func TestCloneOrUpdateErrors(t *testing.T) {
	if err := CheckGit(); err != nil {
		t.Skip(err)
	}
	bare, _ := newUpstream(t)
	other, _ := newUpstream(t)

	path := filepath.Join(t.TempDir(), "clone")
	if _, err := CloneOrUpdate(ResolvedRepo{Name: "r", URL: bare, Path: path}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := CloneOrUpdate(ResolvedRepo{Name: "r", URL: other, Path: path}, io.Discard); err == nil || !strings.Contains(err.Error(), "is a clone of") {
		t.Errorf("different remote: err = %v", err)
	}

	plain := t.TempDir()
	if _, err := CloneOrUpdate(ResolvedRepo{Name: "r", URL: bare, Path: plain}, io.Discard); err == nil || !strings.Contains(err.Error(), "not the root of a git repo") {
		t.Errorf("non-git dir: err = %v", err)
	}
}

func TestCloneWithRef(t *testing.T) {
	if err := CheckGit(); err != nil {
		t.Skip(err)
	}
	bare, pusher := newUpstream(t)
	runGit(t, pusher, "tag", "v1")
	runGit(t, pusher, "push", "-q", "origin", "v1")
	pushNew(t, pusher, "after-tag.txt")

	path := filepath.Join(t.TempDir(), "clone")
	if _, err := CloneOrUpdate(ResolvedRepo{Name: "r", URL: bare, Path: path, Ref: "v1"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, "after-tag.txt")); !os.IsNotExist(err) {
		t.Errorf("clone at ref v1 should not contain after-tag.txt")
	}
}

func TestRepoNameFromURL(t *testing.T) {
	for in, want := range map[string]string{
		"git@github.com:you/dotfiles.git":         "dotfiles",
		"https://github.com/you/dis.git":          "dis",
		"https://github.com/you/dis/":             "dis",
		"ssh://git.example.com:2222/pkg/MyBashrc": "MyBashrc",
		"/tmp/upstream.git":                       "upstream",
	} {
		if got := RepoNameFromURL(in); got != want {
			t.Errorf("RepoNameFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGetRepoStatus(t *testing.T) {
	if err := CheckGit(); err != nil {
		t.Skip(err)
	}
	bare, pusher := newUpstream(t)
	clone := filepath.Join(t.TempDir(), "clone")
	runGit(t, filepath.Dir(clone), "clone", "-q", bare, clone)

	st, err := GetRepoStatus(clone)
	if err != nil {
		t.Fatal(err)
	}
	if want := (RepoStatus{Branch: "main", HasUpstream: true}); st != want {
		t.Errorf("clean clone: got %+v, want %+v", st, want)
	}

	commit(t, clone, "local.txt")
	pushNew(t, pusher, "remote.txt")
	runGit(t, clone, "fetch", "-q")
	if err := os.WriteFile(filepath.Join(clone, "untracked"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	st, err = GetRepoStatus(clone)
	if err != nil {
		t.Fatal(err)
	}
	if want := (RepoStatus{Dirty: true, Branch: "main", HasUpstream: true, Ahead: 1, Behind: 1}); st != want {
		t.Errorf("diverged dirty clone: got %+v, want %+v", st, want)
	}

	st, err = GetRepoStatus(filepath.Join(t.TempDir(), "absent"))
	if err != nil || !st.Missing {
		t.Errorf("absent clone: got %+v, %v", st, err)
	}
}
