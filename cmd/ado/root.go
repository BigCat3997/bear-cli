package ado

import (
	"bear_cli/internal/ado"
	"bear_cli/internal/jsonfile"
	"bear_cli/pkg/prompt"
	"fmt"

	"github.com/spf13/cobra"
)

var org string
var pat string
var project string

type bootstrapConfig struct {
	Org string `json:"org"`
	PAT string `json:"pat"`
}

var AdoCmd = &cobra.Command{
	Use:   "ado",
	Short: "Azure DevOps API utilities",
	Long:  "Interact with Azure DevOps REST API from the CLI.",
}

func init() {
	AdoCmd.AddCommand(bootstrap())
	AdoCmd.AddCommand(listProjectsCmd())
	AdoCmd.AddCommand(listVariableGroupsCmd())
	AdoCmd.AddCommand(addVariableGroupCmd())
}

type AzureBootstrapConfigOptions struct {
	Org string
	PAT string
}

func loadBootstrapConfig() (*AzureBootstrapConfigOptions, error) {
	configPath, err := ado.LoadBootstrapConfigPath()
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	cfg, err := jsonfile.Load[AzureBootstrapConfigOptions](configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &cfg, nil
}

func bootstrap() *cobra.Command {
	opts := &AzureBootstrapConfigOptions{}

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Save Azure DevOps credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Saving Azure DevOps credentials...")
			err := ado.Bootstrap(opts.Org, opts.PAT)
			stopSpinner(err)
			return err
		},
	}

	cmd.Flags().StringVar(&opts.Org, "org", "", "Azure DevOps organization")
	cmd.Flags().StringVar(&opts.PAT, "pat", "", "Azure DevOps Personal Access Token")

	_ = cmd.MarkFlagRequired("org")
	_ = cmd.MarkFlagRequired("pat")

	return cmd
}
