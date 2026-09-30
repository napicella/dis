package cmd

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"text/tabwriter"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search packages available from the sources defined in the distro file",
	Long: `Search the packages available from the sources defined in the distro file.

Patterns are golang regular expressions (https://pkg.go.dev/regexp); they are
not anchored, so "git" matches any package name containing "git".

  --package  filters packages by name. Without other flags, prints the name and
             installer path of each match. Without any flag, lists every package.
  --content  prints the installer lines matching the pattern (the manifest
             header is skipped), restricted to the packages matched by --package.
  --configs  prints the absolute paths of the config files the matched packages
             reference through $DIS_CONFIG_FOLDER, one per line.

Exits non-zero when nothing matches.

Examples:
  dis search --package git
  dis search --content status
  dis search --package git --content status
  dis search --content 'status.*git'
  vim $(dis search --package starship --configs)`,
	PreRunE: bindSharedConfigFlags,
	RunE:    searchCmdFn,
}

var (
	searchPackage string
	searchContent string
	searchConfigs bool
)

func init() {
	searchCmd.Flags().String("distro", "", "Path to the distro YAML file")
	searchCmd.Flags().StringVarP(&searchPackage, "package", "p", "",
		"regular expression matched against package names")
	searchCmd.Flags().StringVarP(&searchContent, "content", "c", "",
		"regular expression matched against installer lines")
	searchCmd.Flags().BoolVar(&searchConfigs, "configs", false,
		"print the config files referenced by the matched packages")
	searchCmd.MarkFlagsMutuallyExclusive("content", "configs")
	rootCmd.AddCommand(searchCmd)
}

// errNoMatches is returned when a search finds nothing, so the exit code is
// non-zero (e.g. "vim $(dis search ... --configs)" does not open an empty buffer).
var errNoMatches = errors.New("no matches")

func searchCmdFn(cmd *cobra.Command, _ []string) error {
	distroFile := viper.GetString("distro")
	if distroFile == "" {
		return fmt.Errorf("required flag \"distro\" not set and not found in config file")
	}

	pkgRe, err := regexp.Compile(searchPackage)
	if err != nil {
		return fmt.Errorf("invalid --package regex: %w", err)
	}
	var contentRe *regexp.Regexp
	if searchContent != "" {
		if contentRe, err = regexp.Compile(searchContent); err != nil {
			return fmt.Errorf("invalid --content regex: %w", err)
		}
	}

	ic, err := dis.NewInstallContextWithCache(distroFile)
	if err != nil {
		return err
	}
	var pkgs []dis.PackageInfo
	for _, p := range ic.ListAvailablePackages() {
		if pkgRe.MatchString(p.Provides) {
			pkgs = append(pkgs, p)
		}
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].Provides < pkgs[j].Provides
	})

	// From here on, failures are search outcomes rather than usage mistakes.
	cmd.SilenceUsage = true
	out := cmd.OutOrStdout()
	switch {
	case contentRe != nil:
		return printContentMatches(out, pkgs, contentRe)
	case searchConfigs:
		return printConfigs(out, pkgs)
	default:
		return printPackages(out, pkgs)
	}
}

// printPackages prints the name and installer path of each package.
func printPackages(w io.Writer, pkgs []dis.PackageInfo) error {
	if len(pkgs) == 0 {
		return errNoMatches
	}
	for i, p := range pkgs {
		if i > 0 {
			fmt.Fprintln(w, "---")
		}
		fmt.Fprintf(w, "Name: %s\nPath: %s\n", p.Provides, p.InstallerPath)
	}
	return nil
}

// printContentMatches prints one aligned "package  path:line  text" row per
// installer line matching re.
func printContentMatches(w io.Writer, pkgs []dis.PackageInfo, re *regexp.Regexp) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	found := false
	for _, p := range pkgs {
		matches, err := dis.SearchContent(p.InstallerPath, re)
		if err != nil {
			return fmt.Errorf("searching %s: %w", p.InstallerPath, err)
		}
		for _, m := range matches {
			found = true
			fmt.Fprintf(tw, "%s\t%s:%d\t%s\n", p.Provides, p.InstallerPath, m.Line, m.Text)
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if !found {
		return errNoMatches
	}
	return nil
}

// printConfigs prints the config paths referenced by pkgs, one per line and
// without duplicates, so the output can be passed straight to an editor.
func printConfigs(w io.Writer, pkgs []dis.PackageInfo) error {
	seen := map[string]bool{}
	for _, p := range pkgs {
		paths, err := dis.ReferencedConfigs(p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p.InstallerPath, err)
		}
		for _, path := range paths {
			if !seen[path] {
				seen[path] = true
				fmt.Fprintln(w, path)
			}
		}
	}
	if len(seen) == 0 {
		return errNoMatches
	}
	return nil
}
