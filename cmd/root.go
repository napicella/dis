package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dis",
	Short: "A tool for managing dotfiles and package installations",
	Long: `dis manages dotfiles and package installations.

Install state is recorded in: ~/.local/share/dis/installed.txt`,
}

// Execute runs the root command with a background context.
func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
