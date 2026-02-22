package cmd

import (
	"bear_cli/cmd/ps"
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bear",
	Short: "A powerful automation toolkit for general DevOps purposes.",
	Long:  "Manage DevOps resources, and infrastructure workflows from a single CLI.",
}

func init() {
	rootCmd.AddCommand(GetVersionCmd())
	rootCmd.AddCommand(ps.PsCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
