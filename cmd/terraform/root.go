package terraform

import (
	"bear_cli/internal/aws"
	"bear_cli/internal/cli"
	"bear_cli/internal/config"
	ps "bear_cli/internal/pluralsight"
	"bear_cli/pkg/prompt"
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewCommand creates the Terraform command tree. A constructor is used
// instead of a package-global command so the configured default path is
// available to subcommands such as "clean".
func NewCommand(cfg config.TerraformConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terraform",
		Short: "Manage Terraform backend for Azure",
		Long:  "Extract, manage, and utilize PluralSight's sandbox credentials for AWS and Azure with ease.",
	}

	cmd.AddCommand(terraformCmd())
	cmd.AddCommand(bootstrapCmd())
	cmd.AddCommand(cleanCmd(cfg))
	return cmd
}

type terraformOptions struct {
	CloudProvider string
	AccountName   string
	ContainerName string
	Location      string
	Output        string
}

func terraformCmd() *cobra.Command {
	opts := &terraformOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Manage Terraform backend for Azure",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !strings.EqualFold(opts.CloudProvider, "azure") {
				return fmt.Errorf("only azure is supported for now")
			}

			stopSpinner := prompt.StartSpinner("Initializing Terraform backend resources...")
			cred, err := ps.LoadAzureSandboxCredential()
			if err != nil {
				stopSpinner(err)
				return err
			}

			setup, err := ps.SetUpTerraformBackend(
				cred.ClientID,
				cred.ClientSecret,
				cred.TenantName,
				cred.SubscriptionID,
				cred.ResourceGroup,
				opts.AccountName,
				opts.Location,
				opts.ContainerName,
			)
			stopSpinner(err)
			if err != nil {
				return err
			}

			prompt.PrintStdOut(setup.Environment, cli.ParseStdOutFormat(opts.Output))
			fmt.Printf("Terraform backend storage account %q is ready\n", opts.AccountName)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.CloudProvider, "cloud-provider", "azure", "Cloud provider (azure)")
	cmd.Flags().StringVar(&opts.AccountName, "account-name", "", "Azure storage account name (required)")
	cmd.Flags().StringVar(&opts.ContainerName, "container-name", "tfstate", "Azure blob container name")
	cmd.Flags().StringVar(&opts.Location, "location", "", "Azure location (required), e.g. southeastasia")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "env", "Output format: env, json, table")
	_ = cmd.MarkFlagRequired("account-name")
	_ = cmd.MarkFlagRequired("location")

	return cmd
}

func bootstrapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			context2 := context.Background()
			stopSpinner := prompt.StartSpinner("Creating DynamoDB terraform locks table...")
			err := aws.CreateTerraformLocksTable(context2, "terraform-locks")
			aws.CreateS3(context2, "bc-s3-terraformstate-dev-0")
			stopSpinner(err)
			if err != nil {
				return err
			}
			fmt.Println("DynamoDB table \"terraform-locks\" is ready")
			return nil
		},
	}
	return cmd
}
