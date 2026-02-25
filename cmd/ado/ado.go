package ado

import (
	"bear_cli/internal/ado"
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var org string
var pat string
var project string

var AdoCmd = &cobra.Command{
	Use:   "ado",
	Short: "Azure DevOps API utilities",
	Long:  "Interact with Azure DevOps REST API from the CLI.",
}

func init() {
	AdoCmd.PersistentFlags().StringVar(&org, "org", "", "Azure DevOps organization (required)")
	AdoCmd.PersistentFlags().StringVar(&pat, "pat", "", "Azure DevOps Personal Access Token (required)")
	AdoCmd.PersistentFlags().StringVar(&project, "project", "", "Azure DevOps project (required)")

	AdoCmd.MarkPersistentFlagRequired("org")
	AdoCmd.MarkPersistentFlagRequired("pat")

	AdoCmd.AddCommand(listProjectsCmd())
	AdoCmd.AddCommand(listVariableGroupsCmd())
}

func listProjectsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-projects",
		Short: "List Azure DevOps projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ado.NewAzureDevOpsClient(org, "", pat)
			resp, err := client.ListProjects(context.Background())
			if err != nil {
				return fmt.Errorf("API error: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("decode error: %w", err)
			}
			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}
}

func listVariableGroupsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-variable-groups",
		Short: "List Azure DevOps variable groups in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if org == "" || pat == "" || project == "" {
				return fmt.Errorf("--org, --pat, and --project are required")
			}
			client := ado.NewAzureDevOpsClient(org, project, pat)
			resp, err := client.ListVariableGroups(context.Background())
			if err != nil {
				return fmt.Errorf("API error: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("decode error: %w", err)
			}
			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}
}
