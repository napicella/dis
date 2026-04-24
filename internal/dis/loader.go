package dis

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// commonSourceToken is the placeholder that expands to the first conventional
// dis packages directory found on disk.
const commonSourceToken = "${common_sources}"

// commonSourceDir returns the first existing conventional dis packages directory,
// probing in XDG-style priority order. Returns "" if none exist.
func commonSourceDir() string {
	home, _ := os.UserHomeDir()
	for _, c := range []string{
		filepath.Join(home, ".local/share/dis/packages"),
		"/usr/local/share/dis/packages",
		"/usr/share/dis/packages",
	} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// loadDistro reads and parses a distro YAML file.
func loadDistro(path string) (DistroConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DistroConfig{}, fmt.Errorf("reading distro file %q: %w", path, err)
	}
	var cfg DistroConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return DistroConfig{}, fmt.Errorf("parsing distro file %q: %w", path, err)
	}
	return cfg, nil
}

// loadInstallers walks all source directories, parses every installer manifest
// whose distro field matches targetOS (or "all"), and returns the collected
// manifests.
func loadInstallers(sources []string, targetOS string) ([]Manifest, error) {
	var manifests []Manifest
	for _, srcDir := range sources {
		ms, err := walkSource(srcDir, targetOS)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, ms...)
	}
	return manifests, nil
}

// walkSource recursively walks srcDir, parses manifest headers from .sh files,
// and returns those applicable for targetOS.
func walkSource(srcDir, targetOS string) ([]Manifest, error) {
	var manifests []Manifest
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".sh") {
			return nil
		}
		m, ok, parseErr := parseManifest(path, srcDir)
		if parseErr != nil {
			return fmt.Errorf("parsing manifest in %q: %w", path, parseErr)
		}
		if !ok {
			return nil
		}
		if !distroMatches(m.Distros, targetOS) {
			return nil
		}
		manifests = append(manifests, m)
		return nil
	})
	return manifests, err
}
