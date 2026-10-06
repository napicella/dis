package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// An unknown subcommand fails instead of printing the help and exiting 0, so
// an installer calling a typo, or a tool an older dis lacks, stops.
func TestUnknownSubcommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantOut string
		wantErr string
	}{
		{"tools alone prints help", []string{"tools"}, "Available Commands:", ""},
		{"tools unknown", []string{"tools", "nope"}, "", `unknown command "nope" for "dis tools"`},
		{"tools typo suggests", []string{"tools", "render-confg"}, "", "Did you mean this?\n\trender-config"},
		{"search typo suggests", []string{"search", "pakages"}, "", "Did you mean this?\n\tpackages"},
		{"root unknown", []string{"nope"}, "", `unknown command "nope" for "dis"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runCmd(t, tt.args...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, tt.wantOut) {
				t.Errorf("output does not contain %q:\n%s", tt.wantOut, out)
			}
		})
	}
}

// Every command with subcommands fails on an unknown one, not just tools.
// It runs through execute, as Execute does, so the completion command cobra
// adds gets the check too.
func TestParentCommandsRejectUnknownSubcommands(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	if err := execute(context.Background(), []string{"completion", "no-such-subcommand"}); err == nil {
		t.Errorf("dis completion no-such-subcommand: want an unknown command error")
	}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if !sub.HasSubCommands() {
				continue
			}
			t.Run(sub.CommandPath(), func(t *testing.T) {
				args := append(strings.Fields(sub.CommandPath())[1:], "no-such-subcommand")
				err := execute(context.Background(), args)
				want := `unknown command "no-such-subcommand" for "` + sub.CommandPath() + `"`
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Errorf("error = %v, want one containing %q", err, want)
				}
			})
			walk(sub)
		}
	}
	walk(rootCmd)
}
