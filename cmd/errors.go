package cmd

import (
	"errors"
	"fmt"

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
