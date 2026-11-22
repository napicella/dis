package dis

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// ttyPath is the terminal dis asks for the token on; tests change it.
var ttyPath = "/dev/tty"

// tokenPath is the file holding the BWS access token.
func tokenPath(home string) string {
	return filepath.Join(home, ".config", "dis", "bws-token")
}

// loadToken returns the BWS access token from ~/.config/dis/bws-token. The
// file must be private (no group or other bits) and owned by the user running
// dis. When it is missing, dis asks for the token on the terminal and saves
// it there.
func loadToken(home string) (string, error) {
	path := tokenPath(home)
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return promptToken(path)
	}
	if err != nil {
		return "", err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if info.Mode().Perm()&0o077 != 0 || !ok || int(st.Uid) != os.Getuid() {
		return "", fmt.Errorf("~/.config/dis/bws-token must be mode 600 and owned by %s", currentUser())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// promptToken asks for the token on the terminal, without echo, and writes it
// to path atomically with mode 0600, in a directory made 0700.
func promptToken(path string) (string, error) {
	tty, err := os.OpenFile(ttyPath, os.O_RDWR, 0)
	if err != nil {
		return "", errors.New("no BWS token at ~/.config/dis/bws-token and no terminal to ask for one: create the file (one line, mode 600)")
	}
	defer tty.Close()
	fd := int(tty.Fd())
	// ReadPassword turns echo off; a signal at the prompt would kill dis with
	// it still off, so restore the terminal first.
	if state, err := term.GetState(fd); err == nil {
		stop := onSignals(func() {
			term.Restore(fd, state)
			fmt.Fprintln(tty)
		}, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		defer stop()
	}
	fmt.Fprint(tty, "Bitwarden Secrets Manager access token: ")
	answer, err := term.ReadPassword(fd)
	fmt.Fprintln(tty)
	if err != nil {
		return "", fmt.Errorf("reading the BWS token: %w", err)
	}
	token := strings.TrimSpace(string(answer))
	if token == "" {
		return "", errors.New("no BWS token given")
	}

	// The directory is private even when dis pull created it 0755 for its
	// config.yaml.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".bws-token-*") // mode 0600
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.WriteString(token + "\n"); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}
	return token, nil
}

// currentUser is the name of the user running dis, or its uid.
func currentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return strconv.Itoa(os.Getuid())
}
