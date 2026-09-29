package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/mitchellh/go-homedir"
	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var pullCmd = &cobra.Command{
	Use:   "pull [git-url]",
	Short: "Clone or update the repos a distro needs, and set it as the default distro",
	Long: `With a git URL, clones the repo that contains the distro file (default path
~/<repo-name>), then clones or updates every repo listed under repos: in the
distro. Without arguments, does the same for the distro in the config file.

Repos that already exist are fast-forwarded only when their work tree is clean,
on a branch with an upstream, and not diverged; otherwise they are skipped, so
local work is never touched. The distro is then written as 'distro:' to
~/.config/dis/config.yaml, so later commands can omit --distro.

dis uses the git CLI, so your SSH config, keys and credential helpers apply.

Examples:
  dis pull git@github.com:napicella/dotfiles.git --distro distros/ubuntu-laptop/ubuntu-laptop.yml
  dis pull     # update the repos of the configured distro`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         pullCmdFn,
	SilenceUsage: true,
}

var (
	pullPath   string
	pullDistro string
)

func init() {
	pullCmd.Flags().StringVar(&pullPath, "path", "", "Where to clone the repo given as argument (default ~/<repo-name>)")
	pullCmd.Flags().StringVar(&pullDistro, "distro", "", "Distro file: relative to the cloned repo when a git URL is given, otherwise a path (default: the configured distro)")
	rootCmd.AddCommand(pullCmd)
}

type pullRow struct {
	name, path string
	res        dis.PullResult
	err        error
}

func pullCmdFn(cmd *cobra.Command, args []string) error {
	if err := dis.CheckGit(); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	var rows []pullRow
	var distroFile string

	if len(args) == 1 {
		url := args[0]
		path := pullPath
		if path == "" {
			path = "~/" + dis.RepoNameFromURL(url)
		}
		path, err := absPath(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "==> %s\n", path)
		res, err := dis.CloneOrUpdate(dis.ResolvedRepo{Name: dis.SelfRepo, URL: url, Path: path}, out)
		if err != nil {
			return err
		}
		rows = append(rows, pullRow{name: dis.SelfRepo, path: path, res: res})

		if distroFile, err = pickDistro(path, pullDistro); err != nil {
			return err
		}
	} else {
		distroFile = pullDistro
		if distroFile == "" {
			distroFile = viper.GetString("distro")
		}
		if distroFile == "" {
			return fmt.Errorf("no distro configured: run 'dis pull <git-url>' or pass --distro")
		}
		var err error
		if distroFile, err = absPath(distroFile); err != nil {
			return err
		}
	}

	cfg, err := dis.LoadDistro(distroFile)
	if err != nil {
		return err
	}
	repos, err := dis.ResolveRepos(cfg, filepath.Dir(distroFile))
	if err != nil {
		return fmt.Errorf("distro %q: %w", distroFile, err)
	}
	for _, r := range repos {
		if r.Name == dis.SelfRepo {
			if len(args) == 1 {
				continue // already cloned or updated above
			}
			if r.URL == "" {
				rows = append(rows, pullRow{name: r.Name, path: r.Path, res: dis.PullResult{Action: dis.PullSkipped, Detail: "not a git repo with an origin"}})
				continue
			}
		}
		fmt.Fprintf(out, "==> %s\n", r.Path)
		res, err := dis.CloneOrUpdate(r, out)
		rows = append(rows, pullRow{name: r.Name, path: r.Path, res: res, err: err})
	}

	old, err := setConfigDistro(distroFile)
	if err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	printPullSummary(out, rows)
	if old != "" && old != distroFile {
		fmt.Fprintf(out, "\nDefault distro changed: %s -> %s\n", old, distroFile)
	} else if old == "" {
		fmt.Fprintf(out, "\nDefault distro set to %s\n", distroFile)
	}

	for _, r := range rows {
		if r.err != nil {
			return errors.New("some repos could not be pulled, see above")
		}
	}
	fmt.Fprintln(out, "Next: dis install")
	return nil
}

func printPullSummary(out io.Writer, rows []pullRow) {
	fmt.Fprintln(out)
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPO\tPATH\tRESULT")
	for _, r := range rows {
		result := string(r.res.Action)
		if r.err != nil {
			result = "error: " + r.err.Error()
		} else if r.res.Detail != "" {
			result += " (" + r.res.Detail + ")"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.name, r.path, result)
	}
	w.Flush()
}

// pickDistro returns the absolute distro file inside repoDir. With an explicit
// path it is resolved against repoDir; otherwise the repo must contain exactly
// one distro file (a YAML file with os: and packages:).
func pickDistro(repoDir, distro string) (string, error) {
	if distro != "" {
		if !filepath.IsAbs(distro) {
			distro = filepath.Join(repoDir, distro)
		}
		if _, err := os.Stat(distro); err != nil {
			return "", fmt.Errorf("distro file %s: %w", distro, err)
		}
		return distro, nil
	}
	found, err := findDistros(repoDir)
	if err != nil {
		return "", err
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return "", fmt.Errorf("no distro file found in %s: pass --distro", repoDir)
	default:
		var b strings.Builder
		for _, f := range found {
			rel, _ := filepath.Rel(repoDir, f)
			b.WriteString("\n  " + rel)
		}
		return "", fmt.Errorf("several distro files in %s, pick one with --distro:%s", repoDir, b.String())
	}
}

func findDistros(repoDir string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(repoDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != repoDir && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if ext := filepath.Ext(p); ext != ".yml" && ext != ".yaml" {
			return nil
		}
		cfg, err := dis.LoadDistro(p)
		if err == nil && cfg.OS != "" && len(cfg.Packages) > 0 {
			found = append(found, p)
		}
		return nil
	})
	sort.Strings(found)
	return found, err
}

// setConfigDistro writes distro: <file> to ~/.config/dis/config.yaml, keeping
// the rest of the file (including comments), and returns the previous value.
func setConfigDistro(distroFile string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".config", "dis", "config.yaml")

	var doc yaml.Node
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return "", fmt.Errorf("parsing %s: %w", path, err)
		}
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("%s: expected a mapping at the top level", path)
	}

	old := ""
	set := false
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "distro" {
			old = root.Content[i+1].Value
			root.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: distroFile}
			set = true
			break
		}
	}
	if !set {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "distro"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: distroFile})
	}

	b, err := yaml.Marshal(&doc)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if old != "" {
		if expanded, err := absPath(old); err == nil {
			old = expanded
		}
	}
	return old, os.WriteFile(path, b, 0o644)
}

func absPath(p string) (string, error) {
	p, err := homedir.Expand(p)
	if err != nil {
		return "", err
	}
	return filepath.Abs(p)
}
