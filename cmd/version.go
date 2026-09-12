package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/meimolihan/fan-files/version"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("fan-files v" + version.Version + "/" + version.CommitSHA)
	},
}
