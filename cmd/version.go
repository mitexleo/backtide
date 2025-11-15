package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version will be set during build via ldflags
var version = "dev"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display the current version of Backtide.

This command shows the version number that was set during the build process.
For development builds, this will show "dev".`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Backtide version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
