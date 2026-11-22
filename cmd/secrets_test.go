package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// secretsFixture writes a distro with the secret ONE, needed by t/one, whose
// installer copies the secret file to $OUT/one and touches $OUT/ran, and a
// precondition that touches $OUT/pre. bws is a fake that logs each call to the
// returned log and answers only the token "good". HOME and XDG_RUNTIME_DIR are
// temp dirs, and the token file holds token.
func secretsFixture(t *testing.T, key, token string) (distro, out, fakeLog string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Chdir(t.TempDir()) // no config.yaml from the cwd
	out = t.TempDir()
	dir := t.TempDir()
	fakeLog = filepath.Join(dir, "fake.log")
	t.Setenv("FAKE_LOG", fakeLog)
	bws := filepath.Join(dir, "fake-bws")
	write(t, bws, `#!/bin/sh
echo "argv: $*" >> "$FAKE_LOG"
[ "$BWS_ACCESS_TOKEN" = good ] || { echo "Error: 401 Unauthorized" >&2; exit 1; }
echo '[{"key":"test/one","value":"v1"}]'
`, 0o755)
	t.Setenv("DIS_BWS", bws)
	write(t, filepath.Join(home, ".config", "dis", "bws-token"), token+"\n", 0o600)

	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "one.sh"), `### -- Manifest
### provides: t/one
### distro: [all]
### requires_env: [ONE]
### -- End
cp "$ONE" `+out+`/one
touch `+out+`/ran
`, 0o755)
	pre := filepath.Join(dir, "pre.sh")
	write(t, pre, "touch "+out+"/pre\n", 0o755)
	distro = filepath.Join(dir, "distro.yml")
	write(t, distro, "os: ubuntu\nsources: ["+src+"]\npackages: [t/one]\n"+
		"preconditions: [{script: "+pre+"}]\nsecrets: {bws: {ONE: "+key+"}}\n", 0o644)
	return distro, out, fakeLog
}

func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func bwsCalls(t *testing.T, fakeLog string) int {
	b, _ := os.ReadFile(fakeLog)
	return strings.Count(string(b), "argv: ")
}

// A package already installed is skipped by dis install and needs no secrets,
// so no bws call; dis config runs it, so it does need them.
func TestSecretsOnlyForPackagesThatRun(t *testing.T) {
	distro, out, fakeLog := secretsFixture(t, "test/one", "good")

	if _, err := runCmd(t, "install", "--distro", distro); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(out, "one")); string(got) != "v1" {
		t.Errorf("installer read %q from the secret file, want v1", got)
	}
	if n := bwsCalls(t, fakeLog); n != 1 {
		t.Fatalf("first install: %d bws calls, want 1", n)
	}
	// The secrets dir is gone after the run.
	if entries, _ := os.ReadDir(os.Getenv("XDG_RUNTIME_DIR")); len(entries) != 0 {
		t.Errorf("$XDG_RUNTIME_DIR holds %v after the run", entries)
	}

	if _, err := runCmd(t, "install", "--distro", distro); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if n := bwsCalls(t, fakeLog); n != 1 {
		t.Errorf("second install of an installed package called bws (%d calls in all)", n)
	}

	if _, err := runCmd(t, "config", "--distro", distro); err != nil {
		t.Fatalf("config: %v", err)
	}
	if n := bwsCalls(t, fakeLog); n != 2 {
		t.Errorf("dis config: %d bws calls in all, want 2", n)
	}
}

// A secret that can't be fetched stops the run before the preconditions and
// the installers.
func TestSecretsFailBeforePreconditions(t *testing.T) {
	tests := []struct{ name, key, token, want string }{
		{"missing key", "test/missing", "good", "BWS key test/missing not found (needed by t/one)"},
		{"rejected token", "test/one", "bad", "the token may be revoked or expired"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distro, out, _ := secretsFixture(t, tt.key, tt.token)
			for _, command := range []string{"install", "config"} {
				_, err := runCmd(t, command, "--distro", distro)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Errorf("dis %s = %v, want an error containing %q", command, err, tt.want)
				}
				for _, f := range []string{"pre", "ran"} {
					if _, err := os.Stat(filepath.Join(out, f)); err == nil {
						t.Errorf("dis %s: $OUT/%s exists: it ran before the secrets failed", command, f)
					}
				}
			}
		})
	}
}
