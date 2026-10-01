package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newSharedFlagsCmd builds a command that declares the shared flags the same
// way plan/install/config/search do.
func newSharedFlagsCmd(use string, got map[string]string) *cobra.Command {
	c := &cobra.Command{
		Use:     use,
		PreRunE: bindSharedConfigFlags,
		RunE: func(_ *cobra.Command, _ []string) error {
			for _, name := range sharedConfigFlags {
				got[name] = viper.GetString(name)
			}
			return nil
		},
	}
	c.Flags().String("distro", "", "Path to the distro YAML file")
	return c
}

func TestSharedConfigFlagsPrecedence(t *testing.T) {
	const configYAML = "distro: /from/config/distro.yml\n"

	tests := []struct {
		name       string
		config     string
		subcmd     string
		args       []string
		wantDistro string
	}{
		{
			name:       "flags override config",
			config:     configYAML,
			subcmd:     "first",
			args:       []string{"--distro", "/from/flag/distro.yml"},
			wantDistro: "/from/flag/distro.yml",
		},
		{
			// Regression: the last registered command's flag used to win.
			name:       "flag honored on command that is not last registered",
			config:     configYAML,
			subcmd:     "first",
			args:       []string{"--distro", "/from/flag/distro.yml"},
			wantDistro: "/from/flag/distro.yml",
		},
		{
			name:       "flag honored on last registered command",
			config:     configYAML,
			subcmd:     "last",
			args:       []string{"--distro", "/from/flag/distro.yml"},
			wantDistro: "/from/flag/distro.yml",
		},
		{
			name:       "config used when flags not set",
			config:     configYAML,
			subcmd:     "first",
			wantDistro: "/from/config/distro.yml",
		},
		{
			name:       "empty when neither flag nor config set",
			subcmd:     "first",
			wantDistro: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Isolate from the real ~/.config/dis/config.yaml: initConfig runs
			// via cobra.OnInitialize and reads from $HOME and the cwd.
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Chdir(t.TempDir())
			if tt.config != "" {
				cfgDir := filepath.Join(home, ".config", "dis")
				if err := os.MkdirAll(cfgDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(tt.config), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			viper.Reset()
			t.Cleanup(viper.Reset)

			got := map[string]string{}
			root := &cobra.Command{Use: "dis"}
			root.AddCommand(newSharedFlagsCmd("first", got))
			root.AddCommand(newSharedFlagsCmd("last", got))
			root.SetArgs(append([]string{tt.subcmd}, tt.args...))
			if err := root.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}

			if got["distro"] != tt.wantDistro {
				t.Errorf("distro = %q, want %q", got["distro"], tt.wantDistro)
			}
		})
	}
}

// TestCommandsBindSharedFlagsAtRunTime guards against reintroducing init-time
// viper bindings: every command declaring the shared flags must bind them in
// PreRunE.
func TestCommandsBindSharedFlagsAtRunTime(t *testing.T) {
	for _, c := range []*cobra.Command{planCmd, installCmd, configCmd, searchCmd} {
		if c.PreRunE == nil {
			t.Errorf("%s: PreRunE not set; shared flags will not be bound", c.Name())
		}
		for _, name := range sharedConfigFlags {
			if c.Flags().Lookup(name) == nil {
				t.Errorf("%s: missing --%s flag", c.Name(), name)
			}
		}
	}
}

func TestWithDepsFlag(t *testing.T) {
	for _, c := range []*cobra.Command{installCmd, configCmd} {
		f := c.Flags().Lookup("with-deps")
		if f == nil {
			t.Errorf("%s: --with-deps flag not registered", c.Name())
			continue
		}
		if f.DefValue != "false" {
			t.Errorf("%s: --with-deps default = %q, want false", c.Name(), f.DefValue)
		}
	}
}
