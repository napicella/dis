package cmd

import (
	"errors"
	"fmt"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
)

// renderPackageNotFound prints a *dis.PackageNotFoundError together with its
// "Did you mean this?" suggestions, silencing cobra's own error and usage
// output so the suggestions are not buried. Other errors are returned
// untouched. The error is always returned so the exit code stays non-zero.
func renderPackageNotFound(cmd *cobra.Command, err error) error {
	var nf *dis.PackageNotFoundError
	if !errors.As(err, &nf) {
		return err
	}
	cmd.SilenceUsage, cmd.SilenceErrors = true, true

	w := cmd.ErrOrStderr()
	fmt.Fprintf(w, "Error: %v\n", nf)
	if len(nf.Suggestions) > 0 {
		fmt.Fprintln(w, "\nDid you mean this?")
		for _, s := range nf.Suggestions {
			fmt.Fprintf(w, "\t%s\n", s)
		}
	}
	return err
}
