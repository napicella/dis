package doctor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/napicella/dis/internal/dis"
	"github.com/napicella/dis/internal/rcstate"
	"github.com/napicella/dis/internal/tools"
)

// maxDiffLines caps the diff shown for a file edited outside dis.
const maxDiffLines = 30

// CheckBashrc reports content in ~/.bashrc outside dis sections, and a
// ~/.bashrc that doesn't source ~/rc/bash_config.sh. It only reports: moving
// the lines would change when and in which shells they run.
func (d *Doctor) CheckBashrc() Result {
	r := Result{
		OK:   "~/.bashrc has only dis sections",
		Hint: "move each line into the installer of the tool that added it (e.g. with 'dis tools add-rc-path --path'), then delete it from ~/.bashrc",
	}
	lines, err := readLines(d.bashrc())
	if os.IsNotExist(err) {
		r.Skipped = "no ~/.bashrc"
		return r
	} else if err != nil {
		r.Skipped = err.Error()
		return r
	}
	sections, outside := tools.ParseRC(lines)
	for _, i := range outside {
		r.Problems = append(r.Problems, Problem{Text: fmt.Sprintf("~/.bashrc:%d  %s", i+1, strings.TrimSpace(lines[i]))})
	}
	r.warn("~/.bashrc has 1 line outside dis sections", "~/.bashrc has %d lines outside dis sections")

	loads := false
	for _, s := range sections {
		loads = loads || strings.Contains(s.Content, "rc/bash_config.sh")
	}
	if !loads {
		r.Problems = append([]Problem{{
			Text:   "~/.bashrc doesn't source ~/rc/bash_config.sh in a dis section",
			Detail: []string{"fix: run 'dis config': the package that wires ~/.bashrc ('dis tools add-home-rc') adds it"},
		}}, r.Problems...)
		r.Warn = "~/.bashrc doesn't load the dis config"
	}
	return r
}

// CheckGenerated reports generated files that differ from the state: edited
// outside dis, missing, or behind the state.
func (d *Doctor) CheckGenerated() Result {
	r := Result{OK: "the generated rc files match the state"}
	switch {
	case d.stateErr != nil:
		r.Problems = []Problem{{Text: d.stateErr.Error(), Detail: []string{"fix: the next dis write backs up a corrupt state file and rebuilds it from the generated files"}}}
		r.Warn = "the state file can't be read"
		return r
	case d.imported:
		r.Skipped = fmt.Sprintf("no %s yet: the next dis write creates it from the generated files", d.short(d.Store.StatePath))
		return r
	}
	for _, file := range rcstate.Files {
		path := d.short(d.Store.Path(file))
		status, err := d.Store.Status(d.state, file)
		switch {
		case err != nil:
			r.Problems = append(r.Problems, Problem{Text: err.Error()})
		case status == rcstate.Edited:
			p := Problem{Text: path + " was edited outside dis"}
			p.Detail = append(d.diff(file), "fix: move the change into an installer; the next dis write backs the file up and regenerates it")
			r.Problems = append(r.Problems, p)
		case status == rcstate.Missing:
			r.Problems = append(r.Problems, Problem{Text: path + " is missing", Detail: []string{"fix: 'dis doctor --fix' regenerates it"}})
		case status == rcstate.Behind:
			r.Problems = append(r.Problems, Problem{Text: path + " is behind the state", Detail: []string{"fix: 'dis doctor --fix' regenerates it"}})
		}
	}
	r.warn("1 generated rc file differs from the state", "%d generated rc files differ from the state")
	return r
}

// diff returns the changes made to file outside dis, as unified diff lines
// against what the state renders, capped at maxDiffLines. It returns nothing
// when the diff command isn't available.
func (d *Doctor) diff(file string) []string {
	tmp, err := os.CreateTemp("", "dis-render-*")
	if err != nil {
		return nil
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(rcstate.Render(d.state, file))
	tmp.Close()
	if err != nil {
		return nil
	}
	var out bytes.Buffer
	cmd := exec.Command("diff", "-u", "--label", "rendered from the state", "--label", d.short(d.Store.Path(file)), tmp.Name(), d.Store.Path(file))
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil && out.Len() == 0 {
		return nil
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) > maxDiffLines {
		lines = append(lines[:maxDiffLines], fmt.Sprintf("... and %d more lines", len(lines)-maxDiffLines))
	}
	return lines
}

// sectionRef names a section for a report: `"Cargo" in bash_paths, owned by
// tools/cargo`.
func sectionRef(file string, s rcstate.Section) string {
	ref := fmt.Sprintf("%q in %s", s.Name, file)
	if s.Owner != "" {
		ref += ", owned by " + s.Owner
	}
	return ref
}

// unguardedPath matches a PATH assignment that adds to the current PATH.
// Lines written by 'add-rc-path --path' start with "case" and don't match.
var unguardedPath = regexp.MustCompile(`^\s*(export\s+)?PATH=.*\$(\{PATH\}|PATH\b)`)

// CheckUnguardedPath reports PATH additions in the sections that aren't
// guarded, so nested shells repeat them.
func (d *Doctor) CheckUnguardedPath() Result {
	r := Result{
		OK:   "PATH additions are guarded",
		Hint: "in the installer that writes the section, use 'dis tools add-rc-path --name NAME --path DIR'",
	}
	if d.stateErr != nil {
		r.Skipped = "the state file can't be read"
		return r
	}
	for _, file := range rcstate.Files {
		for _, s := range d.state.Files[file] {
			for _, l := range strings.Split(s.Content, "\n") {
				if unguardedPath.MatchString(l) {
					r.Problems = append(r.Problems, Problem{Text: sectionRef(file, s), Detail: []string{strings.TrimSpace(l)}})
				}
			}
		}
	}
	r.warn("1 PATH addition is not guarded: nested shells repeat it", "%d PATH additions are not guarded: nested shells repeat them")
	return r
}

// CheckOrphans reports sections owned by a package that is neither in the
// distro nor installed, e.g. left behind by a package removed from the
// distro.
func (d *Doctor) CheckOrphans() Result {
	r := Result{
		OK:   "every rc section belongs to a package in the distro or installed",
		Hint: "'dis doctor --fix' removes them",
	}
	switch {
	case d.Plan == nil:
		r.Skipped = "needs the distro file: pass --distro or run 'dis pull'"
		return r
	case d.stateErr != nil:
		r.Skipped = "the state file can't be read"
		return r
	}
	for _, file := range rcstate.Files {
		for _, s := range d.state.Files[file] {
			if d.isOrphan(s) {
				r.Problems = append(r.Problems, Problem{Text: sectionRef(file, s), Detail: []string{s.Owner + " is neither in this distro nor installed"}})
			}
		}
	}
	r.warn("1 rc section belongs to a package that's gone", "%d rc sections belong to packages that are gone")
	return r
}

// CheckDeadPath reports PATH entries that aren't existing directories, with
// the section or rc line that adds each one and how to fix it.
func (d *Doctor) CheckDeadPath(pathEnv string) Result {
	r := Result{OK: "all PATH dirs exist"}
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			continue
		}
		r.Problems = append(r.Problems, Problem{Text: d.short(dir), Detail: d.pathOrigin(dir)})
	}
	r.warn("1 PATH dir doesn't exist", "%d PATH dirs don't exist")
	return r
}

