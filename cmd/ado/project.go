package ado

import (
	"bear_cli/internal/ado"
	"bear_cli/pkg/prompt"
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func listProjectsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-projects",
		Short: "List Azure DevOps projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Retrieving Azure DevOps projects...")
			cfg, err := loadBootstrapConfig()
			if err != nil {
				stopSpinner(err)
				return err
			}

			client := ado.NewAzureDevOpsClient(cfg.Org, "", cfg.PAT)
			result, err := client.ListProjectsData(context.Background())
			stopSpinner(err)
			if err != nil {
				return fmt.Errorf("API error: %w", err)
			}

			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}
}
