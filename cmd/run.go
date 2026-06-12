package cmd

import (
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"lesiw.io/command/sys"
)

var runCmd = &cobra.Command{
	Use:   "run --distro <path-to-distro.yml> <package-name>",
	Short: "Run a single installer directly, skipping dependency resolution",
	Long: `Runs a single installer by its fully-qualified package name (e.g.
"home-server/containers"), without evaluating or running any of its
declared dependencies. Use this when you are confident all dependencies
are already satisfied.

The distro YAML is still required to locate the installer file, resolve
the target OS, and inject any params declared under the package key.

Examples:
  dotfiles run --distro ~/dotfiles/dis/distros/home-server.yml home-server/containers`,
	Args: cobra.ExactArgs(1),
	RunE: runCmdFn,
}

var runDistroFile string
var runCommonSources string
var runReinstall bool

func init() {
	runCmd.Flags().StringVarP(&runDistroFile, "distro", "d", "", "Path to the distro YAML file (required)")
	runCmd.Flags().StringVarP(&runCommonSources, "sources", "s", "", "Path to use for ${common_sources} (overrides auto-detection)")
	runCmd.Flags().BoolVar(&runReinstall, "reinstall", false, "Re-run the installer even if already recorded as installed")
	_ = runCmd.MarkFlagRequired("distro")
	rootCmd.AddCommand(runCmd)
}

func runCmdFn(cmd *cobra.Command, args []string) error {
	pkgName := args[0]
	ctx := cmd.Context()

	ic, err := dis.NewInstallContextWithCache(runDistroFile, runCommonSources)
	if err != nil {
		return err
	}

	runner, err := dis.NewInstaller(sys.Machine())
	if err != nil {
		return err
	}
	defer runner.Close()
	runner.Reinstall = runReinstall

	if err := runner.RunPreconditions(ctx, ic); err != nil {
		return err
	}
	if err := runner.RunInstaller(ctx, ic, pkgName); err != nil {
		return err
	}

	fmt.Printf("✅ %s installed successfully.\n", pkgName)
	return nil
}
