package ps

import (
	"bear_cli/internal/ado"
	"bear_cli/internal/browser"
	"bear_cli/internal/cli"
	"bear_cli/internal/config"
	"bear_cli/internal/jenkins"
	"bear_cli/internal/jsonfile"
	ps "bear_cli/internal/pluralsight"
	"bear_cli/pkg/prompt"
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type adoBootstrapConfigOptions struct {
	Org string
	PAT string
}

const defaultTerraformVariableGroup = "vg-terraform-cred"

func loadAdoBootstrapConfig() (*adoBootstrapConfigOptions, error) {
	configPath, err := ado.LoadBootstrapConfigPath()
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	cfg, err := jsonfile.Load[adoBootstrapConfigOptions](configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &cfg, nil
}

func pushTerraformVarsToAdoVariableGroup(ctx context.Context, vars map[string]string, project, group string, override bool) (*ado.UpsertVariableGroupResult, error) {
	if project == "" {
		return nil, fmt.Errorf("--ado-project is required when --push-variable-group is enabled")
	}

	cfg, err := loadAdoBootstrapConfig()
	if err != nil {
		return nil, err
	}

	normalVars, secretVars := toAdoVariableInputs(vars)

	client := ado.NewAzureDevOpsClient(cfg.Org, project, cfg.PAT)
	result, err := client.UpsertVariableGroup(ctx, ado.UpsertVariableGroupInput{
		Project:         project,
		Group:           group,
		Description:     "Terraform sandbox credentials managed by Bear CLI",
		Variables:       normalVars,
		SecretVariables: secretVars,
		Override:        override,
	})
	if err != nil {
		return nil, fmt.Errorf("push variable group: %w", err)
	}

	return result, nil
}

type PsCreateCredentialOptions struct {
	UseClipboard      bool
	FilePath          string
	CloudProvider     string
	Login             bool
	Output            string
	Scope             string
	PushVariableGroup bool
	ADOProject        string
	PublishToADO      string
	VariableGroup     string
	PublishToJenkins  string
	JenkinsUser       string
	JenkinsAPIToken   string
	JenkinsScope      string
	JenkinsPrefix     string
	Override          bool
}

type createCredentialFlagState struct {
	publishToADO     bool
	publishToJenkins bool
	scope            bool
	output           bool
	variableGroup    bool
}

func prepareAndValidateCreateCredentialOptions(opts *PsCreateCredentialOptions, changed createCredentialFlagState) error {
	opts.CloudProvider = strings.ToLower(strings.TrimSpace(opts.CloudProvider))
	opts.Scope = strings.ToLower(strings.TrimSpace(opts.Scope))
	opts.Output = strings.ToLower(strings.TrimSpace(opts.Output))
	opts.FilePath = strings.TrimSpace(opts.FilePath)
	opts.ADOProject = strings.TrimSpace(opts.ADOProject)
	opts.PublishToADO = strings.TrimSpace(opts.PublishToADO)
	opts.VariableGroup = strings.TrimSpace(opts.VariableGroup)
	opts.PublishToJenkins = strings.TrimRight(strings.TrimSpace(opts.PublishToJenkins), "/")
	opts.JenkinsUser = strings.TrimSpace(opts.JenkinsUser)
	opts.JenkinsAPIToken = strings.TrimSpace(opts.JenkinsAPIToken)
	opts.JenkinsScope = strings.ToUpper(strings.TrimSpace(opts.JenkinsScope))
	opts.JenkinsPrefix = strings.TrimSpace(opts.JenkinsPrefix)

	if changed.publishToADO && opts.PublishToADO == "" {
		return fmt.Errorf("--publish-to-ado requires an Azure DevOps project")
	}
	if opts.PublishToADO != "" {
		if opts.ADOProject != "" && opts.ADOProject != opts.PublishToADO {
			return fmt.Errorf("--publish-to-ado and deprecated --ado-project refer to different projects")
		}
		opts.PushVariableGroup = true
		opts.ADOProject = opts.PublishToADO
		if !changed.scope {
			opts.Scope = string(ps.ScopeTerraform)
		}
		if !changed.output {
			opts.Output = "none"
		}
	}
	if changed.publishToJenkins && opts.PublishToJenkins == "" {
		return fmt.Errorf("--publish-to-jenkins requires a Jenkins URL")
	}
	if opts.PublishToJenkins != "" {
		if !changed.scope {
			opts.Scope = string(ps.ScopeTerraform)
		}
		if !changed.output {
			opts.Output = "none"
		}
		if opts.JenkinsUser == "" {
			return fmt.Errorf("--jenkins-user is required when publishing to Jenkins")
		}
		if opts.JenkinsAPIToken == "" {
			return fmt.Errorf("Jenkins API token is required; set BEAR_JENKINS_API_TOKEN, configure jenkins_api_token, or use --jenkins-api-token")
		}
		if opts.JenkinsScope != "GLOBAL" && opts.JenkinsScope != "SYSTEM" {
			return fmt.Errorf("invalid --jenkins-credential-scope %q: expected GLOBAL or SYSTEM", opts.JenkinsScope)
		}
		if opts.JenkinsPrefix == "" {
			opts.JenkinsPrefix = "ps-" + opts.CloudProvider
		}
	}

	if opts.FilePath != "" {
		opts.UseClipboard = false
	}

	if opts.CloudProvider != "aws" && opts.CloudProvider != "azure" {
		return fmt.Errorf("invalid --cloud-provider %q: expected aws or azure", opts.CloudProvider)
	}
	if opts.Scope != string(ps.ScopeFull) && opts.Scope != string(ps.ScopeTerraform) {
		return fmt.Errorf("invalid --scope %q: expected full or terraform", opts.Scope)
	}
	if opts.Output != "env" && opts.Output != "json" && opts.Output != "table" && opts.Output != "none" {
		return fmt.Errorf("invalid --output %q: expected env, json, table, or none", opts.Output)
	}
	if !opts.UseClipboard && opts.FilePath == "" {
		return fmt.Errorf("--html-path is required when --clipboard=false")
	}
	if opts.ADOProject != "" && !opts.PushVariableGroup {
		return fmt.Errorf("--ado-project requires deprecated --push-variable-group; use --publish-to-ado instead")
	}
	if changed.variableGroup && !opts.PushVariableGroup {
		return fmt.Errorf("--variable-group requires --publish-to-ado")
	}
	if opts.PushVariableGroup {
		if opts.Scope != string(ps.ScopeTerraform) {
			return fmt.Errorf("Azure DevOps publishing requires --scope terraform")
		}
		if opts.ADOProject == "" {
			return fmt.Errorf("--ado-project is required when --push-variable-group is enabled")
		}
		if opts.VariableGroup == "" {
			return fmt.Errorf("--variable-group cannot be empty")
		}
	}
	if opts.PublishToJenkins != "" && opts.Scope != string(ps.ScopeTerraform) {
		return fmt.Errorf("Jenkins publishing requires --scope terraform")
	}

	return nil
}

func createCredentialCmd(jenkinsConfig config.JenkinsConfig) *cobra.Command {
	opts := &PsCreateCredentialOptions{}

	cmd := &cobra.Command{
		Use:   string(ps.PsCreateCredential),
		Short: "Extract and use Pluralsight sandbox credentials",
		Long: "Extract AWS or Azure credentials from a Pluralsight sandbox page. " +
			"Credentials can be printed for shell or automation use, published to Azure DevOps or Jenkins, and used to open the cloud console.",
		Example: `  # Read an AWS sandbox page from the clipboard and prepare Terraform.
  bear ps create --cloud-provider aws --scope terraform

  # Publish Terraform credentials to Azure DevOps without printing secrets.
  bear ps create --cloud-provider aws \
    --publish-to-ado <ADO_PROJECT>

  # Publish to a custom variable group and open the AWS console.
  bear ps create --cloud-provider aws \
    --publish-to-ado <ADO_PROJECT> \
    --variable-group infrastructure-credentials --login

  # Create or update Jenkins Secret Text credentials.
  BEAR_JENKINS_API_TOKEN=... bear ps create --cloud-provider aws \
    --publish-to-jenkins http://localhost:8080 \
    --jenkins-user admin --jenkins-credential-scope GLOBAL --override

  # Read a saved Azure sandbox page and open the portal.
  bear ps create --cloud-provider azure --html-path ./sandbox.html --login`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("jenkins-user") {
				opts.JenkinsUser = jenkinsConfig.UserID
			}
			if !cmd.Flags().Changed("jenkins-api-token") {
				opts.JenkinsAPIToken = jenkinsConfig.APIToken
			}
			return prepareAndValidateCreateCredentialOptions(opts, createCredentialFlagState{
				publishToADO:     cmd.Flags().Changed("publish-to-ado"),
				publishToJenkins: cmd.Flags().Changed("publish-to-jenkins"),
				scope:            cmd.Flags().Changed("scope"),
				output:           cmd.Flags().Changed("output"),
				variableGroup:    cmd.Flags().Changed("variable-group"),
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cli.ParseStdOutFormat(opts.Output)
			scope := ps.ParseCredentialScope(opts.Scope)
			var scopedEnv map[string]string
			var loginUser, loginPassword, sandboxURL string
			var loginWebsite browser.Website
			stopSpinner := prompt.StartSpinner("Creating sandbox credentials...")

			if strings.EqualFold(opts.CloudProvider, "azure") {
				cred, err := ps.CreatePsAzureCredential(opts.UseClipboard, opts.FilePath)
				if err != nil {
					stopSpinner(err)
					return err
				}
				scopedEnv = cred.ToScopedEnvMap(scope)
				loginUser, loginPassword, sandboxURL = cred.User, cred.Password, cred.SandboxURL
				loginWebsite = browser.AzurePortal
			} else {
				cred, err := ps.CreatePsAWSCredentialWithError(opts.UseClipboard, opts.FilePath)
				if err != nil {
					stopSpinner(err)
					return err
				}
				scopedEnv = cred.ToScopedEnvMap(scope)
				loginUser, loginPassword, sandboxURL = cred.User, cred.Password, cred.SandboxURL
				loginWebsite = browser.AWSConsole
			}
			stopSpinner(nil)
			prompt.PrintStdOut(scopedEnv, format)

			if opts.PushVariableGroup {
				stopPushSpinner := prompt.StartSpinner("Pushing credentials to Azure DevOps...")
				result, err := pushTerraformVarsToAdoVariableGroup(cmd.Context(), scopedEnv, opts.ADOProject, opts.VariableGroup, opts.Override)
				stopPushSpinner(err)
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "Azure DevOps variable group %q %s in project %q\n", opts.VariableGroup, result.Status, opts.ADOProject)
			}

			if opts.PublishToJenkins != "" {
				stopJenkinsSpinner := prompt.StartSpinner("Creating Jenkins credentials...")
				client := jenkins.NewClient(opts.PublishToJenkins, opts.JenkinsUser, opts.JenkinsAPIToken)
				created, err := client.CreateSecretTextCredentials(cmd.Context(), scopedEnv, jenkins.PublishOptions{
					Scope:       opts.JenkinsScope,
					IDPrefix:    opts.JenkinsPrefix,
					Description: "Terraform sandbox credential managed by Bear CLI",
					Override:    opts.Override,
				})
				stopJenkinsSpinner(err)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Published %d Jenkins %s credentials with prefix %q\n", len(created), opts.JenkinsScope, opts.JenkinsPrefix)
			}

			if opts.Login {
				fmt.Fprintf(cmd.ErrOrStderr(), "Opening %s console login...\n", strings.ToUpper(opts.CloudProvider))
				if err := browser.LoginInBrowser(loginUser, loginPassword, loginWebsite, sandboxURL); err != nil {
					return fmt.Errorf("open %s console login: %w", opts.CloudProvider, err)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.UseClipboard, "clipboard", true, "Read sandbox HTML from the clipboard (disabled when --html-path is set)")
	cmd.Flags().StringVar(&opts.FilePath, "html-path", "", "Read sandbox HTML from this file instead of the clipboard")
	cmd.Flags().StringVar(&opts.CloudProvider, "cloud-provider", "azure", "Sandbox cloud provider: aws or azure")
	cmd.Flags().BoolVar(&opts.Login, "login", false, "Open the provider console and prefill the sandbox login")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "env", "Credential output: env, json, table, or none")
	cmd.Flags().StringVarP(&opts.Scope, "scope", "s", "full", "Credential fields to include: full or terraform")
	cmd.Flags().StringVar(&opts.PublishToADO, "publish-to-ado", "", "Publish Terraform credentials to this Azure DevOps project")
	cmd.Flags().StringVar(&opts.VariableGroup, "variable-group", defaultTerraformVariableGroup, "Azure DevOps variable-group name")
	cmd.Flags().StringVar(&opts.PublishToJenkins, "publish-to-jenkins", "", "Publish Terraform credentials to this Jenkins controller URL")
	cmd.Flags().StringVar(&opts.JenkinsUser, "jenkins-user", "", "Jenkins username (default: BEAR_JENKINS_USER_ID, then config)")
	cmd.Flags().StringVar(&opts.JenkinsAPIToken, "jenkins-api-token", "", "Jenkins API token (prefer BEAR_JENKINS_API_TOKEN)")
	cmd.Flags().StringVar(&opts.JenkinsScope, "jenkins-credential-scope", "GLOBAL", "Jenkins credential scope: GLOBAL or SYSTEM")
	cmd.Flags().StringVar(&opts.JenkinsPrefix, "jenkins-credential-prefix", "", "Credential ID prefix (default: ps-<cloud-provider>)")
	cmd.Flags().BoolVar(&opts.Override, "override", false, "Replace matching variables or Jenkins credential IDs when they already exist")

	// Deprecated compatibility flags. New commands should use --publish-to-ado.
	cmd.Flags().BoolVar(&opts.PushVariableGroup, "push-variable-group", false, "Create or update an Azure DevOps variable group with Terraform credentials")
	cmd.Flags().StringVar(&opts.ADOProject, "ado-project", "", "Azure DevOps project that owns the variable group")
	_ = cmd.Flags().MarkDeprecated("push-variable-group", "use --publish-to-ado <project> instead")
	_ = cmd.Flags().MarkDeprecated("ado-project", "use --publish-to-ado <project> instead")

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
		Use:   string(ps.PsGetCredential),
		Short: ps.CommandDescriptions[ps.PsGetCredential],
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Loading sandbox credentials...")
			cred, err := ps.LoadSandboxCredential()
			stopSpinner(err)
			if err != nil {
				return err
			}
			prompt.PrintStdOut(cred.ToEnvMap(), cli.LINUX_ENV_VAR)
			if opts.Login {
				stopLoginSpinner := prompt.StartSpinner("Opening Azure portal login...")
				err := ps.LoginAzurePortalFromSandbox()
				stopLoginSpinner(err)
				if err != nil {
					return err
				}
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
		Use:   string(ps.PsInitCredential),
		Short: ps.CommandDescriptions[ps.PsInitCredential],
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Initializing sandbox credentials...")
			err := ps.ReplaceResourceGroupInPath(opts.Path, opts.SandboxPath)
			if err == nil {
				err = ps.RemoveTerraformStateFiles(opts.Path)
			}
			stopSpinner(err)
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.Path, "path", "", "", "Target path to re-init credential.")
	cmd.Flags().StringVarP(&opts.SandboxPath, "sandbox-path", "", "", "Path of sandbox.json file.")

	return cmd
}
