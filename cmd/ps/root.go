package ps

import (
	"bear_cli/internal/cli"
	"bear_cli/internal/config"
	ps "bear_cli/internal/pluralsight"
	"bear_cli/pkg/prompt"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func NewCommand(jenkinsConfig config.JenkinsConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   string(ps.Ps),
		Short: "Provide PluralSight's credential management capabilities.",
		Long:  "Extract, manage, and utilize PluralSight's sandbox credentials for AWS and Azure with ease.",
	}
	cmd.AddCommand(createCredentialCmd(jenkinsConfig))
	cmd.AddCommand(getCredentialCmd())
	cmd.AddCommand(initCredentialCmd())
	cmd.AddCommand(terraformCmd())
	cmd.AddCommand(loginCmd())
	return cmd
}

func toAdoVariableInputs(vars map[string]string) ([]string, []string) {
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	normalVars := make([]string, 0, len(keys))
	secretVars := make([]string, 0, len(keys))

	for _, key := range keys {
		entry := fmt.Sprintf("%s=%s", key, vars[key])
		upper := strings.ToUpper(key)
		if strings.Contains(upper, "SECRET") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "TOKEN") {
			secretVars = append(secretVars, entry)
			continue
		}
		normalVars = append(normalVars, entry)
	}

	return normalVars, secretVars
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
		Use:   string(ps.PsTerraform),
		Short: ps.CommandDescriptions[ps.PsTerraform],
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
