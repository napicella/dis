package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show the ordered list of installers that would run, without executing anything",
	Long: `Resolves the full transitive dependency graph for the packages declared
in the distro YAML and prints the installers in the order they would run.

No preconditions, or installers are executed. This is useful
for auditing what will be installed and debugging unexpected transitive
dependencies.

Example:
  dis plan --distro ~/dotfiles/dis/distros/home-server.yml`,
	RunE: planCmdFn,
}

func init() {
	planCmd.Flags().String("distro", "", "Path to the distro YAML file")
	planCmd.Flags().String("sources", "", "Path to use for ${common_sources} (overrides auto-detection)")
	_ = viper.BindPFlag("distro", planCmd.Flags().Lookup("distro"))
	_ = viper.BindPFlag("sources", planCmd.Flags().Lookup("sources"))
	rootCmd.AddCommand(planCmd)
}

func planCmdFn(_ *cobra.Command, _ []string) error {
	distroFile := viper.GetString("distro")
	if distroFile == "" {
		return fmt.Errorf("required flag \"distro\" not set and not found in config file")
	}
	commonSources := viper.GetString("sources")

	ic, err := dis.NewInstallContext(distroFile, commonSources)
	if err != nil {
		return err
	}

	toRun, err := ic.ResolveInstallOrder()
	if err != nil {
		return fmt.Errorf("resolving deps: %w", err)
	}

	fmt.Printf("Distro: %s  (OS: %s)\n", distroFile, ic.Cfg.OS)
	fmt.Printf("Packages to install (%d total, in order):\n\n", len(toRun))
	for i, m := range toRun {
		fmt.Printf("  %3d. %-40s  %s\n", i+1, m.Provides, m.InstallerPath)
		fmt.Printf("     package_root:   %s\n", m.PkgRoot)
		if len(m.RequiresEnv) > 0 {
			fmt.Printf("       requires_env: %v\n", m.RequiresEnv)
		}
	}
	return nil
}
