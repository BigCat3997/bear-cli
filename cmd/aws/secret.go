package aws

import (
	"context"
	"fmt"

	"bear_cli/internal/aws"
	"bear_cli/pkg/prompt"

	"github.com/spf13/cobra"
)

var SecretCmd = &cobra.Command{
	Use:   "secret",
	Short: "AWS Secrets Manager utilities",
	Long:  "Create and manage secrets in AWS Secrets Manager.",
}

func init() {
	SecretCmd.AddCommand(createSecretCmd())
	SecretCmd.AddCommand(getSecretCmd())
	SecretCmd.AddCommand(deleteSecretCmd())
}

type createSecretOptions struct {
	Name        string
	SecretValue string
	Description string
	Tags        []string
	Override    bool
	Region      string
	Profile     string
}

func createSecretCmd() *cobra.Command {
	opts := &createSecretOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a secret in AWS Secrets Manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Creating secret...")
			result, err := aws.CreateSecret(context.Background(), aws.CreateSecretInput{
				Name:        opts.Name,
				SecretValue: opts.SecretValue,
				Description: opts.Description,
				Tags:        opts.Tags,
				Override:    opts.Override,
				Region:      opts.Region,
				Profile:     opts.Profile,
			})
			stopSpinner(err)
			if err != nil {
				return err
			}
			fmt.Printf("Secret %q created successfully\n", result.Name)
			fmt.Printf("ARN: %s\n", result.ARN)
			if result.VersionID != "" {
				fmt.Printf("Version ID: %s\n", result.VersionID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Name of the secret (required)")
	cmd.Flags().StringVar(&opts.SecretValue, "value", "", "Secret value (required)")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Description of the secret")
	cmd.Flags().StringSliceVar(&opts.Tags, "tags", []string{}, "Tags to attach to the secret (format: key=value)")
	cmd.Flags().BoolVar(&opts.Override, "override", false, "Update the secret value if the secret already exists")
	cmd.Flags().StringVar(&opts.Region, "region", "", "AWS region")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "AWS profile")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("value")

	return cmd
}

type getSecretOptions struct {
	Name    string
	Region  string
	Profile string
}

func getSecretCmd() *cobra.Command {
	opts := &getSecretOptions{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieve a secret from AWS Secrets Manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Retrieving secret...")
			result, err := aws.GetSecret(context.Background(), aws.GetSecretInput{
				Name:    opts.Name,
				Region:  opts.Region,
				Profile: opts.Profile,
			})
			stopSpinner(err)
			if err != nil {
				return err
			}
			fmt.Printf("Secret: %s\n", result.SecretValue)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Name of the secret (required)")
	cmd.Flags().StringVar(&opts.Region, "region", "", "AWS region")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "AWS profile")
	cmd.MarkFlagRequired("name")

	return cmd
}

type deleteSecretOptions struct {
	Name         string
	RecoveryDays int64
	ForceDelete  bool
	Region       string
	Profile      string
}

func deleteSecretCmd() *cobra.Command {
	opts := &deleteSecretOptions{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a secret from AWS Secrets Manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Deleting secret...")
			result, err := aws.DeleteSecret(context.Background(), aws.DeleteSecretInput{
				Name:         opts.Name,
				RecoveryDays: opts.RecoveryDays,
				ForceDelete:  opts.ForceDelete,
				Region:       opts.Region,
				Profile:      opts.Profile,
			})
			stopSpinner(err)
			if err != nil {
				return err
			}
			fmt.Printf("Secret %q scheduled for deletion\n", result.Name)
			if result.DeletionDate != "" {
				fmt.Printf("Deletion date: %s\n", result.DeletionDate)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Name of the secret (required)")
	cmd.Flags().Int64Var(&opts.RecoveryDays, "recovery-days", 7, "Recovery window in days (0 to delete immediately)")
	cmd.Flags().BoolVar(&opts.ForceDelete, "force", false, "Force immediate deletion (no recovery window)")
	cmd.Flags().StringVar(&opts.Region, "region", "", "AWS region")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "AWS profile")
	cmd.MarkFlagRequired("name")

	return cmd
}
