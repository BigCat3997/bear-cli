package cmd

import (
	"bear_cli/pkg/prompt"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func GetVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			stopSpinner := prompt.StartSpinner("Loading version information...")
			stopSpinner(nil)
			fmt.Printf("bear %s\n", Version)
			fmt.Printf("commit: %s\n", Commit)
			fmt.Printf("built at: %s\n", BuildDate)
		},
	}
}
