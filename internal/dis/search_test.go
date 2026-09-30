package dis

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSearchContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "git.sh")
	writeFile(t, path, `### -- Manifest
### provides: common/status
### -- End
alias log='git log --oneline'
  alias status='git status'
# pane-border-status is a tmux option
`)

	tests := []struct {
		name    string
		pattern string
		want    []ContentMatch
	}{
		{
			name:    "skips the manifest block and trims the line",
			pattern: `status`,
			want: []ContentMatch{
				{Line: 5, Text: "alias status='git status'"},
				{Line: 6, Text: "# pane-border-status is a tmux option"},
			},
		},
		{
			name:    "several terms on the same line",
			pattern: `status.*git`,
			want:    []ContentMatch{{Line: 5, Text: "alias status='git status'"}},
		},
		{
			name:    "whole word",
			pattern: `\bstatus\b`,
			want: []ContentMatch{
				{Line: 5, Text: "alias status='git status'"},
				{Line: 6, Text: "# pane-border-status is a tmux option"},
			},
		},
		{
			name:    "no match",
			pattern: `nope`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchContent(path, regexp.MustCompile(tt.pattern))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReferencedConfigs(t *testing.T) {
	dir := t.TempDir()
	configs := filepath.Join(dir, "configs")
	writeFile(t, filepath.Join(configs, ".tmux.conf"), "")
	writeFile(t, filepath.Join(configs, "starship", "starship.toml"), "")
	writeFile(t, filepath.Join(configs, "git", "attrs"), "")
	writeFile(t, filepath.Join(configs, "backgrounds", "a.png"), "")

	installer := filepath.Join(dir, "installer.sh")
	writeFile(t, installer, `### -- Manifest
### provides: common/x
### -- End
# cp $DIS_CONFIG_FOLDER/commented.conf ~/
cp $DIS_CONFIG_FOLDER/.tmux.conf ~/
cp "$DIS_CONFIG_FOLDER/starship/starship.toml" ~/.config/
cp "${DIS_CONFIG_FOLDER}/git/attrs" ~/.config/git/attrs; echo "$DIS_CONFIG_FOLDER/.tmux.conf"
BG="$DIS_CONFIG_FOLDER/backgrounds/$THEME"
cp $DIS_CONFIG_FOLDER/missing.conf ~/
cp -r "$DIS_CONFIG_FOLDER/." ~/.config/app/
`)

	got, err := ReferencedConfigs(PackageInfo{InstallerPath: installer, ConfigsDir: configs})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(configs, ".tmux.conf"),
		filepath.Join(configs, "starship", "starship.toml"),
		filepath.Join(configs, "git", "attrs"),
		filepath.Join(configs, "backgrounds"),
		configs,
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}

	t.Run("no configs folder", func(t *testing.T) {
		got, err := ReferencedConfigs(PackageInfo{InstallerPath: installer})
		if err != nil || got != nil {
			t.Errorf("got %v, %v; want nil, nil", got, err)
		}
	})
}
