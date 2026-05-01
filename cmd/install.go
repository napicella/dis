package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"lesiw.io/command/sys"
)

var installCmd = &cobra.Command{
	Use:   "install --distro <path-to-distro.yml>",
	Short: "Install packages defined in a distro YAML file",
	Long: `Reads a distro YAML file, walks the declared source folders to collect
installer manifests, resolves dependencies, and runs each installer in
topological order.

Before sourcing each installer script the following env vars are set:
  DIS_PKG_ROOT      - root of the source folder that owns this installer
  DIS_DISTRO        - os name from the distro YAML (e.g. "ubuntu")
  DIS_BINDING       - path to dis/binding.sh
  DIS_EXPORTS_FILE  - path to a per-installer temp file; write KEY=value lines
                      here to export values to downstream installers

Installers run on the host machine.`,
	RunE: installCmdFn,
}

var distroFile string
var installCommonSources string
var installReinstall bool

func init() {
	installCmd.Flags().StringVarP(&distroFile, "distro", "d", "", "Path to the distro YAML file (required)")
	installCmd.Flags().StringVarP(&installCommonSources, "sources", "s", "", "Path to use for ${common_sources} (overrides auto-detection)")
	installCmd.Flags().BoolVar(&installReinstall, "reinstall", false, "Re-run all installers even if already recorded as installed")
	_ = installCmd.MarkFlagRequired("distro")
	rootCmd.AddCommand(installCmd)
}

func installCmdFn(cmd *cobra.Command, _ []string) error {
	ic, err := dis.NewInstallContext(distroFile, installCommonSources)
	if err != nil {
		return err
	}
	

	ctx := cmd.Context()
	runner, err := dis.NewInstaller(sys.Machine())
	if err != nil {
		return err
	}
	defer runner.Close()
	runner.Reinstall = installReinstall

	if err := runner.RunGenerators(ctx, ic); err != nil {
		return err
	}
	if err := runner.RunPreconditions(ctx, ic); err != nil {
		return err
	}

	toRun, err := ic.ResolveInstallOrder()
	if err != nil {
		return fmt.Errorf("resolving deps: %w", err)
	}

	for _, manifest := range toRun {
		if err := runner.RunInstaller(ctx, ic, manifest.Provides); err != nil {
			return fmt.Errorf("installer %q failed: %w", manifest.Provides, err)
		}
	}

	fmt.Println("✅ All packages installed successfully.")
	return nil
}
