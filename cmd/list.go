package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages, or the sources of the distro",
	Long: `Prints the packages that have been installed by dis, one per line.

With --sources, prints the source directories the distro file declares
instead, in declaration order, with ${...} variables resolved. These are the
directories dis loads installers from, so they are where a new package can go.
Each row shows the source as declared, its resolved path, the repo it belongs
to and how many packages dis loaded from it. --distro applies to --sources only.

--json prints the sources as a JSON array of {declared, path, repo, exists,
packages} objects; repo is empty for a source outside every repo, and exists is
false for a source whose directory is missing (e.g. a repo not cloned yet).

Examples:
  dis list
  dis list --sources
  dis list --sources --json | jq -r '.[].path'`,
	PreRunE: bindSharedConfigFlags,
	RunE:    listCmdFn,
}

var (
	listSources bool
	listJSON    bool
)

func init() {
	listCmd.Flags().String("distro", "", "Path to the distro YAML file (with --sources)")
	listCmd.Flags().BoolVar(&listSources, "sources", false,
		"print the distro's source directories instead of the installed packages")
	listCmd.Flags().BoolVar(&listJSON, "json", false,
		"print the sources as a JSON array (with --sources)")
	rootCmd.AddCommand(listCmd)
}

func listCmdFn(cmd *cobra.Command, _ []string) error {
	if listSources {
		return listSourcesFn(cmd)
	}
	if listJSON {
		return fmt.Errorf("--json requires --sources")
	}

	pkgs, err := dis.ListInstalled()
	if err != nil {
		return fmt.Errorf("reading install state: %w", err)
	}

	for _, pkg := range pkgs {
		fmt.Println(pkg)
	}
	return nil
}

// sourceRow is one row of 'dis list --sources' output.
type sourceRow struct {
	Declared string `json:"declared"`
	Path     string `json:"path"`
	// Repo is the name of the distro repo the source lives in; empty when it
	// is in none of them.
	Repo   string `json:"repo"`
	Exists bool   `json:"exists"`
	// Packages is how many packages dis loaded from this source.
	Packages int `json:"packages"`
}

func listSourcesFn(cmd *cobra.Command) error {
	distroFile := viper.GetString("distro")
	if distroFile == "" {
		return fmt.Errorf("required flag \"distro\" not set and not found in config file")
	}
	ic, err := dis.NewInstallContext(distroFile)
	if err != nil {
		return err
	}
	rows := sourceRows(ic.Sources, ic.Repos, ic.ListAvailablePackages())

	out := cmd.OutOrStdout()
	if listJSON {
		return renderSourcesJSON(out, rows)
	}
	return renderSourceRows(out, rows)
}

// sourceRows builds the output rows for sources, attributing each to the repo
// whose clone contains it (the deepest one, should clones be nested) and
// counting the packages whose installer lies under it.
func sourceRows(sources []dis.ResolvedSource, repos []dis.ResolvedRepo, pkgs []dis.PackageInfo) []sourceRow {
	rows := make([]sourceRow, 0, len(sources))
	for _, s := range sources {
		row := sourceRow{Declared: s.Declared, Path: s.Path}

		best := -1
		for _, r := range repos {
			if within(s.Path, r.Path) && len(r.Path) > best {
				row.Repo, best = r.Name, len(r.Path)
			}
		}
		if info, err := os.Stat(s.Path); err == nil && info.IsDir() {
			row.Exists = true
		}
		for _, p := range pkgs {
			if within(p.InstallerPath, s.Path) {
				row.Packages++
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// within reports whether path is dir or lies under it.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// renderSourceRows prints aligned "declared  path  repo  N packages" rows.
func renderSourceRows(w io.Writer, rows []sourceRow) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, r := range rows {
		repo := r.Repo
		if repo == "" {
			repo = "-"
		}
		count := fmt.Sprintf("%d packages", r.Packages)
		if r.Packages == 1 {
			count = "1 package"
		}
		if !r.Exists {
			count = "missing"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Declared, r.Path, repo, count)
	}
	return tw.Flush()
}

// renderSourcesJSON prints rows as an indented JSON array; no sources prints
// "[]" so stdout is always valid JSON.
func renderSourcesJSON(w io.Writer, rows []sourceRow) error {
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}
