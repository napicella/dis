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

// workspaceFile is the name of the optional workspace marker file.
const workspaceFile = "dis.ws.yml"

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

// loadWorkspace attempts to read and parse a dis.workspace file from srcDir.
// All relative paths in the workspace are resolved to absolute paths using
// srcDir as the base. Returns (cfg, true, nil) if found and parsed
// successfully. Returns (_, false, nil) if no workspace file exists.
func loadWorkspace(srcDir string) (WorkspaceConfig, bool, error) {
	wsPath := filepath.Join(srcDir, workspaceFile)
	data, err := os.ReadFile(wsPath)
	if os.IsNotExist(err) {
		return WorkspaceConfig{}, false, nil
	}
	if err != nil {
		return WorkspaceConfig{}, false, fmt.Errorf("reading workspace file %q: %w", wsPath, err)
	}
	var cfg WorkspaceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return WorkspaceConfig{}, false, fmt.Errorf("parsing workspace file %q: %w", wsPath, err)
	}
	// Resolve all paths to absolute.
	for i, pkg := range cfg.Packages {
		if !filepath.IsAbs(pkg.Root) {
			cfg.Packages[i].Root = filepath.Clean(filepath.Join(srcDir, pkg.Root))
		}
		if pkg.Configs != "" && !filepath.IsAbs(pkg.Configs) {
			cfg.Packages[i].Configs = filepath.Clean(filepath.Join(srcDir, pkg.Configs))
		}
	}
	return cfg, true, nil
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

// walkSource loads installers from srcDir. If a dis.workspace file is present,
// only the declared package roots are walked and each manifest receives the
// PkgRoot and ConfigsDir from its workspace entry. Otherwise the entire srcDir
// is walked and PkgRoot defaults to srcDir with an empty ConfigsDir.
func walkSource(srcDir, targetOS string) ([]Manifest, error) {
	ws, hasWS, err := loadWorkspace(srcDir)
	if err != nil {
		return nil, err
	}

	if !hasWS {
		// Legacy / simple mode: walk entire source dir, PkgRoot = srcDir.
		return walkDir(srcDir, srcDir, srcDir, "", targetOS)
	}

	// Workspace mode: walk each declared package root.
	var manifests []Manifest
	for _, pkg := range ws.Packages {
		ms, err := walkDir(pkg.Root, srcDir, pkg.Root, pkg.Configs, targetOS)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, ms...)
	}
	return manifests, nil
}

// walkDir recursively walks walkRoot for .sh installer files, using sourceDir
// as the SourceDir on each manifest and pkgRoot/configsDir as PkgRoot/ConfigsDir.
func walkDir(walkRoot, sourceDir, pkgRoot, configsDir, targetOS string) ([]Manifest, error) {
	var manifests []Manifest
	err := filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".sh") {
			return nil
		}
		m, ok, parseErr := parseManifest(path, sourceDir, pkgRoot, configsDir)
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
