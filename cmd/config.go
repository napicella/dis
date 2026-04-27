package cmd

import (
	"fmt"
	"os"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var cfgCmd = &cobra.Command{
	Use:   "config",
	Short: "Create default configurations",
}

var cfgDistroCmd = &cobra.Command{
	Use:   "distro",
	Short: "Create a distro file",
	RunE:  cfgDistroCmdFn,
}

var cfgInstCmd = &cobra.Command{
	Use:   "installer",
	Short: "Create an empty installer",
	RunE:  cfgInstCmdFn,
}

func init() {
	cfgCmd.AddCommand(cfgDistroCmd, cfgInstCmd)
	rootCmd.AddCommand(cfgCmd)
}

func cfgDistroCmdFn(cmd *cobra.Command, _ []string) error {
	const fn = "distro-templ.yml"
	
	fmt.Printf("Creating %s\n", fn)
	if err := createDistroCfg(fn); err != nil {
		fmt.Printf("create error: %s\n", err)
		return err
	}

	fmt.Println("Success")
	return nil
}

func createDistroCfg(filename string) error {
	d := &dis.DistroConfig{
		OS:       "ubuntu",
		Sources:  []string{"${common_sources}"},
		Packages: []string{"example-1", "example-2"},
		Parameters: map[string]string{
			"MY_PARAM_1": "VAL_1",
			"MY_PARAM_2": "VAL_2",
		},
		ConfigGenerators: []dis.ConfigGenerator{{
			Script: "path/to/generator",
		}},
		Preconditions: []dis.Precondition{{
			Script: "path/to/script",
			Uses:   []string{"MY_PARAM_1"},
		}},
	}
	w, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create distro file: %w", err)
	}
	enc := yaml.NewEncoder(w)
	defer enc.Close()

	err = enc.Encode(d)
	if err != nil {
		return fmt.Errorf("create distro file: %w", err)
	}
	return nil
}

func cfgInstCmdFn(cmd *cobra.Command, _ []string) error {
	const fn = "installer-templ.sh"

	fmt.Printf("Creating %s\n", fn)
	if err := createInstallerCfg(fn); err != nil {
		fmt.Printf("create error: %s\n", err)
		return err
	}

	fmt.Println("Success")
	return nil
}

func createInstallerCfg(filename string) error {
	w, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create installer file: %w", err)
	}
	defer w.Close()

	_, err = fmt.Fprint(w, dis.InstallerTemplate())
	if err != nil {
		return fmt.Errorf("write installer file: %w", err)
	}
	return nil
}
