package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
)

func TestRenderPackageNotFound(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantOutput string
		wantSilent bool
	}{
		{
			name: "with suggestions",
			err:  &dis.PackageNotFoundError{Name: "herdr", Suggestions: []string{"tools/herdr", "tools/herd"}},
			wantOutput: "Error: package \"herdr\" not found in any of the configured sources\n" +
				"\nDid you mean this?\n\ttools/herdr\n\ttools/herd\n",
			wantSilent: true,
		},
		{
			name:       "without suggestions",
			err:        &dis.PackageNotFoundError{Name: "xyzzy"},
			wantOutput: "Error: package \"xyzzy\" not found in any of the configured sources\n",
			wantSilent: true,
		},
		{
			name:       "wrapped",
			err:        fmt.Errorf("outer: %w", &dis.PackageNotFoundError{Name: "xyzzy"}),
			wantOutput: "Error: package \"xyzzy\" not found in any of the configured sources\n",
			wantSilent: true,
		},
		{
			name:       "other error",
			err:        errors.New("boom"),
			wantOutput: "",
			wantSilent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			c := &cobra.Command{}
			c.SetErr(&out)

			if got := renderPackageNotFound(c, tt.err); got != tt.err {
				t.Errorf("returned error = %v, want %v", got, tt.err)
			}
			if out.String() != tt.wantOutput {
				t.Errorf("output = %q, want %q", out.String(), tt.wantOutput)
			}
			if c.SilenceUsage != tt.wantSilent || c.SilenceErrors != tt.wantSilent {
				t.Errorf("SilenceUsage=%v SilenceErrors=%v, want both %v", c.SilenceUsage, c.SilenceErrors, tt.wantSilent)
			}
		})
	}
}
