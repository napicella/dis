package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan --distro <path-to-distro.yml>",
	Short: "Show the ordered list of installers that would run, without executing anything",
	Long: `Resolves the full transitive dependency graph for the packages declared
in the distro YAML and prints the installers in the order they would run.

No generators, preconditions, or installers are executed. This is useful
for auditing what will be installed and debugging unexpected transitive
dependencies.

Example:
  dis plan --distro ~/dotfiles/dis/distros/home-server.yml`,
	RunE: planCmdFn,
}

var planDistroFile string
var planCommonSources string

func init() {
	planCmd.Flags().StringVarP(&planDistroFile, "distro", "d", "", "Path to the distro YAML file (required)")
	planCmd.Flags().StringVarP(&planCommonSources, "sources", "s", "", "Path to use for ${common_sources} (overrides auto-detection)")
	_ = planCmd.MarkFlagRequired("distro")
	rootCmd.AddCommand(planCmd)
}

func planCmdFn(cmd *cobra.Command, _ []string) error {
	ic, err := dis.NewInstallContext(planDistroFile, planCommonSources)
	if err != nil {
		return err
	}

	toRun, err := ic.ResolveInstallOrder()
	if err != nil {
		return fmt.Errorf("resolving deps: %w", err)
	}

	fmt.Printf("Distro: %s  (OS: %s)\n", planDistroFile, ic.Cfg.OS)
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
