package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"lesiw.io/command/sys"
)

var installCmd = &cobra.Command{
	Use:   "install --distro <path-to-distro.yml> [package-name]",
	Short: "Install packages defined in a distro YAML file",
	Long: `Reads a distro YAML file and installs packages.

When called without a package name, walks the declared source folders to collect
installer manifests, resolves dependencies, and runs each installer in
topological order.

When called with a package name, runs that single installer directly, skipping
dependency resolution. Use this when you are confident all dependencies are
already satisfied.

Before running each installer script the following env vars are set:
  DIS_PKG_ROOT      - root of the source folder that owns this installer
  DIS_DISTRO        - os name from the distro YAML (e.g. "ubuntu")
  DIS_EXPORTS_FILE  - path to a per-installer temp file; write KEY=value lines
                      here to export values to downstream installers

Each installer is run inside a wrapper that sources ~/rc/configs-generated/bash_paths
and ~/rc/configs-generated/bash_aliases so that PATH additions from earlier
installers are available. Installer scripts can call 'dis tools ...' directly
because the dis binary directory is prepended to PATH.

Installers run on the host machine.

Examples:
  dis install --distro ~/dotfiles/dis/distros/home-server.yml
  dis install --distro ~/dotfiles/dis/distros/home-server.yml home-server/containers`,
	Args: cobra.MaximumNArgs(1),
	RunE: installCmdFn,
}

var distroFile string
var installCommonSources string
var installReinstall bool

func init() {
	installCmd.Flags().StringVarP(&distroFile, "distro", "d", "", "Path to the distro YAML file (required)")
	installCmd.Flags().StringVarP(&installCommonSources, "sources", "s", "", "Path to use for ${common_sources} (overrides auto-detection)")
	installCmd.Flags().BoolVar(&installReinstall, "reinstall", false, "Re-run installers even if already recorded as installed")
	_ = installCmd.MarkFlagRequired("distro")
	rootCmd.AddCommand(installCmd)
}

func installCmdFn(cmd *cobra.Command, args []string) error {
	ic, err := dis.NewInstallContextWithCache(distroFile, installCommonSources)
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

	if err := runner.RunPreconditions(ctx, ic); err != nil {
		return err
	}

	// If a package name is provided, run just that single installer (skipping dep resolution).
	if len(args) == 1 {
		pkgName := args[0]
		if err := runner.RunInstaller(ctx, ic, pkgName); err != nil {
			return err
		}
		fmt.Printf("✅ %s installed successfully.\n", pkgName)
		return nil
	}

	// Otherwise, resolve install order and run all packages.
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
