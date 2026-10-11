package ado

import (
	"bear_cli/internal/ado"
	"bear_cli/pkg/prompt"
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type AdoOptions struct {
	Project string
}

type addVariableGroupOptions struct {
	Project     string
	Group       string
	Description string
	Variables   []string
	SecretVars  []string
	Override    bool
}

func listVariableGroupsCmd() *cobra.Command {
	opts := &AdoOptions{}

	cmd := &cobra.Command{
		Use:   "list-variable-groups",
		Short: "List Azure DevOps variable groups in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Retrieving Azure DevOps variable groups...")
			cfg, err := loadBootstrapConfig()
			if err != nil {
				stopSpinner(err)
				return err
			}

			client := ado.NewAzureDevOpsClient(cfg.Org, opts.Project, cfg.PAT)
			result, err := client.ListVariableGroupsData(context.Background())
			stopSpinner(err)
			if err != nil {
				return fmt.Errorf("API error: %w", err)
			}

			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Azure DevOps project")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func addVariableGroupCmd() *cobra.Command {
	opts := &addVariableGroupOptions{}

	cmd := &cobra.Command{
		Use:   "add-variable-group",
		Short: "Create a variable group or add variables to an existing one",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			stopSpinner := prompt.StartSpinner("Updating Azure DevOps variable group...")

			cfg, err := loadBootstrapConfig()
			if err != nil {
				stopSpinner(err)
				return err
			}

			client := ado.NewAzureDevOpsClient(cfg.Org, opts.Project, cfg.PAT)
			result, err := client.UpsertVariableGroup(ctx, ado.UpsertVariableGroupInput{
				Project:         opts.Project,
				Group:           opts.Group,
				Description:     opts.Description,
				Variables:       opts.Variables,
				SecretVariables: opts.SecretVars,
				Override:        opts.Override,
			})
			stopSpinner(err)
			if err != nil {
				return err
			}

			output, err := json.MarshalIndent(result.Group, "", "  ")
			if err != nil {
				return fmt.Errorf("encode result: %w", err)
			}

			fmt.Printf("Variable group %s\n%s\n", result.Status, string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Azure DevOps project")
	cmd.Flags().StringVar(&opts.Group, "group", "", "Variable group name")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Variable group description")
	cmd.Flags().StringSliceVar(&opts.Variables, "variable", nil, "Variable to add in key=value form (repeat flag or provide CSV)")
	cmd.Flags().StringSliceVar(&opts.SecretVars, "secret-variable", nil, "Secret variable to add in key=value form (repeat flag or provide CSV)")
	cmd.Flags().BoolVar(&opts.Override, "override", false, "Replace existing variables when the group already contains the same key")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("group")
	cmd.MarkFlagsOneRequired("variable", "secret-variable")

	return cmd
}
