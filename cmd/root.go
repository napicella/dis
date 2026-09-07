package cmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "dis",
	Short: "A tool for managing dotfiles and package installations",
	Long: `dis manages dotfiles and package installations.

Install state is recorded in: ~/.local/share/dis/installed.txt

Configuration file (optional): ~/.config/dis/config.yaml
  distro: ~/dotfiles/dis/distros/home-server.yml
  sources: ~/dotfiles/dis/packages`,
}

// Execute runs the root command with a background context.
func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}

func init() {
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
