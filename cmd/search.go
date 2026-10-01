package cmd

import (
	"encoding/json"
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
	Short: "Search the packages available from the sources defined in the distro file",
	Long: `Search the packages available from the sources defined in the distro file,
installed or not. Say what to search with a subcommand:

  packages REGEX     package names
  installers REGEX   installer lines (the manifest header is skipped)
  configs REGEX      paths of the config files installers reference through
                     $DIS_CONFIG_FOLDER

Patterns are golang regular expressions (https://pkg.go.dev/regexp); they are
not anchored, so "git" matches anything containing "git". installers and configs
take --package REGEX to only look at the packages whose name matches.

Every subcommand prints the same shape: one "package  path" row per result, where
path is the installer (or the config file for configs). Installer matches append
":line" to the path and the matching text as a third column. --json prints the
same results as a JSON array of {package, path, line, text} objects; line and
text are set only for installer matches.

Exits non-zero when nothing matches.

Examples:
  dis search packages git
  dis search installers status
  dis search installers status --package git
  dis search installers 'status.*git'
  dis search configs . --package starship
  vim $(dis search configs . --package starship --json | jq -r '.[].path')`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return fmt.Errorf("say what to search: dis search packages|installers|configs REGEX")
	},
}

var searchPackagesCmd = &cobra.Command{
	Use:     "packages REGEX",
	Short:   "Search package names",
	Example: "  dis search packages git\n  dis search packages '^common/'",
	Args:    cobra.ExactArgs(1),
	PreRunE: bindSharedConfigFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearch(cmd, args[0], "", func(pkgs []dis.PackageInfo, re *regexp.Regexp) ([]searchResult, error) {
			var matched []dis.PackageInfo
			for _, p := range pkgs {
				if re.MatchString(p.Provides) {
					matched = append(matched, p)
				}
			}
			return collectPackages(matched), nil
		})
	},
}

var searchInstallersCmd = &cobra.Command{
	Use:     "installers REGEX",
	Short:   "Search installer lines",
	Example: "  dis search installers 'alias status='\n  dis search installers status --package git",
	Args:    cobra.ExactArgs(1),
	PreRunE: bindSharedConfigFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgFilter, _ := cmd.Flags().GetString("package")
		return runSearch(cmd, args[0], pkgFilter, collectContent)
	},
}

var searchConfigsCmd = &cobra.Command{
	Use:     "configs REGEX",
	Short:   "Search the paths of the config files installers reference",
	Example: "  dis search configs tmux\n  dis search configs . --package starship",
	Args:    cobra.ExactArgs(1),
	PreRunE: bindSharedConfigFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgFilter, _ := cmd.Flags().GetString("package")
		return runSearch(cmd, args[0], pkgFilter, func(pkgs []dis.PackageInfo, re *regexp.Regexp) ([]searchResult, error) {
			all, err := collectConfigs(pkgs)
			if err != nil {
				return nil, err
			}
			results := []searchResult{}
			for _, r := range all {
				if re.MatchString(r.Path) {
					results = append(results, r)
				}
			}
			return results, nil
		})
	},
}

func init() {
	for _, c := range []*cobra.Command{searchPackagesCmd, searchInstallersCmd, searchConfigsCmd} {
		c.Flags().String("distro", "", "Path to the distro YAML file")
		c.Flags().Bool("json", false, "print the results as a JSON array")
		searchCmd.AddCommand(c)
	}
	for _, c := range []*cobra.Command{searchInstallersCmd, searchConfigsCmd} {
		c.Flags().StringP("package", "p", "", "only search the packages whose name matches this regular expression")
	}
	rootCmd.AddCommand(searchCmd)
}

// errNoMatches is returned when a search finds nothing, so the exit code is
// non-zero and scripts can tell an empty search apart from a successful one.
var errNoMatches = errors.New("no matches")

// searchResult is one row of search output. Every subcommand produces the same
// type so that consumers see one shape whatever they searched.
type searchResult struct {
	Package string `json:"package"`
	// Path is the installer path, or the config path for configs.
	Path string `json:"path"`
	// Line and Text are the matching installer line; set only for installers.
	Line int    `json:"line,omitempty"`
	Text string `json:"text,omitempty"`
}

// runSearch loads the distro's available packages, keeps those whose name
// matches pkgFilter (all when empty), and prints what collect finds in them for
// pattern.
func runSearch(cmd *cobra.Command, pattern, pkgFilter string,
	collect func([]dis.PackageInfo, *regexp.Regexp) ([]searchResult, error)) error {
	distroFile := viper.GetString("distro")
	if distroFile == "" {
		return fmt.Errorf("required flag \"distro\" not set and not found in config file")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}
	pkgRe, err := regexp.Compile(pkgFilter)
	if err != nil {
		return fmt.Errorf("invalid --package regex: %w", err)
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

	results, err := collect(pkgs, re)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		err = renderJSON(out, results)
	} else {
		err = renderRows(out, results)
	}
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return errNoMatches
	}
	return nil
}

// collectPackages returns one result per package, pointing at its installer.
func collectPackages(pkgs []dis.PackageInfo) []searchResult {
	results := make([]searchResult, 0, len(pkgs))
	for _, p := range pkgs {
		results = append(results, searchResult{Package: p.Provides, Path: p.InstallerPath})
	}
	return results
}

// collectContent returns one result per installer line matching re.
func collectContent(pkgs []dis.PackageInfo, re *regexp.Regexp) ([]searchResult, error) {
	results := []searchResult{}
	for _, p := range pkgs {
		matches, err := dis.SearchContent(p.InstallerPath, re)
		if err != nil {
			return nil, fmt.Errorf("searching %s: %w", p.InstallerPath, err)
		}
		for _, m := range matches {
			results = append(results, searchResult{
				Package: p.Provides,
				Path:    p.InstallerPath,
				Line:    m.Line,
				Text:    m.Text,
			})
		}
	}
	return results, nil
}

// collectConfigs returns one result per config path referenced by each
// package. ReferencedConfigs already deduplicates within a package, so a file
// shared by two packages is reported once for each.
func collectConfigs(pkgs []dis.PackageInfo) ([]searchResult, error) {
	results := []searchResult{}
	for _, p := range pkgs {
		paths, err := dis.ReferencedConfigs(p)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", p.InstallerPath, err)
		}
		for _, path := range paths {
			results = append(results, searchResult{Package: p.Provides, Path: path})
		}
	}
	return results, nil
}

// renderRows prints aligned "package  path" rows; content matches append
// ":line" to the path and the matching text as a third column.
func renderRows(w io.Writer, results []searchResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, r := range results {
		if r.Line > 0 {
			fmt.Fprintf(tw, "%s\t%s:%d\t%s\n", r.Package, r.Path, r.Line, r.Text)
		} else {
			fmt.Fprintf(tw, "%s\t%s\n", r.Package, r.Path)
		}
	}
	return tw.Flush()
}

// renderJSON prints results as an indented JSON array; an empty search prints
// "[]" so stdout is always valid JSON.
func renderJSON(w io.Writer, results []searchResult) error {
	if results == nil {
		results = []searchResult{}
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}
