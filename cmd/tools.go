package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/napicella/dis/internal/rcstate"
	"github.com/napicella/dis/internal/tools"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Collection of helper tools for use inside installer scripts",
}

// ── GNOME shortcut ────────────────────────────────────────────────────────────

var createGnomeShortCmd = &cobra.Command{
	Use:   "create-gnome-shortcut",
	Short: "Create a GNOME keyboard shortcut.",
	Long: `Create a GNOME keyboard shortcut with the name provided. For example:
dis tools create-gnome-shortcut --name test-key --cmd date --bind "<Super>Insert"

creates a shortcut named "test-key" that runs "date" when pressing the keys Super + Insert.
`,
	RunE: createGnomeShortcutCmdFn,
}

var (
	name    string
	command string
	binding string
)

func init() {
	createGnomeShortCmd.Flags().StringVar(&name, "name", "", "shortcut name")
	createGnomeShortCmd.Flags().StringVar(&command, "cmd", "", "command to run")
	createGnomeShortCmd.Flags().StringVar(&binding, "bind", "", "keybinding (e.g. <Super>q)")

	createGnomeShortCmd.MarkFlagRequired("name")
	createGnomeShortCmd.MarkFlagRequired("cmd")
	createGnomeShortCmd.MarkFlagRequired("bind")

	toolsCmd.AddCommand(createGnomeShortCmd)

	// RC helpers
	rcFlags(addRCInitCmd)
	addRCPathCmd.Flags().StringVar(&rcName, "name", "", "section identifier (unique per file)")
	addRCPathCmd.Flags().StringVar(&rcContent, "content", "", "content to write into the section")
	addRCPathCmd.Flags().StringArrayVar(&rcPaths, "path", nil, "directory to prepend to PATH, skipped if PATH already has it (repeatable)")
	addRCPathCmd.MarkFlagRequired("name") //nolint:errcheck
	addRCPathCmd.MarkFlagsOneRequired("content", "path")
	addRCPathCmd.MarkFlagsMutuallyExclusive("content", "path")
	rcFlags(addRCAliasesCmd)
	addRCAliasesCmd.Flags().StringVar(&rcOwner, "owner", "", "lock the section to this owner (package name); only the same owner can overwrite or remove it")
	rcFlags(addHomeRCCmd)
	rmRCSectionCmd.Flags().StringVar(&rcFile, "file", "", "generated file: "+strings.Join(rcstate.Files, ", "))
	rmRCSectionCmd.Flags().StringVar(&rcName, "name", "", "section identifier (unique per file)")
	rmRCSectionCmd.Flags().StringVar(&rcOwner, "owner", "", "owner the section is locked to, if any; a locked section can only be removed by its owner")
	rmRCSectionCmd.MarkFlagRequired("file") //nolint:errcheck
	rmRCSectionCmd.MarkFlagRequired("name") //nolint:errcheck
	toolsCmd.AddCommand(addRCInitCmd)
	toolsCmd.AddCommand(addRCPathCmd)
	toolsCmd.AddCommand(addRCAliasesCmd)
	toolsCmd.AddCommand(rmRCSectionCmd)
	toolsCmd.AddCommand(addHomeRCCmd)

	exportEnvCmd.Flags().StringVar(&exportKey, "key", "", "key to export")
	exportEnvCmd.Flags().StringVar(&exportValue, "value", "", "value to export")
	toolsCmd.AddCommand(exportEnvCmd)

	rootCmd.AddCommand(toolsCmd)
}

func createGnomeShortcutCmdFn(cmd *cobra.Command, _ []string) error {
	index, path, err := tools.CreateGNOMEShortcut(name, command, binding)
	if err != nil {
		return err
	}

	fmt.Println("Created shortcut:")
	fmt.Printf("  custom%d\n", index)
	fmt.Printf("  %s -> %s (%s)\n", binding, name, command)
	fmt.Printf("  path: %s\n", path)

	return nil
}

// ── RC helpers ────────────────────────────────────────────────────────────────

var (
	rcName    string
	rcContent string
	rcOwner   string
	rcPaths   []string
	rcFile    string
)

// rcFlags registers --name and --content on a command and marks them required.
func rcFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&rcName, "name", "", "section identifier (unique per file)")
	cmd.Flags().StringVar(&rcContent, "content", "", "content to write into the section")
	cmd.MarkFlagRequired("name")    //nolint:errcheck
	cmd.MarkFlagRequired("content") //nolint:errcheck
}

// rcFilePath returns $HOME/<rel>, failing if $HOME is unset.
func rcFilePath(rel string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, rel), nil
}

// rcSectionsHelp is shared by the add-rc-* commands.
const rcSectionsHelp = `The generated files are rendered from ~/.local/share/dis/sections.yaml:
don't edit them, the next write overwrites the change (after backing up the
file). An updated section keeps its place in the file.

Run from an installer, the section is owned by the installer's package
($DIS_PACKAGE). The last writer wins: a section is replaced, and changes owner,
unless it is locked to another owner.`