// notFound is the origin of a PATH dir that no section or rc file adds.
var notFound = []string{
	"not found in the rc files: likely inherited from a process started before the fix",
	"fix: restart what launched this shell (terminal, tmux/herdr server, or the desktop session: log out and back in)",
}

// pathOrigin returns where dir is added to PATH and how to fix it.
func (d *Doctor) pathOrigin(dir string) []string {
	if d.stateErr == nil {
		for _, file := range rcstate.Files {
			for _, s := range d.state.Files[file] {
				if !addsDir(d.expandHome(s.Content), dir) {
					continue
				}
				where := "added by " + sectionRef(file, s)
				switch {
				case d.isOrphan(s):
					return []string{where, s.Owner + " is neither in this distro nor installed", "fix: 'dis doctor --fix' removes the section"}
				case s.Owner != "":
					return []string{where, fmt.Sprintf("fix: install %s again ('dis install --reinstall %s'), or drop the dir from its installer", s.Owner, s.Owner)}
				}
				return []string{where, fmt.Sprintf("fix: remove the section with 'dis tools rm-rc-section --file %s --name %q'", file, s.Name)}
			}
		}
	}
	for _, path := range d.homeRCFiles() {
		lines, err := readLines(path)
		if err != nil {
			continue
		}
		for i, l := range lines {
			if strings.Contains(l, "PATH") && addsDir(d.expandHome(l), dir) {
				return []string{fmt.Sprintf("added by %s:%d", d.short(path), i+1), "fix: edit " + d.short(path)}
			}
		}
	}
	return notFound
}

// addsDir reports whether text has dir as a PATH element: followed by a
// separator, a quote or the end of a line.
func addsDir(text, dir string) bool {
	for i := strings.Index(text, dir); i >= 0; {
		end := i + len(dir)
		if end == len(text) || strings.ContainsRune(":\"' ;\n", rune(text[end])) {
			return true
		}
		next := strings.Index(text[end:], dir)
		if next < 0 {
			break
		}
		i = end + next
	}
	return false
}

// CheckRepos reports source repos that are missing, have local changes, or
// differ from their upstream as of the last fetch.
func (d *Doctor) CheckRepos(repos []dis.ResolvedRepo) Result {
	r := Result{
		OK:   "source repos have no local changes and nothing left to pull",
		Hint: "commit and push local changes; 'dis pull' fetches and fast-forwards clean repos",
	}
	for _, repo := range repos {
		st, err := dis.GetRepoStatus(repo.Path)
		if err != nil {
			r.Problems = append(r.Problems, Problem{Text: fmt.Sprintf("%s: %v", repo.Name, err)})
			continue
		}
		var issues []string
		switch {
		case st.Missing:
			issues = append(issues, "not cloned")
		case st.Branch == "":
			issues = append(issues, "detached HEAD")
		case !st.HasUpstream:
			issues = append(issues, "no upstream branch")
		}
		if st.Dirty {
			issues = append(issues, "uncommitted changes")
		}
		if st.Ahead > 0 {
			issues = append(issues, fmt.Sprintf("%d commits not pushed", st.Ahead))
		}
		if st.Behind > 0 {
			issues = append(issues, fmt.Sprintf("%d commits behind", st.Behind))
		}
		if len(issues) > 0 {
			r.Problems = append(r.Problems, Problem{Text: fmt.Sprintf("%s (%s): %s", repo.Name, d.short(repo.Path), strings.Join(issues, ", "))})
		}
	}
	r.warn("1 source repo is out of sync", "%d source repos are out of sync")
	return r
}
