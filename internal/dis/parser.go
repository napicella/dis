package dis

import (
	"bufio"
	"os"
	"strings"
)

// parseManifest reads the manifest header block from a .sh installer file.
// Returns (manifest, true, nil) if a valid manifest was found.
// Returns (_, false, nil) if no manifest block was found.
// Returns (_, false, err) on a parse error.
//
// Expected format (lines between "### -- Manifest" and "### -- End"):
//
//	### provides: common/tools
//	### depends_on: [common/mise,common/os-libs]
//	### distro: [ubuntu]
func parseManifest(path, sourceDir, pkgRoot, configsDir string) (Manifest, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, false, err
	}
	defer f.Close()

	var (
		inBlock     bool
		provides    string
		dependsOn   []string
		distros     []string
		requiresEnv []string
		exportsEnv  []string
	)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "### -- Manifest" {
			inBlock = true
			continue
		}
		if line == "### -- End" {
			break
		}
		if !inBlock {
			continue
		}

		if !strings.HasPrefix(line, "### ") {
			continue
		}
		content := strings.TrimPrefix(line, "### ")

		if val, ok := parseField(content, "provides"); ok {
			provides = strings.TrimSpace(val)
		} else if val, ok := parseField(content, "depends_on"); ok {
			dependsOn = parseList(val)
		} else if val, ok := parseField(content, "distro"); ok {
			distros = parseList(val)
		} else if val, ok := parseField(content, "requires_env"); ok {
			requiresEnv = parseList(val)
		} else if val, ok := parseField(content, "exports_env"); ok {
			exportsEnv = parseList(val)
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, false, err
	}

	if !inBlock || provides == "" {
		return Manifest{}, false, nil
	}

	return Manifest{
		Provides:      provides,
		InstallerPath: path,
		Distros:       distros,
		DependsOn:     dependsOn,
		RequiresEnv:   requiresEnv,
		ExportsEnv:    exportsEnv,
		SourceDir:     sourceDir,
		PkgRoot:       pkgRoot,
		ConfigsDir:    configsDir,
	}, true, nil
}

// parseField checks if line starts with "key: " and returns the value.
func parseField(line, key string) (string, bool) {
	prefix := key + ":"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
}

// parseList parses "[a,b,c]" or "[]" into a slice of trimmed strings.
func parseList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// distroMatches returns true if distros contains targetOS or "all".
func distroMatches(distros []string, targetOS string) bool {
	for _, d := range distros {
		if d == "all" || d == targetOS {
			return true
		}
	}
	return false
}
