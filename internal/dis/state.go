package dis

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// stateFilePath returns the path to the installed-packages state file.
// It follows the XDG Base Directory spec: ~/.local/share/dis/installed.txt
func stateFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "dis", "installed.txt"), nil
}

// RecordInstalled appends pkgName to the state file if it is not already
// present. The state directory is created automatically on first use.
func RecordInstalled(pkgName string) error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	already, err := IsInstalled(pkgName)
	if err != nil {
		return err
	}
	if already {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open state file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, pkgName)
	return err
}

// RemoveInstalled removes pkgName from the state file.
// It is a no-op if the package is not recorded.
func RemoveInstalled(pkgName string) error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}

	lines, err := readStateLines(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	filtered := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != pkgName {
			filtered = append(filtered, l)
		}
	}

	return writeStateLines(path, filtered)
}

// IsInstalled reports whether pkgName is recorded in the state file.
func IsInstalled(pkgName string) (bool, error) {
	path, err := stateFilePath()
	if err != nil {
		return false, err
	}

	lines, err := readStateLines(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	for _, l := range lines {
		if l == pkgName {
			return true, nil
		}
	}
	return false, nil
}

// ListInstalled returns all package names recorded in the state file.
func ListInstalled() ([]string, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}

	lines, err := readStateLines(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return lines, nil
}

func readStateLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines, scanner.Err()
}

func writeStateLines(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	defer f.Close()

	for _, l := range lines {
		if _, err := fmt.Fprintln(f, l); err != nil {
			return err
		}
	}
	return nil
}
