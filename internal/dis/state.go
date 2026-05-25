package dis

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// exportsCacheFilePath returns the path to the exports cache file.
// ~/.local/share/dis/exports-cache.txt stores all pkg:KEY=value pairs
// ever exported by installers, keyed by qualified name.
func exportsCacheFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "dis", "exports-cache.txt"), nil
}

// ReadExportsCache reads all cached exports and returns them as a map of
// qualified key ("pkg:VAR") → value.
func ReadExportsCache() (map[string]string, error) {
	path, err := exportsCacheFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading exports cache: %w", err)
	}

	result := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 1 {
			continue
		}
		result[strings.TrimSpace(line[:idx])] = line[idx+1:]
	}
	return result, nil
}

// UpdateExportsCache merges newEntries into the persistent exports cache,
// overwriting existing values for the same keys and writing the result back.
func UpdateExportsCache(newEntries map[string]string) error {
	if len(newEntries) == 0 {
		return nil
	}
	path, err := exportsCacheFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Read current cache, merge new entries, rewrite.
	existing, err := ReadExportsCache()
	if err != nil {
		return err
	}
	for k, v := range newEntries {
		existing[k] = v
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("write exports cache: %w", err)
	}
	defer f.Close()
	for k, v := range existing {
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, v); err != nil {
			return err
		}
	}
	return nil
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
	sort.Strings(lines)
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
