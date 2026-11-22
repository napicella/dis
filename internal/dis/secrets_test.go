package dis

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// fakeBws is a stand-in for bws: it logs its argv, and whether it got a token,
// to $FAKE_LOG, and lists secrets only for the token "good".
const fakeBws = `#!/bin/sh
printf 'argv: %s\n' "$*" >> "$FAKE_LOG"; [ -n "$BWS_ACCESS_TOKEN" ] && echo "token-in-env" >> "$FAKE_LOG"
[ "$BWS_ACCESS_TOKEN" = good ] || { echo "Error: 401 Unauthorized" >&2; exit 1; }
printf '[{"key":"test/one","value":"v1"},{"key":"test/two","value":"a\\nb"},{"key":"dup","value":"x"},{"key":"dup","value":"y"}]\n'
`

// secretsEnv isolates a test from the real home, bws and terminal: HOME and
// XDG_RUNTIME_DIR are temp dirs, DIS_BWS is the fake, and there is no tty.
// It returns the home and the fake's log path.
func secretsEnv(t *testing.T) (home, fakeLog string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	bin := filepath.Join(t.TempDir(), "fake-bws")
	if err := os.WriteFile(bin, []byte(fakeBws), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DIS_BWS", bin)
	fakeLog = filepath.Join(t.TempDir(), "fake.log")
	t.Setenv("FAKE_LOG", fakeLog)
	old := ttyPath
	ttyPath = filepath.Join(t.TempDir(), "no-tty")
	t.Cleanup(func() { ttyPath = old })
	return home, fakeLog
}

// writeToken places the token file with the given content and mode.
func writeToken(t *testing.T, home, token string, mode os.FileMode) {
	t.Helper()
	path := tokenPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(token+"\n"), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

// secretsContext is an install context whose distro maps the given secrets
// to BWS keys.
func secretsContext(bws map[string]string) *InstallContext {
	secrets := Secrets{}
	for name, key := range bws {
		secrets[name] = SecretRef{Manager: secretManagerBWS, Key: key}
	}
	return &InstallContext{Cfg: DistroConfig{Secrets: secrets}, parameters: map[string]string{}}
}

var (
	needsOne = Manifest{Provides: "t/one", RequiresEnv: []string{"ONE"}}
	needsDup = Manifest{Provides: "t/dup", RequiresEnv: []string{"DUP"}}
	noSecret = Manifest{Provides: "t/plain", RequiresEnv: []string{}}
)

func TestPrepareSecretsWritesFilesAndInjectsPaths(t *testing.T) {
	home, fakeLog := secretsEnv(t)
	writeToken(t, home, "good", 0o600)
	ic := secretsContext(map[string]string{"ONE": "test/one", "TWO": "test/two"})

	sf, err := ic.PrepareSecrets([]Manifest{needsOne, noSecret})
	if err != nil {
		t.Fatalf("PrepareSecrets: %v", err)
	}
	env, err := ic.envForInstaller(needsOne)
	if err != nil {
		t.Fatalf("envForInstaller: %v", err)
	}
	path := env["ONE"]
	if filepath.Dir(filepath.Dir(path)) != os.Getenv("XDG_RUNTIME_DIR") {
		t.Errorf("secret file %s is not in a dir under $XDG_RUNTIME_DIR", path)
	}
	if got, _ := os.ReadFile(path); string(got) != "v1" {
		t.Errorf("secret file holds %q, want %q", got, "v1")
	}
	assertMode(t, path, 0o600)
	assertMode(t, filepath.Dir(path), 0o700)
	// Only the names that run needs are written.
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "TWO")); err == nil {
		t.Errorf("TWO was written, but nothing that runs needs it")
	}

	log, _ := os.ReadFile(fakeLog)
	if want := "argv: secret list --output json\ntoken-in-env\n"; string(log) != want {
		t.Errorf("fake bws log = %q, want %q", log, want)
	}
	assertMode(t, filepath.Join(home, ".config", "bws"), 0o700)

	if err := sf.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("secrets dir still there after Close: %v", err)
	}
	if err := sf.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestPrepareSecretsValueWrittenExactly(t *testing.T) {
	home, _ := secretsEnv(t)
	writeToken(t, home, "good", 0o600)
	ic := secretsContext(map[string]string{"TWO": "test/two"})
	sf, err := ic.PrepareSecrets([]Manifest{{Provides: "t/two", RequiresEnv: []string{"TWO"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer sf.Close()
	if got, _ := os.ReadFile(ic.secretFiles["TWO"]); string(got) != "a\nb" {
		t.Errorf("TWO holds %q, want %q", got, "a\nb")
	}
}

// Nothing that runs lists a secret: no bws, no token, no network.
func TestPrepareSecretsNoneNeeded(t *testing.T) {
	for name, ic := range map[string]*InstallContext{
		"distro without secrets":       secretsContext(nil),
		"secret no installer requires": secretsContext(map[string]string{"ONE": "test/one"}),
	} {
		t.Run(name, func(t *testing.T) {
			home, fakeLog := secretsEnv(t)
			t.Setenv("DIS_BWS", "") // a download would fail the test: no pin served
			old := bwsReleaseURL
			bwsReleaseURL = "http://127.0.0.1:1/"
			t.Cleanup(func() { bwsReleaseURL = old })

			sf, err := ic.PrepareSecrets([]Manifest{noSecret})
			if err != nil || sf != nil {
				t.Fatalf("PrepareSecrets = %v, %v; want nil, nil", sf, err)
			}
			if _, err := os.Stat(fakeLog); !os.IsNotExist(err) {
				t.Errorf("bws ran")
			}
			for _, p := range []string{".local", ".config"} {
				if _, err := os.Stat(filepath.Join(home, p)); !os.IsNotExist(err) {
					t.Errorf("~/%s was created", p)
				}
			}
			if err := sf.Close(); err != nil {
				t.Errorf("Close on nil: %v", err)
			}
		})
	}
}

func TestPrepareSecretsErrors(t *testing.T) {
	tests := []struct {
		name    string
		token   string // "" for no token file
		mode    os.FileMode
		secrets map[string]string
		run     []Manifest
		want    []string
	}{
		{"missing key", "good", 0o600, map[string]string{"ONE": "test/missing"}, []Manifest{needsOne},
			[]string{"secret ONE: BWS key test/missing not found (needed by t/one)"}},
		{"duplicate key", "good", 0o600, map[string]string{"DUP": "dup"}, []Manifest{needsDup},
			[]string{"secret DUP: BWS key dup found 2 times"}},
		{"rejected token", "bad", 0o600, map[string]string{"ONE": "test/one"}, []Manifest{needsOne},
			[]string{"Error: 401 Unauthorized", "the token may be revoked or expired: replace ~/.config/dis/bws-token"}},
		{"token readable by others", "good", 0o644, map[string]string{"ONE": "test/one"}, []Manifest{needsOne},
			[]string{"~/.config/dis/bws-token must be mode 600 and owned by "}},
		{"no token and no terminal", "", 0, map[string]string{"ONE": "test/one"}, []Manifest{needsOne},
			[]string{"no BWS token at ~/.config/dis/bws-token and no terminal to ask for one: create the file (one line, mode 600)"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home, _ := secretsEnv(t)
			if tt.token != "" {
				writeToken(t, home, tt.token, tt.mode)
			}
			ic := secretsContext(tt.secrets)
			sf, err := ic.PrepareSecrets(tt.run)
			if err == nil {
				sf.Close()
				t.Fatal("PrepareSecrets succeeded, want an error")
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
			if strings.Contains(err.Error(), "v1") || strings.Contains(err.Error(), "good") {
				t.Errorf("error leaks a secret or the token: %q", err)
			}
			// No secrets dir is left behind.
			if entries, _ := os.ReadDir(os.Getenv("XDG_RUNTIME_DIR")); len(entries) != 0 {
				t.Errorf("$XDG_RUNTIME_DIR holds %d entries, want none", len(entries))
			}
		})
	}
}

func TestSecretsDirNeverOnDisk(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := secretsDir([]string{"", missing}); err == nil ||
		!strings.Contains(err.Error(), "neither $XDG_RUNTIME_DIR nor /dev/shm exists") {
		t.Errorf("secretsDir with no base = %v, want the no tmpfs error", err)
	}
	second := t.TempDir()
	dir, err := secretsDir([]string{missing, second})
	if err != nil || filepath.Dir(dir) != second {
		t.Errorf("secretsDir = %q, %v; want a dir in %s", dir, err, second)
	}
}

// A requires_env name in the distro's secrets that PrepareSecrets didn't fetch
// fails, instead of falling back to a parameter of the same name.
func TestEnvForInstallerSecretNotFetched(t *testing.T) {
	ic := secretsContext(map[string]string{"ONE": "test/one"})
	ic.parameters["ONE"] = "a parameter"
	if _, err := ic.envForInstaller(needsOne); err == nil || !strings.Contains(err.Error(), `secret "ONE" but it was not fetched`) {
		t.Errorf("envForInstaller = %v, want the not fetched error", err)
	}
}

func TestInstallBws(t *testing.T) {
	zipOf := func(entries map[string]string) []byte {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for name, content := range entries {
			w, _ := zw.Create(name)
			w.Write([]byte(content))
		}
		zw.Close()
		return buf.Bytes()
	}
	sha := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

	tests := []struct {
		name    string
		zip     []byte
		sha     string // "" for the zip's own
		wantErr string
	}{
		{"installs", zipOf(map[string]string{"bws": "#!/bin/sh\n"}), "", ""},
		{"checksum mismatch", zipOf(map[string]string{"bws": "tampered"}), strings.Repeat("0", 64), "bws download: checksum mismatch"},
		{"other entry", zipOf(map[string]string{"bws": "x", "../bws": "y"}), "", "unexpected entry"},
		{"no bws", zipOf(map[string]string{"bin/bws": "x"}), "", "unexpected entry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write(tt.zip) }))
			defer srv.Close()
			dir := filepath.Join(t.TempDir(), "bin")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			old := filepath.Join(dir, "bws-1.0.0")
			os.WriteFile(old, nil, 0o755)
			want := tt.sha
			if want == "" {
				want = sha(tt.zip)
			}
			path := filepath.Join(dir, "bws-"+bwsVersion)

			err := installBws(dir, path, srv.URL+"/asset.zip", want)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("installBws = %v, want %q", err, tt.wantErr)
				}
				// Nothing left behind but what was there before.
				if entries, _ := os.ReadDir(dir); len(entries) != 1 || entries[0].Name() != "bws-1.0.0" {
					t.Errorf("dir holds %v, want only bws-1.0.0", entries)
				}
				return
			}
			if err != nil {
				t.Fatalf("installBws: %v", err)
			}
			if got, _ := os.ReadFile(path); string(got) != "#!/bin/sh\n" {
				t.Errorf("bws content = %q", got)
			}
			assertMode(t, path, 0o755)
			if entries, _ := os.ReadDir(dir); len(entries) != 1 {
				t.Errorf("dir holds %v, want only %s (older pins removed)", entries, filepath.Base(path))
			}
		})
	}
}

// The pinned bws in place is used without a download.
func TestBwsExecutableUsesInstalledPin(t *testing.T) {
	if _, ok := bwsPins[runtime.GOOS+"/"+runtime.GOARCH]; !ok || checkSystemCAs() != nil {
		t.Skip("no pin for this platform, or no system CAs")
	}
	home := t.TempDir()
	t.Setenv("DIS_BWS", "")
	old := bwsReleaseURL
	bwsReleaseURL = "http://127.0.0.1:1/"
	t.Cleanup(func() { bwsReleaseURL = old })
	path := filepath.Join(home, ".local", "share", "dis", "bin", "bws-"+bwsVersion)
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, nil, 0o755)
	if got, err := bwsExecutable(home); err != nil || got != path {
		t.Errorf("bwsExecutable = %q, %v; want %q", got, err, path)
	}
}

func TestEnsureBwsStateDir(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "bws")
	os.MkdirAll(dir, 0o755)
	if err := ensureBwsStateDir(home); err != nil {
		t.Fatal(err)
	}
	assertMode(t, dir, 0o700)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s: mode %o, want %o", path, got, want)
	}
}

