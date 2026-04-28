package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	Long:  `Prints the packages that have been installed by dis, one per line.`,
	RunE:  listCmdFn,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func listCmdFn(cmd *cobra.Command, _ []string) error {
	pkgs, err := dis.ListInstalled()
	if err != nil {
		return fmt.Errorf("reading install state: %w", err)
	}

	for _, pkg := range pkgs {
		fmt.Println(pkg)
	}
	return nil
}
