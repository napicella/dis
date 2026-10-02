package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/napicella/dis/internal/dis"
	"github.com/napicella/dis/internal/doctor"
	"github.com/napicella/dis/internal/rcstate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the shell rc files, install state and source repos for drift",
	Long: `Checks for drift that dis doesn't catch on its own:

  ~/.bashrc         lines outside dis sections, e.g. a tool's own PATH line;
                    a ~/.bashrc that doesn't load ~/rc/bash_config.sh
  generated files   files in ~/rc/configs-generated edited outside dis, missing,
                    or behind the state they are rendered from
  unguarded PATH    'export PATH=X:$PATH' in a section, repeated by every
                    nested shell
  orphan sections   sections owned by a package that is neither in the distro
                    nor installed
  PATH dirs         dirs on PATH that don't exist, with the section or rc line
                    that adds each one
  source repos      uncommitted changes, unpushed commits, and commits fetched
                    but not merged

Orphan sections and source repos need the distro file and are skipped without
one. A section without an owner is never an orphan. PATH is read from the shell
dis runs in, which inherits it from whatever started it (terminal, multiplexer
server, desktop session): after a fix, old entries stay until those are
restarted. Repos are not fetched: 'dis pull' does that.

--fix first makes the mechanical fixes:
  - removes orphan sections, after backing up their files
  - regenerates generated files that are missing or behind the state
It never overwrites a generated file edited outside dis. The other problems
are only reported.

Exits non-zero when a check finds a problem.

Examples:
  dis doctor
  dis doctor --fix`,
	Args:    cobra.NoArgs,
	PreRunE: bindSharedConfigFlags,
	RunE:    doctorCmdFn,
}

var doctorFix bool

func init() {
	doctorCmd.Flags().String("distro", "", "Path to the distro YAML file")
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "remove orphan sections and regenerate missing or outdated generated files, then check")
	rootCmd.AddCommand(doctorCmd)
}

// errDoctorProblems makes 'dis doctor' exit non-zero after it printed the
// problems itself.
var errDoctorProblems = errors.New("doctor found problems")

func doctorCmdFn(cmd *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}
	store, err := rcstate.DefaultStore()
	if err != nil {
		return err
	}
	d := doctor.New(home, store)
	out := cmd.OutOrStdout()

	// The distro is optional: without it the checks that need it are skipped.
	var ic *dis.InstallContext
	distroErr := "needs the distro file: pass --distro or run 'dis pull'"
	installed, installedErr := dis.ListInstalled()
	if distroFile := viper.GetString("distro"); distroFile != "" {
		if ic, err = dis.NewInstallContext(distroFile); err != nil {
			distroErr = fmt.Sprintf("loading the distro: %v", err)
		} else if plan, err := ic.ResolveInstallOrder(); err != nil {
			distroErr = fmt.Sprintf("resolving the install plan: %v", err)
			ic = nil
		} else if installedErr != nil {
			distroErr = fmt.Sprintf("reading install state: %v", installedErr)
			ic = nil
		} else {
			d.Plan, d.Installed = map[string]bool{}, map[string]bool{}
			for _, m := range plan {
				d.Plan[m.Provides] = true
			}
			for _, p := range installed {
				d.Installed[p] = true
			}
		}
	}

	if doctorFix {
		done, err := d.Fix()
		if err != nil {
			return err
		}
		for _, s := range done {
			fmt.Fprintf(out, "fixed  %s\n", s)
		}
		if len(done) > 0 {
			fmt.Fprintln(out)
		}
	}

	results := []doctor.Result{
		d.CheckBashrc(),
		d.CheckGenerated(),
		d.CheckUnguardedPath(),
		d.CheckOrphans(),
		d.CheckDeadPath(os.Getenv("PATH")),
	}
	repos := doctor.Result{OK: "source repos are in sync", Skipped: distroErr}
	if ic != nil {
		repos = d.CheckRepos(ic.Repos)
	}
	results = append(results, repos)

	if renderDoctorResults(out, results) {
		cmd.SilenceUsage, cmd.SilenceErrors = true, true
		return errDoctorProblems
	}
	return nil
}

// renderDoctorResults prints one line per check, followed by its problems and
// fixes, and reports whether any check found a problem.
func renderDoctorResults(w io.Writer, results []doctor.Result) bool {
	failed := 0
	for _, r := range results {
		switch {
		case r.Skipped != "":
			fmt.Fprintf(w, "skip   %s: %s\n", r.OK, r.Skipped)
		case len(r.Problems) == 0:
			fmt.Fprintf(w, "ok     %s\n", r.OK)
		default:
			failed++
			fmt.Fprintf(w, "warn   %s\n", r.Warn)
			for _, p := range r.Problems {
				fmt.Fprintf(w, "         %s\n", p.Text)
				for _, d := range p.Detail {
					fmt.Fprintf(w, "           %s\n", d)
				}
			}
			if r.Hint != "" {
				fmt.Fprintf(w, "       fix: %s\n", r.Hint)
			}
		}
	}
	if failed > 0 {
		fmt.Fprintf(w, "\n%d of %d checks found problems\n", failed, len(results))
	}
	return failed > 0
}
