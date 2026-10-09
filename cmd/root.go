package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "dis",
	Short: "A tool for managing dotfiles and package installations",
	Long: `dis manages dotfiles and package installations.

Install state is recorded in: ~/.local/share/dis/installed.txt

Configuration file (optional, written by 'dis pull'): ~/.config/dis/config.yaml
  distro: ~/dotfiles/distros/laptop/laptop.yml`,
}

// Execute runs the root command with a background context.
func Execute() {
	if err := execute(context.Background(), os.Args[1:]); err != nil {
		os.Exit(1)
	}
}

// execute sets up the command tree and runs args; Execute and the tests run
// commands through it.
func execute(ctx context.Context, args []string) error {
	setupCompletionCmd()
	rootCmd.SetArgs(args)
	defer rootCmd.SetArgs(nil)
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.Version = buildVersion()
	cobra.OnInitialize(initConfig)
}

// initConfig loads the optional dis config file from ~/.config/dis/config.yaml
// (or a config.yaml in the current directory). CLI flags always take precedence.
func initConfig() {
	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "dis"))
	}
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	// Silently ignore missing config file — it's optional.
	_ = viper.ReadInConfig()
}

// unknownSubcommand is the Args of a parent command: an argument there is a
// subcommand cobra didn't find, so it fails as an unknown command does on the
// root, with suggestions. Cobra would otherwise print the parent's help and
// exit 0, so `dis tools typo` in an installer would silently do nothing.
func unknownSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
	if !cmd.DisableSuggestions {
		if cmd.SuggestionsMinimumDistance <= 0 {
			cmd.SuggestionsMinimumDistance = 2 // cobra's default
		}
		if s := cmd.SuggestionsFor(args[0]); len(s) > 0 {
			msg += "\n\nDid you mean this?\n\t" + strings.Join(s, "\n\t") + "\n"
		}
	}
	return errors.New(msg)
}

// showHelp is the Run of a parent command called without a subcommand. A
// command needs a Run for cobra to check its Args.
func showHelp(cmd *cobra.Command, _ []string) error { return cmd.Help() }

// setupCompletionCmd adds cobra's completion command now rather than when the
// root command runs, so it gets the same unknown-subcommand check.
func setupCompletionCmd() {
	rootCmd.InitDefaultCompletionCmd()
	if c, _, err := rootCmd.Find([]string{"completion"}); err == nil && c != rootCmd {
		c.Args, c.RunE = unknownSubcommand, showHelp
	}
}
