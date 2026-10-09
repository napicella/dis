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
			err:  &dis.PackageNotFoundError{Name: "neovim", Suggestions: []string{"tools/neovim", "tools/neovide"}},
			wantOutput: "Error: package \"neovim\" not found in any of the configured sources\n" +
				"\nDid you mean this?\n\ttools/neovim\n\ttools/neovide\n",
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

func TestRenderMissingDeps(t *testing.T) {
	tests := []struct {
		name      string
		distro    string
		reinstall bool
		want      string
	}{
		{
			name: "install",
			want: "Error: tools/app depends on packages that are not installed: common/base, common/mid\n" +
				"  Run 'dis install --with-deps tools/app' to install them first,\n" +
				"  or pass --no-deps-check if they were installed outside dis.\n",
		},
		{
			name:      "reinstall",
			reinstall: true,
			want: "Error: tools/app depends on packages that are not installed: common/base, common/mid\n" +
				"  Run 'dis install --with-deps tools/app' to install them first,\n" +
				"  then 'dis install --reinstall tools/app' again,\n" +
				"  or pass --no-deps-check if they were installed outside dis.\n",
		},
		{
			name:   "--distro given on the command line",
			distro: "/x/distro.yml",
			want: "Error: tools/app depends on packages that are not installed: common/base, common/mid\n" +
				"  Run 'dis install --distro /x/distro.yml --with-deps tools/app' to install them first,\n" +
				"  or pass --no-deps-check if they were installed outside dis.\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("distro", "", "")
			if tt.distro != "" {
				if err := cmd.Flags().Set("distro", tt.distro); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			cmd.SetErr(&out)

			err := renderMissingDeps(cmd, "tools/app", []string{"common/base", "common/mid"}, tt.reinstall)
			if err == nil {
				t.Fatal("renderMissingDeps returned nil, want an error")
			}
			if out.String() != tt.want {
				t.Errorf("output =\n%s\nwant\n%s", out.String(), tt.want)
			}
			if !cmd.SilenceUsage || !cmd.SilenceErrors {
				t.Error("cobra usage and error output not silenced")
			}
		})
	}
}
