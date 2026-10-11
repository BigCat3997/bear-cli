package aws

import (
	"bear_cli/internal/aws"
	"bear_cli/pkg/prompt"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var DynamoDBCmd = &cobra.Command{
	Use:   "dynamodb",
	Short: "AWS DynamoDB utilities",
	Long:  "Create and manage DynamoDB tables.",
}

func init() {
	DynamoDBCmd.AddCommand(createTerraformLocksTableCmd())
}

type createTerraformLocksTableOptions struct {
	TableName string
}

func createTerraformLocksTableCmd() *cobra.Command {
	opts := &createTerraformLocksTableOptions{}

	cmd := &cobra.Command{
		Use:   "create-terraform-locks-table",
		Short: "Create a Terraform locks table in DynamoDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Creating DynamoDB terraform locks table...")
			err := aws.CreateTerraformLocksTable(context.Background(), opts.TableName)
			stopSpinner(err)
			if err != nil {
				return err
			}
			fmt.Printf("DynamoDB table %q is ready\n", opts.TableName)

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.TableName, "table", "", "DynamoDB table name (required)")
	cmd.MarkFlagRequired("table")

	return cmd
}
