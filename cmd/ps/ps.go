package ps

import (
	"bear_cli/internal/browser"
	"bear_cli/internal/ps"
	"bear_cli/models"
	"bear_cli/pkg/prompt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var PsCmd = &cobra.Command{
	Use:   string(models.Ps),
	Short: "Provide PluralSight's credential management capabilities.",
	Long:  "Extract, manage, and utilize PluralSight's sandbox credentials for AWS and Azure with ease.",
}

func init() {
	PsCmd.AddCommand(createCredentialCmd())
	PsCmd.AddCommand(getCredentialCmd())
	PsCmd.AddCommand(initCredentialCmd())
	PsCmd.AddCommand(loginCmd())
}

type PsCreateCredentialOptions struct {
	UseClipboard  bool
	FilePath      string
	CloudProvider string
	Login         bool
	Output        string
	Scope         string
}

func createCredentialCmd() *cobra.Command {
	opts := &PsCreateCredentialOptions{}

	cmd := &cobra.Command{
		Use:   string(models.PsCreateCredential),
		Short: models.CommandDescriptions[models.PsCreateCredential],
		RunE: func(cmd *cobra.Command, args []string) error {
			format := models.ParseStdOutFormat(opts.Output)
			scope := models.ParseCredentialScope(opts.Scope)

			if strings.EqualFold(opts.CloudProvider, "azure") {
				cred := ps.CreatePsAzureCredential(opts.UseClipboard, opts.FilePath)
				prompt.PrintStdOut(cred.ToScopedEnvMap(scope), format)
				if opts.Login {
					browser.LoginInBrowser(cred.User, cred.Password, browser.AzurePortal, cred.SandboxURL)
				}
			} else {
				cred := ps.CreatePsAWSCredential(opts.UseClipboard, opts.FilePath)
				prompt.PrintStdOut(cred.ToScopedEnvMap(scope), format)
				if opts.Login {
					browser.LoginInBrowser(cred.User, cred.Password, browser.AWSConsole, cred.SandboxURL)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.UseClipboard, "clipboard", "", true, "Read HTML from clipboard")
	cmd.Flags().StringVarP(&opts.FilePath, "html-path", "", "", "Path of HTML file")
	cmd.Flags().StringVarP(&opts.CloudProvider, "cloud-provider", "", "Azure", "Cloud provider (aws, azure)")
	cmd.Flags().BoolVarP(&opts.Login, "login", "", false, "Will login or not")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "env", "Output format: env, json, table")
	cmd.Flags().StringVarP(&opts.Scope, "scope", "s", "full", "Credential scope: full, terraform")

	return cmd
}

type PluralSightOptions struct {
	UseClipboard bool
	HTMLPath     string
	Login        bool
}

func getCredentialCmd() *cobra.Command {
	opts := &PluralSightOptions{}

	cmd := &cobra.Command{
		Use:   string(models.PsGetCredential),
		Short: models.CommandDescriptions[models.PsGetCredential],
		RunE: func(cmd *cobra.Command, args []string) error {
			cred, error := ps.LoadSandboxCredential()
			if error != nil {
				return error
			}
			prompt.PrintStdOut(cred.ToEnvMap(), models.LINUX_ENV_VAR)
			if opts.Login {
				ps.LoginAzurePortalFromSandbox()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.HTMLPath, "html-path", "", "", "Path of HTML file")
	cmd.Flags().BoolVarP(&opts.UseClipboard, "clipboard", "", true, "Read HTML from clipboard")
	cmd.Flags().BoolVarP(&opts.Login, "login", "", false, "Will login or not")

	return cmd
}

type initPsCredentialOptions struct {
	Path        string
	SandboxPath string
}

func initCredentialCmd() *cobra.Command {
	opts := &initPsCredentialOptions{}

	cmd := &cobra.Command{
		Use:   string(models.PsInitCredential),
		Short: models.CommandDescriptions[models.PsInitCredential],
		RunE: func(cmd *cobra.Command, args []string) error {
			ps.ReplaceResourceGroupInPath(opts.Path, opts.SandboxPath)
			ps.RemoveTerraformStateFiles(opts.Path)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Path, "path", "", "", "Target path to re-init credential.")
	cmd.Flags().StringVarP(&opts.SandboxPath, "sandbox-path", "", "", "Path of sandbox.json file.")

	return cmd
}

func loginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   string(models.PsLoginByCredential),
		Short: models.CommandDescriptions[models.PsLoginByCredential],
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxUrl := os.Getenv("ARM_SANDBOX_URL")
			username := os.Getenv("ARM_USERNAME")
			password := os.Getenv("ARM_PASSWORD")
			if username != "" && password != "" {
				browser.LoginInBrowser(username, password, browser.AzurePortal, sandboxUrl)
			} else {
				ps.LoginAzurePortalFromSandbox()
			}
			return nil
		},
	}

	return cmd
}
