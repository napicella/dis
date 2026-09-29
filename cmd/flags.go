package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// sharedConfigFlags are flags declared by several commands that can also be
// set in the config file (~/.config/dis/config.yaml).
var sharedConfigFlags = []string{"distro"}

// bindSharedConfigFlags binds the running command's shared flags to viper.
//
// It must be called from PreRunE rather than init(): viper keeps a single
// binding per key, so binding the same key from multiple commands at init time
// means only the last registered command's flag is ever consulted. Binding at
// run time ensures the executing command's flag is the one used. An explicitly
// passed flag takes precedence over the config file value.
func bindSharedConfigFlags(cmd *cobra.Command, _ []string) error {
	for _, name := range sharedConfigFlags {
		if err := viper.BindPFlag(name, cmd.Flags().Lookup(name)); err != nil {
			return err
		}
	}
	return nil
}
