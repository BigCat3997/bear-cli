package godaddy

import (
	"bear_cli/pkg/prompt"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func printJSON(v any) error {
	output, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

type listDNSServersOptions struct {
	Domain string
}

func listDNSServersCmd() *cobra.Command {
	opts := &listDNSServersOptions{}

	cmd := &cobra.Command{
		Use:     "list-dnsservers",
		Aliases: []string{"list-nameservers"},
		Short:   "List nameservers of a domain",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			stopSpinner := prompt.StartSpinner("Retrieving domain nameservers...")
			nameServers, err := client.ListNameServers(context.Background(), opts.Domain)
			stopSpinner(err)
			if err != nil {
				return err
			}

			return printJSON(map[string]any{
				"domain":      opts.Domain,
				"nameServers": nameServers,
			})
		},
	}

	cmd.Flags().StringVar(&opts.Domain, "domain", "", "Domain name (required)")
	_ = cmd.MarkFlagRequired("domain")

	return cmd
}

func splitCSV(input string) []string {
	parts := strings.Split(input, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}

	return items
}

type changeDNSServerOptions struct {
	Domain      string
	NameServers []string
}

func changeDNSServerCmd() *cobra.Command {
	opts := &changeDNSServerOptions{}

	cmd := &cobra.Command{
		Use:     "change-dnsserver",
		Aliases: []string{"set-nameservers"},
		Short:   "Change nameservers of a domain",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(opts.NameServers) == 1 && strings.Contains(opts.NameServers[0], ",") {
				opts.NameServers = splitCSV(opts.NameServers[0])
			}

			client := newClient()
			stopSpinner := prompt.StartSpinner("Updating domain nameservers...")
			err := client.ChangeNameServers(context.Background(), opts.Domain, opts.NameServers)
			stopSpinner(err)
			if err != nil {
				return err
			}

			return printJSON(map[string]any{
				"domain":      opts.Domain,
				"nameServers": opts.NameServers,
				"updated":     true,
			})
		},
	}

	cmd.Flags().StringVar(&opts.Domain, "domain", "", "Domain name (required)")
	cmd.Flags().StringSliceVar(&opts.NameServers, "nameserver", nil, "Nameserver values (repeat flag or provide CSV)")
	_ = cmd.MarkFlagRequired("domain")
	_ = cmd.MarkFlagRequired("nameserver")

	return cmd
}
