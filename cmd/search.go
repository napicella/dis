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
	Short: "Search packages available from the sources defined in the distro file",
	Long: `Search the packages available from the sources defined in the distro file.

Patterns are golang regular expressions (https://pkg.go.dev/regexp); they are
not anchored, so "git" matches any package name containing "git".

  --package  filters packages by name. Without any flag, every package matches.
  --content  searches the installer lines of the matched packages (the manifest
             header is skipped).
  --configs  lists the config files the matched packages reference through
             $DIS_CONFIG_FOLDER.

Every mode prints the same shape: one "package  path" row per result, where path
is the installer (or the config file with --configs). Content matches append
":line" to the path and the matching text as a third column. --json prints the
same results as a JSON array of {package, path, line, text} objects; line and
text are set only with --content.

Exits non-zero when nothing matches.

Examples:
  dis search --package git
  dis search --content status
  dis search --package git --content status
  dis search --content 'status.*git'
  vim $(dis search --package starship --configs --json | jq -r '.[].path')`,
	PreRunE: bindSharedConfigFlags,
	RunE:    searchCmdFn,
}

var (
	searchPackage string
	searchContent string
	searchConfigs bool
	searchJSON    bool
)

func init() {
	searchCmd.Flags().String("distro", "", "Path to the distro YAML file")
	searchCmd.Flags().StringVarP(&searchPackage, "package", "p", "",
		"regular expression matched against package names")
	searchCmd.Flags().StringVarP(&searchContent, "content", "c", "",
		"regular expression matched against installer lines")
	searchCmd.Flags().BoolVar(&searchConfigs, "configs", false,
		"print the config files referenced by the matched packages")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false,
		"print the results as a JSON array")
	searchCmd.MarkFlagsMutuallyExclusive("content", "configs")
	rootCmd.AddCommand(searchCmd)
}

// errNoMatches is returned when a search finds nothing, so the exit code is
// non-zero and scripts can tell an empty search apart from a successful one.
var errNoMatches = errors.New("no matches")

// searchResult is one row of search output. Every mode produces the same type
// so that consumers see one shape regardless of the flags.
type searchResult struct {
	Package string `json:"package"`
	// Path is the installer path, or the config path with --configs.
	Path string `json:"path"`
	// Line and Text are the matching installer line; set only with --content.
	Line int    `json:"line,omitempty"`
	Text string `json:"text,omitempty"`
}

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

	var results []searchResult
	switch {
	case contentRe != nil:
		results, err = collectContent(pkgs, contentRe)
	case searchConfigs:
		results, err = collectConfigs(pkgs)
	default:
		results = collectPackages(pkgs)
	}
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if searchJSON {
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
