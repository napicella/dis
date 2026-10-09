package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"lesiw.io/command/sys"
)

var configCmd = &cobra.Command{
	Use:   "config [package-name]",
	Short: "Re-apply the configuration part of installed packages",
	Long: `Reads a distro YAML file and runs each installer without DIS_INSTALL set,
instructing the script to perform only its configuration steps.

This is useful when a config file has changed and you want to re-deploy it
without re-running the full installation (downloading binaries, apt-get, etc.).
The already-installed state is ignored — every matched package is always run —
and no install state is written after the run.

With a package name, only that package is configured. Add --with-deps to also
configure the packages it depends on, in dependency order (e.g. every package of
a bundle).

Installer scripts opt into the distinction by checking DIS_INSTALL:

  if [[ -n "${DIS_INSTALL:-}" ]]; then
    # install-only steps (e.g. apt install, go install, curl …)
  fi
  # configuration steps always run (e.g. cp config files, dis tools add-rc-*)

Before running each script the following env vars are set (same as install):
  DIS_PKG_ROOT      - root of the source folder that owns this installer
  DIS_DISTRO        - os name from the distro YAML (e.g. "ubuntu")
  DIS_EXPORTS_FILE  - path to a per-installer temp file; write KEY=value lines
                      here to export values to downstream installers
  DIS_INSTALL       - set to "1" only by 'dis install'; absent when running
                      'dis config' so scripts can skip install-only steps

Examples:
  dis config --distro ~/dotfiles/distros/laptop/laptop.yml
  dis config --distro ~/dotfiles/distros/laptop/laptop.yml common/starship
  dis config --distro ~/dotfiles/distros/laptop/laptop.yml --with-deps bundle/shell`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: bindSharedConfigFlags,
	RunE:    configCmdFn,
}

var configWithDeps bool

func init() {
	configCmd.Flags().String("distro", "", "Path to the distro YAML file")
	configCmd.Flags().BoolVar(&configWithDeps, "with-deps", false, "With a package name, also configure the packages it depends on, dependencies first")
	rootCmd.AddCommand(configCmd)
}

func configCmdFn(cmd *cobra.Command, args []string) error {
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

	if err := runner.RunPreconditions(ctx, ic); err != nil {
		return err
	}

	// With --with-deps, configure the named package and everything it depends on.
	if len(args) == 1 && configWithDeps {
		pkgName := args[0]
		toRun, err := ic.ResolveInstallOrderFor(pkgName)
		if err != nil {
			return renderPackageNotFound(cmd, err)
		}
		for _, manifest := range toRun {
			if err := runner.RunConfig(ctx, ic, manifest.Provides); err != nil {
				return fmt.Errorf("config %q failed: %w", manifest.Provides, err)
			}
		}
		fmt.Printf("✅ %s and its dependencies configured successfully.\n", pkgName)
		return nil
	}

	// If a package name is provided, configure just that single package.
	if len(args) == 1 {
		pkgName := args[0]
		if err := runner.RunConfig(ctx, ic, pkgName); err != nil {
			return renderPackageNotFound(cmd, err)
		}
		fmt.Printf("✅ %s configured successfully.\n", pkgName)
		return nil
	}

	// Otherwise, resolve install order and re-apply configs for all packages.
	toRun, err := ic.ResolveInstallOrder()
	if err != nil {
		return fmt.Errorf("resolving deps: %w", err)
	}

	for _, manifest := range toRun {
		if err := runner.RunConfig(ctx, ic, manifest.Provides); err != nil {
			return fmt.Errorf("config %q failed: %w", manifest.Provides, err)
		}
	}

	fmt.Println("✅ All packages configured successfully.")
	return nil
}
