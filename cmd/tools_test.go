package cmd

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/napicella/dis/internal/rcstate"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

// runRC runs a dis tools command as the installer of pkg, in home. The flag
// variables and their Changed marks are reset first: cobra keeps them between
// runs of the same command.
func runRC(t *testing.T, home, pkg string, args ...string) error {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("DIS_PACKAGE", pkg)
	rcName, rcContent, rcOwner, rcPaths, rcFile = "", "", "", nil, ""
	for _, c := range toolsCmd.Commands() {
		c.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
	}
	_, err := runCmd(t, append([]string{"tools"}, args...)...)
	return err
}

// rcState returns the sections of the generated files in home.
func rcState(t *testing.T, home string) map[string][]rcstate.Section {
	t.Helper()
	t.Setenv("HOME", home)
	store, err := rcstate.DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	st, _, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	return st.Files
}

func TestAddRCEnv(t *testing.T) {
	home := t.TempDir()
	if err := runRC(t, home, "common/go", "add-rc-env", "--name", "GOBIN env", "--content", `export GOBIN="$HOME/go/bin"`); err != nil {
		t.Fatal(err)
	}
	want := []rcstate.Section{{Name: "GOBIN env", Owner: "common/go", Content: `export GOBIN="$HOME/go/bin"`}}
	if got := rcState(t, home)["bash_env"]; !reflect.DeepEqual(got, want) {
		t.Errorf("bash_env = %+v, want %+v", got, want)
	}
	if err := runRC(t, home, "common/go", "add-rc-env", "--name", "x"); err == nil || !strings.Contains(err.Error(), `required flag(s) "content" not set`) {
		t.Errorf("no --content: error = %v", err)
	}
}

func TestAddRCPathContentRemoved(t *testing.T) {
	home := t.TempDir()
	err := runRC(t, home, "common/x", "add-rc-path", "--name", "Editor default", "--content", "export EDITOR=vim")
	if err == nil || err.Error() != "add-rc-path --content was removed: write exports with 'dis tools add-rc-env'" {
		t.Errorf("--content: error = %v", err)
	}
	if err := runRC(t, home, "common/x", "add-rc-path", "--name", "Bin"); err == nil || !strings.Contains(err.Error(), `required flag(s) "path" not set`) {
		t.Errorf("no --path: error = %v", err)
	}
	if got := rcState(t, home)["bash_paths"]; len(got) != 0 {
		t.Errorf("bash_paths = %+v", got)
	}
	if f := addRCPathCmd.Flags().Lookup("content"); f == nil || !f.Hidden {
		t.Errorf("--content is not a hidden flag: %+v", f)
	}

	if err := runRC(t, home, "common/x", "add-rc-path", "--name", "Bin", "--path", "/b/", "--path", "/a"); err != nil {
		t.Fatal(err)
	}
	want := []rcstate.Section{{Name: "Bin", Owner: "common/x", Content: "case \":$PATH:\" in *\":/a:\"*) ;; *) export PATH=\"/a:$PATH\" ;; esac\n" +
		"case \":$PATH:\" in *\":/b:\"*) ;; *) export PATH=\"/b:$PATH\" ;; esac"}}
	if got := rcState(t, home)["bash_paths"]; !reflect.DeepEqual(got, want) {
		t.Errorf("bash_paths = %+v, want %+v", got, want)
	}
}

// The section of the same name that 'add-rc-path --content' wrote moves to
// bash_env, unless another package owns it or locked it.
func TestAddRCEnvMovesFromBashPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, err := rcstate.DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	old := []rcstate.Section{
		{Name: "Editor default", Owner: "common/bash-config", Content: "export EDITOR=vim"},
		{Name: "Locale", Owner: "common/os-libs", Content: "export LANG=C"},
		{Name: "Pinned", Owner: "tools/pin", Locked: true, Content: "export P=1"},
		{Name: "Mine", Owner: "common/bash-config", Locked: true, Content: "export M=1"},
		{Name: "./local/bin", Owner: "common/bash-config", Content: `case ":$PATH:" in *":$HOME/.local/bin:"*) ;; *) export PATH="$HOME/.local/bin:$PATH" ;; esac`},
	}
	err = store.Update(func(st *rcstate.State) error {
		st.Files["bash_paths"] = append([]rcstate.Section(nil), old...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"Editor default", "Locale", "Pinned", "Mine"} {
		if err := runRC(t, home, "common/bash-config", "add-rc-env", "--name", name, "--content", "export NEW=1"); err != nil {
			t.Fatal(err)
		}
	}
	files := rcState(t, home)
	// Locale is owned by another package and Pinned locked by one: they stay.
	if got, want := files["bash_paths"], []rcstate.Section{old[1], old[2], old[4]}; !reflect.DeepEqual(got, want) {
		t.Errorf("bash_paths:\ngot  %+v\nwant %+v", got, want)
	}
	var names []string
	for _, s := range files["bash_env"] {
		names = append(names, s.Name)
	}
	if want := []string{"Editor default", "Locale", "Pinned", "Mine"}; !reflect.DeepEqual(names, want) {
		t.Errorf("bash_env sections = %q, want %q", names, want)
	}

	// Running again changes nothing.
	if err := runRC(t, home, "common/bash-config", "add-rc-env", "--name", "Editor default", "--content", "export NEW=1"); err != nil {
		t.Fatal(err)
	}
	if again := rcState(t, home); !reflect.DeepEqual(again, files) {
		t.Errorf("second run:\ngot  %+v\nwant %+v", again, files)
	}
}
