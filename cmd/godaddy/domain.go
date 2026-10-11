package godaddy

import (
	"bear_cli/pkg/prompt"
	"context"

	"github.com/spf13/cobra"
)

type listDomainsOptions struct {
	Limit  int
	Offset int
}

func listDomainsCmd() *cobra.Command {
	opts := &listDomainsOptions{}

	cmd := &cobra.Command{
		Use:   "list-domains",
		Short: "List domains in your GoDaddy account",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			stopSpinner := prompt.StartSpinner("Retrieving GoDaddy domains...")
			domains, err := client.ListDomains(context.Background(), opts.Limit, opts.Offset)
			stopSpinner(err)
			if err != nil {
				return err
			}

			return printJSON(domains)
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "Maximum number of domains to return")
	cmd.Flags().IntVar(&opts.Offset, "offset", 0, "Pagination offset")

	return cmd
}
