package cmd

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/napicella/dis/internal/dis"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search packages available frm the sources defined in the disto file",
	RunE:  searchCmdFn,
}

var searchReg string

func init() {
	searchCmd.Flags().StringVarP(&distroFile, "distro", "d", "",
		"Path to the distro YAML file (required)")
	searchCmd.Flags().StringVarP(&searchReg, "regex", "r", "",
		"A golang regular expression (https://pkg.go.dev/regexp) used to match the installer name")
	searchCmd.Flags().StringVarP(&installCommonSources, "sources", "s", "",
		"Path to use for ${common_sources} (overrides auto-detection)")
	_ = searchCmd.MarkFlagRequired("distro")
	_ = searchCmd.MarkFlagRequired("regex")
	rootCmd.AddCommand(searchCmd)
}

func searchCmdFn(_ *cobra.Command, _ []string) error {
	ic, err := dis.NewInstallContextWithCache(distroFile, installCommonSources)
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
