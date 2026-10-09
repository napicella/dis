package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"lesiw.io/command/sys"
)

var editCmd = &cobra.Command{
	Use:   "edit <package-name>",
	Short: "Open the installer and config files of a package in an editor",
	Long: `Opens a package in an editor, all its files at once: the installer
script first, then the config files it references through $DIS_CONFIG_FOLDER
(the ones 'dis search configs' lists). Referenced directories are expanded to
the text files they contain, so binary configs such as wallpapers are skipped.

The package may be given by its short name, the part after the last "/"
(e.g. "starship" for "common/starship"), when no other package shares it.

  --installer  open the installer only
  --configs    open the config files only

The editor is the first one set among:
  DIS_EDITOR  - an editor used only by dis, e.g. a GUI editor you do not want
                other tools (less, git, crontab) to pick up
  VISUAL
  EDITOR
  vi          - when none of the above is set

The value is run through the shell as git does, so it may carry arguments,
e.g. DIS_EDITOR="code --wait". The editor must not return until the files are
closed: GUI editors need their wait flag, otherwise --apply runs too early.

With --apply, the package's configuration steps are re-run (as 'dis config
<package-name>' does) once the editor exits successfully. If the editor exits
with an error, nothing is applied.

Examples:
  dis edit common/starship
  dis edit common/starship --apply
  dis edit common/git --installer
  DIS_EDITOR="code --wait" dis edit starship`,
	Args:    cobra.ExactArgs(1),
	PreRunE: bindSharedConfigFlags,
	RunE:    editCmdFn,
}

var (
	editApply     bool
	editConfigs   bool
	editInstaller bool
)

func init() {
	editCmd.Flags().String("distro", "", "Path to the distro YAML file")
	editCmd.Flags().BoolVar(&editApply, "apply", false,
		"re-run the package's configuration steps after the editor exits")
	editCmd.Flags().BoolVar(&editConfigs, "configs", false,
		"open the config files only, not the installer")
	editCmd.Flags().BoolVar(&editInstaller, "installer", false,
		"open the installer only, not the config files")
	editCmd.MarkFlagsMutuallyExclusive("configs", "installer")
	rootCmd.AddCommand(editCmd)
}

func editCmdFn(cmd *cobra.Command, args []string) error {
	distroFile := viper.GetString("distro")
	if distroFile == "" {
		return fmt.Errorf("required flag \"distro\" not set and not found in config file")
	}

	ic, err := dis.NewInstallContextWithCache(distroFile)
	if err != nil {
		return err
	}
	pkg, err := ic.FindPackage(args[0])
	if err != nil {
		return renderPackageNotFound(cmd, err)
	}
	pkgName := pkg.Provides
	if pkgName != args[0] {
		fmt.Fprintf(cmd.ErrOrStderr(), "==> Editing %s\n", pkgName)
	}

	// From here on, failures are not usage mistakes.
	cmd.SilenceUsage = true

	files, err := filesToEdit(pkg, editInstaller, editConfigs)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	editor := resolveEditor(os.Getenv)
	if err := runEditor(ctx, editor, files, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
		return fmt.Errorf("editor %q: %w", editor, err)
	}
	if !editApply {
		return nil
	}

	runner, err := dis.NewInstaller(sys.Machine())
	if err != nil {
		return err
	}
	defer runner.Close()
	if err := runner.RunPreconditions(ctx, ic); err != nil {
		return err
	}
	if err := runner.RunConfig(ctx, ic, pkgName); err != nil {
		return err
	}
	fmt.Printf("✅ %s configured successfully.\n", pkgName)
	return nil
}

// filesToEdit returns the files to open for pkg: its installer, the configs it
// references, or both (the default). The installer comes first, as the file the
// configs are read from.
func filesToEdit(pkg dis.PackageInfo, installerOnly, configsOnly bool) ([]string, error) {
	var files []string
	if !configsOnly {
		files = append(files, pkg.InstallerPath)
	}
	if installerOnly {
		return files, nil
	}

	refs, err := dis.ReferencedConfigs(pkg)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", pkg.InstallerPath, err)
	}
	configs, err := expandConfigFiles(refs)
	if err != nil {
		return nil, err
	}
	if configsOnly && len(configs) == 0 {
		return nil, fmt.Errorf("package %q references no text config files; its installer is %s",
			pkg.Provides, pkg.InstallerPath)
	}
	return append(files, configs...), nil
}

// resolveEditor returns the editor command dis edit runs: the first non-empty
// of DIS_EDITOR, VISUAL and EDITOR, falling back to vi.
func resolveEditor(getenv func(string) string) string {
	for _, key := range []string{"DIS_EDITOR", "VISUAL", "EDITOR"} {
		if v := getenv(key); v != "" {
			return v
		}
	}
	return "vi"
}

// runEditor opens files in editor and waits for it to exit. Like git, it runs
// the editor through sh so the value may carry arguments ("code --wait") while
// each file is still passed as a single, unsplit argument.
func runEditor(ctx context.Context, editor string, files []string, stdin io.Reader, stdout, stderr io.Writer) error {
	args := append([]string{"-c", editor + ` "$@"`, editor}, files...)
	c := exec.CommandContext(ctx, "sh", args...)
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
	return c.Run()
}

// expandConfigFiles returns the files to open for the referenced config paths:
// files are kept as is, directories are replaced by the text files they
// contain (binary files such as images are skipped). Order is preserved and
// duplicates are dropped.
func expandConfigFiles(paths []string) ([]string, error) {
	var (
		files []string
		seen  = map[string]bool{}
	)
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			add(p)
			continue
		}
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.Type().IsRegular() {
				return nil
			}
			text, err := isTextFile(path)
			if err != nil {
				return err
			}
			if text {
				add(path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("expanding %s: %w", p, err)
		}
	}
	return files, nil
}

// isTextFile reports whether the file at path looks like text, using git's
// heuristic: a NUL byte within the first 8000 bytes marks it as binary.
func isTextFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, 8000)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, err
	}
	return !bytes.Contains(buf[:n], []byte{0}), nil
}
