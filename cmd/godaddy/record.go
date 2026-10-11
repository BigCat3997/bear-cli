package godaddy

import (
	"bear_cli/pkg/prompt"
	"context"

	"github.com/spf13/cobra"
)

type listRecordsOptions struct {
	Domain     string
	RecordType string
	RecordName string
}

func listRecordsCmd() *cobra.Command {
	opts := &listRecordsOptions{}

	cmd := &cobra.Command{
		Use:   "list-records",
		Short: "List DNS records of a domain",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			stopSpinner := prompt.StartSpinner("Retrieving DNS records...")
			records, err := client.ListRecords(context.Background(), opts.Domain, opts.RecordType, opts.RecordName)
			stopSpinner(err)
			if err != nil {
				return err
			}

			return printJSON(records)
		},
	}

	cmd.Flags().StringVar(&opts.Domain, "domain", "", "Domain name (required)")
	cmd.Flags().StringVar(&opts.RecordType, "type", "", "DNS record type filter (A, CNAME, TXT, etc.)")
	cmd.Flags().StringVar(&opts.RecordName, "name", "", "DNS record name filter (requires --type)")
	_ = cmd.MarkFlagRequired("domain")

	return cmd
}
