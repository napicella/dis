package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new dis workspace in the current directory",
	Long: `Creates a minimal dis workspace in the current directory:

  dis.ws.yml                          - workspace file declaring package roots
  distro.yml                          - example distro configuration
  packages/hello/installers/hello.sh  - example installer with a full manifest

Edit these files to match your OS, package structure, and installer logic,
then run 'dis install --distro distro.yml' to execute the installation.`,
	RunE: initCmdFn,
}

type initOpt uint

var argString string

const (
	initManifestOpt initOpt = 1 << iota
	initDistroOpt   initOpt = 1 << iota
	initWsOpt       initOpt = 1 << iota
)

func init() {
	initCmd.Flags().StringVarP(&argString, "args", "a", "dmw",
		"files to generate: (d)istro, (m)anifest, and (w)orkspace)")
	rootCmd.AddCommand(initCmd)
}

func parseInitArgs(args string) (initOpt, error) {
	var o initOpt

	if args == "" {
		args = "dmw"
	}
	for _, c := range args {
		switch c {
		case 'm':
			o = o | initManifestOpt
		case 'd':
			o = o | initDistroOpt
		case 'w':
			o = o | initWsOpt
		default:
			return 0, fmt.Errorf("invalid %q option", c)
		}
	}

	return o, nil
}

func initCmdFn(_ *cobra.Command, _ []string) error {
	genOpt, err := parseInitArgs(argString)
	if err != nil {
		return err
	}
	files := []struct {
		path    string
		content string
		fmask   initOpt
	}{
		{
			path:  "dis.ws.yml",
			fmask: initWsOpt,
			content: `# dis workspace file
# Declares the package roots that dis will walk when this directory is listed
# as a source in a distro YAML.
#
# Each entry may optionally specify a configs directory that installers can
# access via the DIS_CONFIG_FOLDER environment variable.
packages:
  - root: ./packages/hello
    configs: ./packages/hello/configs
`,
		},
		{
			path:  "distro.yml",
			fmask: initDistroOpt,
			content: `# dis distro file
# Describes what to install and where to find the installers.
os: ubuntu   # target OS: ubuntu | amazon_linux | all

# Static parameters injected into installers via requires_env.
# A plain string value is global — available to every package that declares it.
# An object with value + packages is scoped — injected only into the listed packages.
parameters:
  MY_PARAM: my-value   # global

  MY_SCOPED_PARAM:
    value: scoped-value
    packages: [hello/greet]   # only injected into hello/greet

# Source directories dis will search for installer manifests.
# ${common_sources} resolves to the dis built-in packages (~/.local/share/dis/packages).
sources:
  - .
  - ${common_sources}

# Ordered list of packages to install.
packages:
  - hello/greet
`,
		},
		{
			path:  filepath.Join("packages", "hello", "installers", "hello.sh"),
			fmask: initManifestOpt,
			content: `#!/usr/bin/env bash
### -- Manifest
### provides: hello/greet
### depends_on: []
### distro: [ubuntu]
### requires_env: [MY_PARAM, MY_SCOPED_PARAM]
### -- End

echo "Hello from dis! MY_PARAM=${MY_PARAM}  MY_SCOPED_PARAM=${MY_SCOPED_PARAM}"
`,
		},
	}

	for _, f := range files {
		if f.fmask&genOpt == f.fmask {
			if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
				return fmt.Errorf("creating directory for %s: %w", f.path, err)
			}
			if _, err := os.Stat(f.path); err == nil {
				fmt.Printf("  skip  %s (already exists)\n", f.path)
				continue
			}
			if err := os.WriteFile(f.path, []byte(f.content), 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", f.path, err)
			}
			fmt.Printf("  create %s\n", f.path)
		}
	}

	fmt.Println("\n✅ Done. Next steps:")
	fmt.Println("  1. Edit distro.yml — set the correct os and packages.")
	fmt.Println("  2. Edit packages/hello/installers/hello.sh — implement your installer.")
	fmt.Println("  3. Run: dis plan --distro distro.yml")
	fmt.Println("  4. Run: dis install --distro distro.yml")
	return nil
}
