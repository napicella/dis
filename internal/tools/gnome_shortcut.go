package tools

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	baseSchema     = "org.gnome.settings-daemon.plugins.media-keys"
	customBasePath = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/"
)

// CreateGNOMEShortcut creates a GNOME custom keyboard shortcut
// and returns (customIndex, gsettingsPath, error).
func CreateGNOMEShortcut(name, command, binding string) (int, string, error) {
	existing, err := gsettingsGet(baseSchema, "custom-keybindings")
	if err != nil {
		return 0, "", err
	}

	used := extractIndices(existing)
	next := firstUnused(used)

	newPath := fmt.Sprintf("%scustom%d/", customBasePath, next)

	updated := appendPath(existing, newPath)
	if err := gsettingsSet(baseSchema, "custom-keybindings", updated); err != nil {
		return 0, "", err
	}

	schema := fmt.Sprintf("%s.custom-keybinding:%s", baseSchema, newPath)

	if err := gsettingsSet(schema, "name", quote(name)); err != nil {
		return 0, "", err
	}
	if err := gsettingsSet(schema, "command", quote(command)); err != nil {
		return 0, "", err
	}
	if err := gsettingsSet(schema, "binding", quote(binding)); err != nil {
		return 0, "", err
	}

	return next, newPath, nil
}

// extractIndices finds all used customN indices in GNOME settings.
func extractIndices(s string) map[int]bool {
	re := regexp.MustCompile(`custom(\d+)/`)
	matches := re.FindAllStringSubmatch(s, -1)

	used := make(map[int]bool)
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err == nil {
			used[n] = true
		}
	}
	return used
}

// firstUnused returns the smallest unused integer index.
func firstUnused(used map[int]bool) int {
	i := 0
	for {
		if !used[i] {
			return i
		}
		i++
	}
}

// appendPath appends a new custom keybinding path to GNOME list.
func appendPath(existing, newPath string) string {
	existing = strings.TrimSpace(existing)

	if existing == "@as []" || existing == "[]" {
		return fmt.Sprintf("['%s']", newPath)
	}

	trimmed := strings.TrimSuffix(existing, "]")
	return fmt.Sprintf("%s, '%s']", trimmed, newPath)
}

// gsettingsGet reads a gsettings value.
func gsettingsGet(schema, key string) (string, error) {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// gsettingsSet writes a gsettings value.
func gsettingsSet(schema, key, value string) error {
	cmd := exec.Command("gsettings", "set", schema, key, value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

// quote wraps a string in single quotes for gsettings.
func quote(s string) string {
	return fmt.Sprintf("'%s'", s)
}