package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/tools"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "",
}

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
