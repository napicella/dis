// Package doctor implements the checks and fixes of 'dis doctor': drift in the
// shell rc files, the install state and the source repos.
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/napicella/dis/internal/rcstate"
)

// maxShown caps the problems listed for one check, e.g. a stock ~/.bashrc
// has about a hundred lines outside dis sections.
const maxShown = 10

// Result is the outcome of one check.
type Result struct {
	// OK describes the passing state, e.g. "all PATH dirs exist".
	OK string
	// Warn describes the problems found, with their count, e.g.
	// "2 PATH dirs don't exist"; set by warn.
	Warn string
	// Problems lists what the check found; empty when the check passed.
	Problems []Problem
	// Hint says how to fix all the problems, for checks whose problems share
	// one fix; empty when each problem carries its own.
	Hint string
	// Skipped explains why the check did not run; empty when it ran.
	Skipped string
}

// Problem is one thing a check found.
type Problem struct {
	Text string
	// Detail adds lines under Text, e.g. where it comes from and its fix.
	Detail []string
}

// warn sets r.Warn from the problem count, using singular or plural, and caps
// the problems listed at maxShown.
func (r *Result) warn(singular, plural string) {
	switch len(r.Problems) {
	case 0:
		return
	case 1:
		r.Warn = singular
	default:
		r.Warn = fmt.Sprintf(plural, len(r.Problems))
	}
	if len(r.Problems) > maxShown {
		more := len(r.Problems) - maxShown
		r.Problems = append(r.Problems[:maxShown], Problem{Text: fmt.Sprintf("... and %d more", more)})
	}
}

// Doctor runs the checks for one home directory.
type Doctor struct {
	Home  string
	Store *rcstate.Store
	// Plan holds the packages the distro installs, dependencies included, and
	// Installed those recorded as installed. Plan is nil without a distro
	// file, and the checks that need it are skipped.
	Plan, Installed map[string]bool

	state    *rcstate.State
	imported bool // no state file yet: state was imported from the files
	stateErr error
}

// New returns a Doctor for home, with the state loaded from store.
func New(home string, store *rcstate.Store) *Doctor {
	d := &Doctor{Home: home, Store: store}
	d.reload()
	return d
}

func (d *Doctor) reload() {
	d.state, d.imported, d.stateErr = d.Store.Load()
}

func (d *Doctor) bashrc() string { return filepath.Join(d.Home, ".bashrc") }

// homeRCFiles are the rc files in $HOME that can add to PATH.
func (d *Doctor) homeRCFiles() []string {
	return []string{d.bashrc(), filepath.Join(d.Home, ".bash_profile"), filepath.Join(d.Home, ".profile")}
}

// short replaces the home directory prefix of p with ~.
func (d *Doctor) short(p string) string {
	if rel, err := filepath.Rel(d.Home, p); err == nil && rel != ".." && !strings.HasPrefix(rel, "../") {
		return filepath.Join("~", rel)
	}
	return p
}

// tildePath matches a ~/ that starts a path: at the start of a word, an
// assignment or a PATH element.
var tildePath = regexp.MustCompile(`(^|[\s=:"'])~/`)

// expandHome replaces $HOME, ${HOME} and ~/ in s with the home dir, as the
// shell would when sourcing the line.
func (d *Doctor) expandHome(s string) string {
	s = strings.ReplaceAll(s, "${HOME}", d.Home)
	s = strings.ReplaceAll(s, "$HOME", d.Home)
	return tildePath.ReplaceAllString(s, "${1}"+d.Home+"/")
}

// isOrphan reports whether sec is owned by a package that is neither in the
// distro nor installed. A section without an owner is never an orphan.
func (d *Doctor) isOrphan(sec rcstate.Section) bool {
	return d.Plan != nil && sec.Owner != "" && !d.Plan[sec.Owner] && !d.Installed[sec.Owner]
}

func readLines(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), nil
}
