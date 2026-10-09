package doctor

import (
	"fmt"
	"os"

	"github.com/napicella/dis/internal/rcstate"
)

// Fix makes the mechanical fixes and returns what it did:
//   - removes orphan sections, after backing up their files;
//   - regenerates generated files that are missing or behind the state.
//
// It never overwrites a file edited outside dis: orphans in such a file stay
// until the edit is dealt with.
func (d *Doctor) Fix() ([]string, error) {
	if d.stateErr != nil {
		return nil, nil // CheckGenerated reports it; the next dis write rebuilds it
	}

	// Without a state file there is nothing to compare the files with: the
	// write below imports them first.
	edited := map[string]bool{}
	var regenerate []string
	if !d.imported {
		for _, file := range rcstate.Files {
			status, err := d.Store.Status(d.state, file)
			if err != nil {
				return nil, err
			}
			switch status {
			case rcstate.Edited:
				edited[file] = true
			case rcstate.Missing, rcstate.Behind:
				regenerate = append(regenerate, file)
			}
		}
	}

	var done []string
	type orphan struct {
		file string
		sec  rcstate.Section
	}
	var orphans []orphan
	for file, s := range d.state.All() {
		switch {
		case !d.isOrphan(s):
		case edited[file]:
			done = append(done, fmt.Sprintf("kept orphan section %s: %s was edited outside dis", sectionRef(file, s), d.short(d.Store.Path(file))))
		default:
			orphans = append(orphans, orphan{file, s})
		}
	}
	if len(orphans) == 0 && len(regenerate) == 0 {
		return done, nil
	}

	backedUp := map[string]bool{}
	for _, o := range orphans {
		path := d.Store.Path(o.file)
		if backedUp[path] {
			continue
		}
		backedUp[path] = true
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if _, err := rcstate.Backup(path, d.Store.Now()); err != nil {
			return nil, fmt.Errorf("backing up %s: %w", path, err)
		}
	}
	err := d.Store.UpdateKeepingEdits(func(st *rcstate.State) error {
		for _, o := range orphans {
			if err := st.Remove(o.file, o.sec.Name, o.sec.Owner); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, o := range orphans {
		done = append(done, fmt.Sprintf("removed orphan section %s (%s is neither in this distro nor installed)", sectionRef(o.file, o.sec), o.sec.Owner))
	}
	for _, file := range regenerate {
		done = append(done, "regenerated "+d.short(d.Store.Path(file)))
	}
	d.reload()
	return done, nil
}
