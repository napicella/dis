package dis

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ContentMatch is a line of an installer body that matched a content search.
type ContentMatch struct {
	// Line is the 1-based line number within the installer file.
	Line int
	// Text is the matching line with surrounding whitespace trimmed.
	Text string
}

// configRefRe matches a reference to the configs folder, e.g.
// "$DIS_CONFIG_FOLDER/starship/starship.toml" or "${DIS_CONFIG_FOLDER}/.tmux.conf".
// The captured path stops at whitespace, quotes and shell syntax, and at the
// first "$": the static prefix of "$DIS_CONFIG_FOLDER/backgrounds/$THEME" is
// "/backgrounds/".
var configRefRe = regexp.MustCompile(`\$\{?DIS_CONFIG_FOLDER\}?(/[^\s"'$;)|&<>` + "`" + `]*)?`)

// scanBody calls fn for every line of the installer at path that is not part
// of the manifest header block.
func scanBody(path string, fn func(lineNo int, line string)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	inManifest := false
	scanner := bufio.NewScanner(f)
	for n := 1; scanner.Scan(); n++ {
		line := scanner.Text()
		switch strings.TrimSpace(line) {
		case "### -- Manifest":
			inManifest = true
			continue
		case "### -- End":
			if inManifest {
				inManifest = false
				continue
			}
		}
		if !inManifest {
			fn(n, line)
		}
	}
	return scanner.Err()
}

// SearchContent returns the lines of the installer at path that match re.
// The manifest header block is skipped.
func SearchContent(path string, re *regexp.Regexp) ([]ContentMatch, error) {
	var matches []ContentMatch
	err := scanBody(path, func(n int, line string) {
		if re.MatchString(line) {
			matches = append(matches, ContentMatch{Line: n, Text: strings.TrimSpace(line)})
		}
	})
	return matches, err
}

// ReferencedConfigs returns the absolute paths of the files and directories
// the package's installer references through $DIS_CONFIG_FOLDER, in order of
// first appearance. Comment lines are ignored, as are paths that do not exist.
// Returns nil when the package has no configs folder.
func ReferencedConfigs(pkg PackageInfo) ([]string, error) {
	if pkg.ConfigsDir == "" {
		return nil, nil
	}

	var (
		paths []string
		seen  = map[string]bool{}
	)
	err := scanBody(pkg.InstallerPath, func(_ int, line string) {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			return
		}
		for _, m := range configRefRe.FindAllStringSubmatch(line, -1) {
			p := filepath.Join(pkg.ConfigsDir, m[1])
			if seen[p] {
				continue
			}
			seen[p] = true
			if _, err := os.Stat(p); err == nil {
				paths = append(paths, p)
			}
		}
	})
	return paths, err
}
