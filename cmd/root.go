package cmd

import (
	_ "embed"
	"log"
	"strings"

	"bear_cli/cmd/ado"
	"bear_cli/cmd/aws"
	"bear_cli/cmd/godaddy"
	jenkinscmd "bear_cli/cmd/jenkins"
	"bear_cli/cmd/ps"
	"bear_cli/cmd/terraform"
	"bear_cli/internal/config"

	"github.com/spf13/cobra"
)

//go:embed assets/banner.txt
var banner string

func NewRootCommand(cfg config.Config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bear",
		Short: "A powerful automation toolkit for general DevOps purposes.",
		Long:  strings.TrimRight(banner, "\n") + "\n\nManage DevOps resources and infrastructure workflows from a single CLI.",
	}
	rootCmd.AddCommand(GetVersionCmd())
	rootCmd.AddCommand(ps.NewCommand(cfg.Jenkins))
	rootCmd.AddCommand(ado.AdoCmd)
	rootCmd.AddCommand(aws.CodeCommitCmd)
	rootCmd.AddCommand(aws.DynamoDBCmd)
	rootCmd.AddCommand(aws.SecretCmd)
	rootCmd.AddCommand(godaddy.GoDaddyCmd)
	rootCmd.AddCommand(jenkinscmd.NewCommand(cfg.Jenkins))
	rootCmd.AddCommand(terraform.NewCommand(cfg.Terraform))
	return rootCmd
}

func Execute() {
	result, err := config.Load(config.LoadOptions{})
	if err != nil {
		log.Fatal(err)
	}

	rootCmd := NewRootCommand(result.Config)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
