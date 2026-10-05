package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"lesiw.io/command/sys"
)

var installCmd = &cobra.Command{
	Use:   "install [package-name]",
	Short: "Install packages defined in a distro YAML file",
	Long: `Reads a distro YAML file and installs packages.

When called without a package name, walks the declared source folders to collect
installer manifests, resolves dependencies, and runs each installer in
topological order.

When called with a package name, runs that single installer only. Before
running it, dis checks that the packages it depends on are recorded as
installed, and fails if one is not. The dependencies of an installed package are
not checked, and neither is a package that is already installed and will be
skipped. --no-deps-check skips the check, for dependencies installed outside
dis. Add --with-deps to also run the installers it depends on, in dependency
order (e.g. every package of a bundle). Already-installed packages are skipped
as usual, unless --reinstall is set.

Before running each installer script the following env vars are set:
  DIS_PACKAGE       - name of the package being installed; 'dis tools add-rc-*'
                      records it as the owner of the sections it writes
  DIS_PKG_ROOT      - root of the source folder that owns this installer
  DIS_DISTRO        - os name from the distro YAML (e.g. "ubuntu")
  DIS_EXPORTS_FILE  - path to a per-installer temp file; write KEY=value lines
                      here to export values to downstream installers
  DIS_INSTALL       - set to "1"; use this to guard install-only steps in scripts

Each installer is run inside a wrapper that sources ~/rc/configs-generated/bash_paths
and ~/rc/configs-generated/bash_aliases so that PATH additions from earlier
installers are available. Installer scripts can call 'dis tools ...' directly
because the dis binary directory is prepended to PATH.

Installers run on the host machine.

Examples:
  dis install --distro ~/dotfiles/dis/distros/home-server.yml
  dis install --distro ~/dotfiles/dis/distros/home-server.yml home-server/containers
  dis install --distro ~/dotfiles/dis/distros/home-server.yml --with-deps bundle/cli-tools`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: bindSharedConfigFlags,
	RunE:    installCmdFn,
}

var (
	installReinstall   bool
	installWithDeps    bool
	installNoDepsCheck bool
)

func init() {
	installCmd.Flags().String("distro", "", "Path to the distro YAML file")
	installCmd.Flags().BoolVar(&installReinstall, "reinstall", false, "Re-run installers even if already recorded as installed")
	installCmd.Flags().BoolVar(&installWithDeps, "with-deps", false, "With a package name, also run the installers it depends on, dependencies first")
	installCmd.Flags().BoolVar(&installNoDepsCheck, "no-deps-check", false, "With a package name and without --with-deps, run it even if its dependencies are not recorded as installed")
	rootCmd.AddCommand(installCmd)
}

func installCmdFn(cmd *cobra.Command, args []string) error {
	distroFile := viper.GetString("distro")
	if distroFile == "" {
		return fmt.Errorf("required flag \"distro\" not set and not found in config file")
	}

	ic, err := dis.NewInstallContextWithCache(distroFile)
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

	// With --with-deps, run the named package and everything it depends on.
	if len(args) == 1 && installWithDeps {
		pkgName := args[0]
		toRun, err := ic.ResolveInstallOrderFor(pkgName)
		if err != nil {
			return renderPackageNotFound(cmd, err)
		}
		for _, manifest := range toRun {
			if err := runner.RunInstaller(ctx, ic, manifest.Provides); err != nil {
				return fmt.Errorf("installer %q failed: %w", manifest.Provides, err)
			}
		}
		fmt.Printf("✅ %s and its dependencies installed successfully.\n", pkgName)
		return nil
	}

	// If a package name is provided, run just that single installer, once its
	// dependencies are known to be installed.
	if len(args) == 1 {
		pkgName := args[0]
		if !installNoDepsCheck {
			if err := checkDepsInstalled(cmd, ic, pkgName); err != nil {
				return err
			}
		}
		if err := runner.RunInstaller(ctx, ic, pkgName); err != nil {
			return renderPackageNotFound(cmd, err)
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

// checkDepsInstalled fails when the installer of pkgName is about to run but
// some of the packages it depends on are not recorded as installed. It passes
// when the package is already installed and will be skipped (no --reinstall).
func checkDepsInstalled(cmd *cobra.Command, ic *dis.InstallContext, pkgName string) error {
	if !installReinstall {
		installed, err := ic.State.IsInstalled(pkgName)
		if err != nil {
			return fmt.Errorf("checking install state for %q: %w", pkgName, err)
		}
		if installed {
			return nil
		}
	}
	missing, err := ic.MissingDeps(pkgName)
	if err != nil {
		return renderPackageNotFound(cmd, err)
	}
	if len(missing) > 0 {
		return renderMissingDeps(cmd, pkgName, missing, installReinstall)
	}
	return nil
}
