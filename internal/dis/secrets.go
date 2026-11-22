package dis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
)

// SecretFiles is the private tmpfs directory holding one file per secret a
// run needs. Close removes it; a nil *SecretFiles is a run without secrets.
type SecretFiles struct {
	dir  string
	once sync.Once
	stop func()
}

// Close removes the secrets directory and stops the signal handler. It is
// safe to call more than once, and on nil.
func (s *SecretFiles) Close() error {
	if s == nil {
		return nil
	}
	var err error
	s.once.Do(func() {
		s.stop()
		err = os.RemoveAll(s.dir)
	})
	return err
}

// watchSignals removes the directory on SIGINT or SIGTERM, then exits as a
// shell does: deferred cleanup doesn't run when a signal kills the process.
func (s *SecretFiles) watchSignals() {
	s.stop = onSignals(func() { s.Close() }, syscall.SIGINT, syscall.SIGTERM)
}

// onSignals runs cleanup on the first of sigs, then exits with 128 plus the
// signal's number. The returned stop ends the watch; call it once.
func onSignals(cleanup func(), sigs ...os.Signal) (stop func()) {
	ch := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(ch, sigs...)
	go func() {
		select {
		case sig := <-ch:
			cleanup()
			os.Exit(128 + int(sig.(syscall.Signal)))
		case <-done:
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}

// bwsSecret is one entry of `bws secret list --output json`.
type bwsSecret struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PrepareSecrets fetches the secrets the manifests in toRun list in
// requires_env and writes each to a file that envForInstaller injects the
// path of. toRun is what will actually execute. When none of them needs a
// secret it does nothing: no bws, no token, no network, and returns nil.
// Otherwise it fetches them from their managers (bwsValues). The caller must
// Close the result.
func (ic *InstallContext) PrepareSecrets(toRun []Manifest) (*SecretFiles, error) {
	// needed maps each secret name to the packages that need it.
	needed := map[string][]string{}
	for _, m := range toRun {
		for _, name := range m.RequiresEnv {
			if _, ok := ic.Cfg.Secrets[name]; ok && !slices.Contains(needed[name], m.Provides) {
				needed[name] = append(needed[name], m.Provides)
			}
		}
	}
	if len(needed) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(needed))
	for name := range needed {
		names = append(names, name)
	}
	slices.Sort(names)

	values := make(map[string]string, len(names))
	for _, manager := range secretManagers {
		var group []string
		for _, name := range names {
			if ic.Cfg.Secrets[name].Manager == manager {
				group = append(group, name)
			}
		}
		if len(group) == 0 {
			continue
		}
		var err error
		switch manager {
		case secretManagerBWS:
			err = ic.bwsValues(group, needed, values)
		default:
			err = fmt.Errorf("no fetch for secret manager %q", manager)
		}
		if err != nil {
			return nil, err
		}
	}

	dir, err := secretsDir([]string{os.Getenv("XDG_RUNTIME_DIR"), "/dev/shm"})
	if err != nil {
		return nil, err
	}
	sf := &SecretFiles{dir: dir}
	sf.watchSignals()
	ic.secretFiles = make(map[string]string, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(values[name]), 0o600); err != nil {
			sf.Close()
			return nil, err
		}
		ic.secretFiles[name] = path
	}
	return sf, nil
}

// bwsValues fetches the named secrets from Bitwarden Secrets Manager into
// values. needed maps each name to the packages that need it. It installs bws
// if needed, reads the token (asking for it when ~/.config/dis/bws-token is
// missing), and runs one `bws secret list`.
func (ic *InstallContext) bwsValues(names []string, needed map[string][]string, values map[string]string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}
	bws, err := bwsExecutable(home)
	if err != nil {
		return err
	}
	token, err := loadToken(home)
	if err != nil {
		return err
	}
	if err := ensureBwsStateDir(home); err != nil {
		return err
	}
	list, err := fetchSecrets(bws, token)
	if err != nil {
		return err
	}

	for _, name := range names {
		key := ic.Cfg.Secrets[name].Key
		found := 0
		for _, s := range list {
			if s.Key == key {
				values[name] = s.Value
				found++
			}
		}
		switch {
		case found == 0:
			return fmt.Errorf("secret %s: BWS key %s not found (needed by %s)", name, key, strings.Join(needed[name], ", "))
		case found > 1:
			return fmt.Errorf("secret %s: BWS key %s found %d times: keys must be unique", name, key, found)
		}
	}
	return nil
}

// fetchSecrets runs `bws secret list` with the token in BWS_ACCESS_TOKEN, never
// on the command line, and parses its output.
func fetchSecrets(bws, token string) ([]bwsSecret, error) {
	cmd := exec.Command(bws, "secret", "list", "--output", "json")
	cmd.Env = append(os.Environ(), "BWS_ACCESS_TOKEN="+token)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bws secret list: %w: %s\nthe token may be revoked or expired: replace ~/.config/dis/bws-token",
			err, strings.TrimSpace(stderr.String()))
	}
	var list []bwsSecret
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		return nil, fmt.Errorf("bws secret list: parsing its output: %w", err)
	}
	return list, nil
}

// secretsDir creates a private directory in the first of bases that exists.
// They are tmpfs mounts ($XDG_RUNTIME_DIR, /dev/shm): secrets never go to disk,
// so there is no fallback.
func secretsDir(bases []string) (string, error) {
	for _, base := range bases {
		if base == "" {
			continue
		}
		if info, err := os.Stat(base); err == nil && info.IsDir() {
			return os.MkdirTemp(base, "dis-secrets-") // mode 0700
		}
	}
	return "", errors.New("no tmpfs for secrets: neither $XDG_RUNTIME_DIR nor /dev/shm exists")
}