// loadSecrets loads a distro file with the given secrets: block.
func loadSecrets(t *testing.T, block string) (Secrets, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "distro.yml")
	if err := os.WriteFile(path, []byte("os: all\n"+block), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadDistro(path)
	return cfg.Secrets, err
}

func TestSecretsYAMLLoads(t *testing.T) {
	got, err := loadSecrets(t, "secrets:\n  bws:\n    ONE: test/one\n    TWO: test/two\n")
	if err != nil {
		t.Fatalf("loadDistro: %v", err)
	}
	want := Secrets{"ONE": {Manager: "bws", Key: "test/one"}, "TWO": {Manager: "bws", Key: "test/two"}}
	if len(got) != len(want) || got["ONE"] != want["ONE"] || got["TWO"] != want["TWO"] {
		t.Errorf("Secrets = %v, want %v", got, want)
	}
}

func TestSecretsYAMLEmpty(t *testing.T) {
	for name, block := range map[string]string{
		"no secrets":    "",
		"secrets empty": "secrets:\n",
		"secrets {}":    "secrets: {}\n",
		"bws {}":        "secrets:\n  bws: {}\n",
		"bws empty":     "secrets:\n  bws:\n",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := loadSecrets(t, block)
			if err != nil || len(got) != 0 {
				t.Errorf("Secrets = %v, %v; want none, nil", got, err)
			}
		})
	}
}