// upsertRCSection writes a section to a generated file, owned by the running
// installer's package. A section locked to another owner is skipped with a
// warning, as before.
func upsertRCSection(file, content string) error {
	store, err := rcstate.DefaultStore()
	if err != nil {
		return err
	}
	sec := rcstate.Section{Name: rcName, Owner: os.Getenv("DIS_PACKAGE"), Content: content}
	if rcOwner != "" {
		sec.Owner, sec.Locked = rcOwner, true
	}
	err = store.Update(func(st *rcstate.State) error { return st.Upsert(file, sec) })
	if errors.Is(err, rcstate.ErrLocked) {
		fmt.Fprintf(os.Stderr, "skipping section: %v\n", err)
		return nil
	}
	return err
}

var addRCInitCmd = &cobra.Command{
	Use:   "add-rc-init",
	Short: "Upsert a named section in ~/rc/configs-generated/bash_init",
	Long: `Upsert a named section in ~/rc/configs-generated/bash_init.

bash_init is intended for code that must run in interactive shells (e.g.
tool activation hooks). It is sourced by ~/.bashrc, not by the dis wrapper.

` + rcSectionsHelp + `

Example:
  dis tools add-rc-init \
    --name "Autojump" \
    --content '[[ -s ~/.autojump/etc/profile.d/autojump.sh ]] && source ~/.autojump/etc/profile.d/autojump.sh'
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return upsertRCSection("bash_init", rcContent)
	},
}

var addRCPathCmd = &cobra.Command{
	Use:   "add-rc-path",
	Short: "Upsert a named section in ~/rc/configs-generated/bash_paths",
	Long: `Upsert a named section in ~/rc/configs-generated/bash_paths.

bash_paths is sourced by the dis wrapper before each installer runs, so PATH
additions written here are available to subsequent installers in the same run.

Use --path for PATH entries: each dir is prepended to PATH, and skipped if PATH
already has it, so shells started from another shell (tmux, herdr) don't repeat
it. The first --path ends up first in PATH. Use --content for other exports.

` + rcSectionsHelp + `

Examples:
  dis tools add-rc-path \
    --name "Mise path" \
    --path '$HOME/.local/share/mise/shims'

  dis tools add-rc-path \
    --name "Editor default" \
    --content 'export EDITOR="${EDITOR:-vim}"'
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		content := rcContent
		if len(rcPaths) > 0 {
			content = tools.PathPrependContent(rcPaths)
		}
		return upsertRCSection("bash_paths", content)
	},
}

var addRCAliasesCmd = &cobra.Command{
	Use:   "add-rc-aliases",
	Short: "Upsert a named section in ~/rc/configs-generated/bash_aliases",
	Long: `Upsert a named section in ~/rc/configs-generated/bash_aliases.

bash_aliases is sourced by the dis wrapper before each installer runs.

Use --owner to lock the section to a package: only that package can overwrite
or remove it.

` + rcSectionsHelp + `

Examples:
  dis tools add-rc-aliases \
    --name 'ls aliases' \
    --owner 'common/eza' \
    --content "alias ls='eza --icons=auto'"
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return upsertRCSection("bash_aliases", rcContent)
	},
}

var rmRCSectionCmd = &cobra.Command{
	Use:   "rm-rc-section",
	Short: "Remove a named section from a generated rc file",
	Long: `Remove a named section from a file in ~/rc/configs-generated:
bash_paths, bash_aliases or bash_init.

Use this when a package stops providing a section it used to add, so the stale
section does not linger. Removing a section that is not present is a no-op. A
section locked with --owner can only be removed by the same owner.

Example:
  dis tools rm-rc-section --file bash_aliases --name 'Coder'
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !rcstate.IsFile(rcFile) {
			return fmt.Errorf("--file must be one of %s, not %q", strings.Join(rcstate.Files, ", "), rcFile)
		}
		store, err := rcstate.DefaultStore()
		if err != nil {
			return err
		}
		err = store.Update(func(st *rcstate.State) error {
			_, err := st.Remove(rcFile, rcName, rcOwner)
			return err
		})
		if errors.Is(err, rcstate.ErrLocked) {
			fmt.Fprintf(os.Stderr, "skipping section: %v\n", err)
			return nil
		}
		return err
	},
}

var addHomeRCCmd = &cobra.Command{
	Use:   "add-home-rc",
	Short: "Upsert a named section in ~/.bashrc",
	Long: `Upsert a named section directly in ~/.bashrc.

Use this to wire the dotfiles .bashrc or similar top-level shell configuration
into the user's home ~/.bashrc. The file is created if it does not exist.

Example:
  dis tools add-home-rc \
    --name "bashrc" \
    --content 'if [ -f /path/to/dotfiles/.bashrc ]; then . /path/to/dotfiles/.bashrc; fi'
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := rcFilePath(".bashrc")
		if err != nil {
			return err
		}
		return tools.AddRCSection(path, rcName, rcContent, "")
	},
}

var (
	exportKey   string
	exportValue string
)

var exportEnvCmd = &cobra.Command{
	Use: "export-env",
	RunE: func(cmd *cobra.Command, args []string) error {
		exportFile, exists := os.LookupEnv("DIS_EXPORTS_FILE")
		if !exists || exportFile == "" {
			return errors.New("failed to export env variable: DIS_EXPORTS_FILE env variable is empty")
		}
		f, err := os.OpenFile(exportFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = fmt.Fprintf(f, "%s=%s\n", exportKey, exportValue)
		return err
	},
}
