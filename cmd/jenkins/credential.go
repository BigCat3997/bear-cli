package jenkins

import (
	"fmt"
	"os"
	"strings"

	jenkinsservice "bear_cli/internal/jenkins"

	"github.com/spf13/cobra"
)

type secretTextOptions struct {
	ID          string
	Secret      string
	SecretFile  string
	SecretEnv   string
	Description string
	Scope       string
	Override    bool
}

type sshPrivateKeyOptions struct {
	ID             string
	Username       string
	PrivateKeyFile string
	PrivateKeyEnv  string
	PassphraseFile string
	PassphraseEnv  string
	Description    string
	Scope          string
	Override       bool
}

func newCredentialCommand(connection *connectionOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "credential", Short: "Manage Jenkins credentials"}
	cmd.AddCommand(newCreateSecretTextCommand(connection))
	cmd.AddCommand(newCreateSSHPrivateKeyCommand(connection))
	return cmd
}

func newCreateSecretTextCommand(connection *connectionOptions) *cobra.Command {
	opts := &secretTextOptions{}
	cmd := &cobra.Command{
		Use:   "create-secret-text",
		Short: "Create or update a Jenkins Secret Text credential",
		Example: `  BEAR_JENKINS_URL=http://localhost:8080 \
  BEAR_JENKINS_USER_ID=admin BEAR_JENKINS_API_TOKEN=... \
  bear jenkins credential create-secret-text \
    --id terraform-client-secret --secret-env ARM_CLIENT_SECRET

  bear jenkins credential create-secret-text \
    --url http://localhost:8080 --user admin \
    --id deploy-token --secret-file ./deploy-token.txt --scope SYSTEM --override`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			secret, err := resolveSecretText(opts)
			if err != nil {
				return err
			}

			resolved := connection.resolved()
			client := jenkinsservice.NewClient(resolved.URL, resolved.UserID, resolved.APIToken)
			result, err := client.PutSecretTextCredential(cmd.Context(), jenkinsservice.SecretTextCredentialInput{
				ID: opts.ID, Secret: secret, Description: opts.Description, Scope: opts.Scope, Override: opts.Override,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Jenkins Secret Text credential %q %s with %s scope\n", result.ID, result.Status, result.Scope)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "Credential ID")
	cmd.Flags().StringVar(&opts.Secret, "secret", "", "Secret value (prefer --secret-env or --secret-file)")
	cmd.Flags().StringVar(&opts.SecretFile, "secret-file", "", "Read the secret from a file; one trailing newline is removed")
	cmd.Flags().StringVar(&opts.SecretEnv, "secret-env", "", "Read the secret from this environment variable")
	cmd.Flags().StringVar(&opts.Description, "description", "Managed by Bear CLI", "Credential description")
	cmd.Flags().StringVar(&opts.Scope, "scope", "GLOBAL", "Credential scope: GLOBAL or SYSTEM")
	cmd.Flags().BoolVar(&opts.Override, "override", false, "Update the credential when the ID already exists")
	_ = cmd.MarkFlagRequired("id")
	cmd.MarkFlagsOneRequired("secret", "secret-file", "secret-env")
	cmd.MarkFlagsMutuallyExclusive("secret", "secret-file", "secret-env")
	return cmd
}

func resolveSecretText(opts *secretTextOptions) (string, error) {
	switch {
	case opts.SecretFile != "":
		content, err := os.ReadFile(opts.SecretFile)
		if err != nil {
			return "", fmt.Errorf("read secret file: %w", err)
		}
		return strings.TrimSuffix(strings.TrimSuffix(string(content), "\n"), "\r"), nil
	case opts.SecretEnv != "":
		secret, ok := os.LookupEnv(opts.SecretEnv)
		if !ok {
			return "", fmt.Errorf("secret environment variable %q is not set", opts.SecretEnv)
		}
		return secret, nil
	default:
		return opts.Secret, nil
	}
}

func newCreateSSHPrivateKeyCommand(connection *connectionOptions) *cobra.Command {
	opts := &sshPrivateKeyOptions{}
	cmd := &cobra.Command{
		Use:   "create-ssh-private-key",
		Short: "Create or update an SSH Username with private key credential",
		Example: `  BEAR_JENKINS_URL=http://localhost:8080 \
  BEAR_JENKINS_USER_ID=admin BEAR_JENKINS_API_TOKEN=... \
  bear jenkins credential create-ssh-private-key \
    --id ps-agent-1 --username ubuntu \
    --private-key-file ~/.ssh/mac_to_linux --scope SYSTEM --override`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			privateKey, err := resolveFileOrEnvironment(opts.PrivateKeyFile, opts.PrivateKeyEnv, "private key")
			if err != nil {
				return err
			}

			passphrase := ""
			if opts.PassphraseFile != "" || opts.PassphraseEnv != "" {
				passphrase, err = resolveFileOrEnvironment(opts.PassphraseFile, opts.PassphraseEnv, "passphrase")
				if err != nil {
					return err
				}
				passphrase = trimOneTrailingNewline(passphrase)
			}

			resolved := connection.resolved()
			client := jenkinsservice.NewClient(resolved.URL, resolved.UserID, resolved.APIToken)
			result, err := client.PutSSHPrivateKeyCredential(cmd.Context(), jenkinsservice.SSHPrivateKeyCredentialInput{
				ID: opts.ID, Username: opts.Username, PrivateKey: privateKey, Passphrase: passphrase,
				Description: opts.Description, Scope: opts.Scope, Override: opts.Override,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Jenkins SSH private-key credential %q %s with %s scope\n", result.ID, result.Status, result.Scope)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "Credential ID")
	cmd.Flags().StringVar(&opts.Username, "username", "", "SSH username on the remote machine")
	cmd.Flags().StringVar(&opts.PrivateKeyFile, "private-key-file", "", "Read the SSH private key from this file")
	cmd.Flags().StringVar(&opts.PrivateKeyEnv, "private-key-env", "", "Read the SSH private key from this environment variable")
	cmd.Flags().StringVar(&opts.PassphraseFile, "passphrase-file", "", "Read the optional private-key passphrase from this file")
	cmd.Flags().StringVar(&opts.PassphraseEnv, "passphrase-env", "", "Read the optional private-key passphrase from this environment variable")
	cmd.Flags().StringVar(&opts.Description, "description", "Managed by Bear CLI", "Credential description")
	cmd.Flags().StringVar(&opts.Scope, "scope", "GLOBAL", "Credential scope: GLOBAL or SYSTEM")
	cmd.Flags().BoolVar(&opts.Override, "override", false, "Update the credential when the ID already exists")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("username")
	cmd.MarkFlagsOneRequired("private-key-file", "private-key-env")
	cmd.MarkFlagsMutuallyExclusive("private-key-file", "private-key-env")
	cmd.MarkFlagsMutuallyExclusive("passphrase-file", "passphrase-env")
	return cmd
}

func resolveFileOrEnvironment(filePath, envName, valueName string) (string, error) {
	if filePath != "" {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read %s file: %w", valueName, err)
		}
		return string(content), nil
	}

	value, ok := os.LookupEnv(envName)
	if !ok {
		return "", fmt.Errorf("%s environment variable %q is not set", valueName, envName)
	}
	return value, nil
}

func trimOneTrailingNewline(value string) string {
	return strings.TrimSuffix(strings.TrimSuffix(value, "\n"), "\r")
}