func TestSecretsYAMLErrors(t *testing.T) {
	tests := []struct {
		name  string
		block string
		want  []string // substrings of the error
	}{
		{
			"NAME: key directly under secrets",
			"secrets:\n  AUTHELIA_SESSION_SECRET: authelia/session_secret\n",
			[]string{"line 3", `unknown secret manager "AUTHELIA_SESSION_SECRET" (known: bws)`, "write secrets: {bws: {NAME: key}}"},
		},
		{
			"unknown manager",
			"secrets:\n  vault:\n    ONE: test/one\n",
			[]string{"line 3", `unknown secret manager "vault" (known: bws)`},
		},
		{
			"manager twice",
			"secrets:\n  bws:\n    ONE: test/one\n  bws:\n    TWO: test/two\n",
			[]string{"line 5", `secrets: "bws" already defined at line 3`},
		},
		{"manager not a map", "secrets:\n  bws: [ONE]\n", []string{"line 3"}},
		{"secrets not a map", "secrets: [ONE]\n", []string{"line 2", "secrets: must be a map"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadSecrets(t, tt.block)
			if err == nil {
				t.Fatal("loadDistro succeeded, want an error")
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
		})
	}
}

// A name under two managers is an error. bws is the only manager today, so
// this calls the step that adds a manager's group.
func TestSecretsSameNameUnderTwoManagers(t *testing.T) {
	s := Secrets{}
	if err := s.add("bws", map[string]string{"ONE": "test/one"}); err != nil {
		t.Fatal(err)
	}
	err := s.add("other", map[string]string{"ONE": "x"})
	if err == nil || !strings.Contains(err.Error(), "secret ONE is under both bws and other") {
		t.Errorf("add = %v, want the both managers error", err)
	}
}

// A manager dis knows how to read but not to fetch from fails, rather than
// writing empty secret files.
func TestPrepareSecretsManagerWithoutFetch(t *testing.T) {
	secretsEnv(t)
	old := secretManagers
	secretManagers = append(slices.Clone(old), "other")
	t.Cleanup(func() { secretManagers = old })
	ic := &InstallContext{Cfg: DistroConfig{Secrets: Secrets{"ONE": {Manager: "other", Key: "k"}}}, parameters: map[string]string{}}
	sf, err := ic.PrepareSecrets([]Manifest{needsOne})
	if err == nil || !strings.Contains(err.Error(), `no fetch for secret manager "other"`) {
		sf.Close()
		t.Errorf("PrepareSecrets = %v, want the no fetch error", err)
	}
}
