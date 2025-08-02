package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version information, set during build
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print version, git commit, and build date information for golte.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("golte version %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Built: %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
