package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
)

// renderPackageNotFound prints a *dis.PackageNotFoundError together with its
// "Did you mean this?" suggestions, or a *dis.AmbiguousPackageError together
// with the packages it matched, silencing cobra's own error and usage output
// so the names are not buried. Other errors are returned untouched. The error
// is always returned so the exit code stays non-zero.
func renderPackageNotFound(cmd *cobra.Command, err error) error {
	var (
		// lookupErr is the unwrapped lookup error, printed without the
		// context callers may have wrapped it in.
		lookupErr error
		header    string
		names     []string
		nf        *dis.PackageNotFoundError
		amb       *dis.AmbiguousPackageError
	)
	switch {
	case errors.As(err, &nf):
		lookupErr, header, names = nf, "Did you mean this?", nf.Suggestions
	case errors.As(err, &amb):
		lookupErr, header, names = amb, "Use one of:", amb.Matches
	default:
		return err
	}
	cmd.SilenceUsage, cmd.SilenceErrors = true, true

	w := cmd.ErrOrStderr()
	fmt.Fprintf(w, "Error: %v\n", lookupErr)
	if len(names) > 0 {
		fmt.Fprintf(w, "\n%s\n", header)
		for _, s := range names {
			fmt.Fprintf(w, "\t%s\n", s)
		}
	}
	return err
}

// renderMissingDeps prints the error for a package whose dependencies are not
// recorded as installed, with the ways forward, silencing cobra's own error and
// usage output. The suggested commands repeat --distro when it was given on the
// command line. The error is returned so the exit code stays non-zero.
func renderMissingDeps(cmd *cobra.Command, pkgName string, missing []string, reinstall bool) error {
	err := fmt.Errorf("%s depends on packages that are not installed: %s", pkgName, strings.Join(missing, ", "))
	cmd.SilenceUsage, cmd.SilenceErrors = true, true

	prefix := "dis install"
	if f := cmd.Flags().Lookup("distro"); f != nil && f.Changed {
		prefix += " --distro " + f.Value.String()
	}
	w := cmd.ErrOrStderr()
	fmt.Fprintf(w, "Error: %v\n", err)
	fmt.Fprintf(w, "  Run '%s --with-deps %s' to install them first,\n", prefix, pkgName)
	if reinstall {
		// --with-deps skips pkgName, which is installed: reinstall it after.
		fmt.Fprintf(w, "  then '%s --reinstall %s' again,\n", prefix, pkgName)
	}
	fmt.Fprintf(w, "  or pass --no-deps-check if they were installed outside dis.\n")
	return err
}
