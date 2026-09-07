package cmd

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search packages available from the sources defined in the distro file",
	RunE:  searchCmdFn,
}

var searchReg string

func init() {
	searchCmd.Flags().String("distro", "", "Path to the distro YAML file")
	searchCmd.Flags().StringVarP(&searchReg, "regex", "r", "",
		"A golang regular expression (https://pkg.go.dev/regexp) used to match the installer name")
	searchCmd.Flags().String("sources", "", "Path to use for ${common_sources} (overrides auto-detection)")
	_ = viper.BindPFlag("distro", searchCmd.Flags().Lookup("distro"))
	_ = viper.BindPFlag("sources", searchCmd.Flags().Lookup("sources"))
	_ = searchCmd.MarkFlagRequired("regex")
	rootCmd.AddCommand(searchCmd)
}

func searchCmdFn(_ *cobra.Command, _ []string) error {
	distroFile := viper.GetString("distro")
	if distroFile == "" {
		return fmt.Errorf("required flag \"distro\" not set and not found in config file")
	}
	commonSources := viper.GetString("sources")

	ic, err := dis.NewInstallContextWithCache(distroFile, commonSources)
	if err != nil {
		return err
	}
	list := ic.ListAvailablePackages()
	sort.Slice(list, func(i, j int) bool {
		return list[i].Provides < list[j].Provides
	})
	var matches []dis.PackageInfo
	for _, s := range list {
		if matched, err := regexp.MatchString(searchReg, s.Provides); matched {
			matches = append(matches, s)
		} else if err != nil {
			return fmt.Errorf("failed to build regex: %w", err)
		}
	}
	for i := 0; i < len(matches); i++ {
		v := matches[i]
		fmt.Printf("Name: %s\nPath: %s\n", v.Provides, v.InstallerPath)
		if i+1 < len(matches) {
			fmt.Println("---")
		}
	}

	return nil
}
