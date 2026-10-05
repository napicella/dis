package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	Long: `Prints the packages that have been installed by dis, one per line.

To see where packages come from, use 'dis sources'; to find a package that
is available but not installed, use 'dis search packages REGEX'.`,
	Args: cobra.NoArgs,
	RunE: listCmdFn,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func listCmdFn(_ *cobra.Command, _ []string) error {
	state, err := dis.DefaultState()
	if err != nil {
		return err
	}
	pkgs, err := state.ListInstalled()
	if err != nil {
		return fmt.Errorf("reading install state: %w", err)
	}

	for _, pkg := range pkgs {
		fmt.Println(pkg)
	}
	return nil
}
