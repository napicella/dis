package dis

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// installedFile lists the installed packages, one per line.
	installedFile = "installed.txt"
	// exportsCacheFile stores all pkg:KEY=value pairs ever exported by
	// installers, keyed by qualified name.
	exportsCacheFile = "exports-cache.txt"
)

// State is the install state dis keeps on this machine: the packages recorded
// as installed and the exports cache. Both are files in one directory, created
// on first write.
type State struct {
	dir string
}

// NewState returns the State kept in dir.
func NewState(dir string) *State {
	return &State{dir: dir}
}

// DefaultState returns the State kept in ~/.local/share/dis, following the XDG
// Base Directory spec.
func DefaultState() (*State, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	return NewState(filepath.Join(home, ".local", "share", "dis")), nil
}

// ReadExports reads all cached exports and returns them as a map of
// qualified key ("pkg:VAR") → value.
func (s *State) ReadExports() (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, exportsCacheFile))
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

// UpdateExports merges newEntries into the exports cache, overwriting existing
// values for the same keys and writing the result back.
func (s *State) UpdateExports(newEntries map[string]string) error {
	if len(newEntries) == 0 {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Read current cache, merge new entries, rewrite.
	existing, err := s.ReadExports()
	if err != nil {
		return err
	}
	for k, v := range newEntries {
		existing[k] = v
	}

	f, err := os.Create(filepath.Join(s.dir, exportsCacheFile))
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

// RecordInstalled records pkgName as installed, if it is not already.
func (s *State) RecordInstalled(pkgName string) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	already, err := s.IsInstalled(pkgName)
	if err != nil {
		return err
	}
	if already {
		return nil
	}

	f, err := os.OpenFile(filepath.Join(s.dir, installedFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open state file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, pkgName)
	return err
}

// RemoveInstalled removes pkgName from the packages recorded as installed.
// It is a no-op if the package is not recorded.
func (s *State) RemoveInstalled(pkgName string) error {
	path := filepath.Join(s.dir, installedFile)
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

// IsInstalled reports whether pkgName is recorded as installed.
func (s *State) IsInstalled(pkgName string) (bool, error) {
	lines, err := readStateLines(filepath.Join(s.dir, installedFile))
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

// ListInstalled returns all package names recorded as installed, sorted.
func (s *State) ListInstalled() ([]string, error) {
	lines, err := readStateLines(filepath.Join(s.dir, installedFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Strings(lines)
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
