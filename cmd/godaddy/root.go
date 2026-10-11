package godaddy

import (
	godaddysvc "bear_cli/internal/godaddy"

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
