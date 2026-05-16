package godaddy

import (
	godaddysvc "bear_cli/internal/godaddy"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var apiKey string
var apiSecret string
var baseURL string

var GoDaddyCmd = &cobra.Command{
	Use:   "godaddy",
	Short: "GoDaddy API utilities",
	Long:  "Interact with GoDaddy domains and DNS records from the CLI.",
}

func init() {
	GoDaddyCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "GoDaddy API key (required)")
	GoDaddyCmd.PersistentFlags().StringVar(&apiSecret, "api-secret", "", "GoDaddy API secret (required)")
	GoDaddyCmd.PersistentFlags().StringVar(&baseURL, "base-url", "https://api.godaddy.com", "GoDaddy API base URL")

	_ = GoDaddyCmd.MarkPersistentFlagRequired("api-key")
	_ = GoDaddyCmd.MarkPersistentFlagRequired("api-secret")

	GoDaddyCmd.AddCommand(listDomainsCmd())
	GoDaddyCmd.AddCommand(listDNSServersCmd())
	GoDaddyCmd.AddCommand(listRecordsCmd())
	GoDaddyCmd.AddCommand(changeDNSServerCmd())
}

func newClient() *godaddysvc.Client {
	return godaddysvc.NewClient(apiKey, apiSecret, baseURL)
}

func printJSON(v any) error {
	output, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

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
			domains, err := client.ListDomains(context.Background(), opts.Limit, opts.Offset)
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
			nameServers, err := client.ListNameServers(context.Background(), opts.Domain)
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
			records, err := client.ListRecords(context.Background(), opts.Domain, opts.RecordType, opts.RecordName)
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
			if err := client.ChangeNameServers(context.Background(), opts.Domain, opts.NameServers); err != nil {
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
