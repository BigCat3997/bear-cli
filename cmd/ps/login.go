package ps

import (
	"bear_cli/internal/browser"
	ps "bear_cli/internal/pluralsight"
	"bear_cli/pkg/prompt"
	"os"

	"github.com/spf13/cobra"
)

func loginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   string(ps.PsLoginByCredential),
		Short: ps.CommandDescriptions[ps.PsLoginByCredential],
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Opening sandbox login...")
			sandboxUrl := os.Getenv("ARM_SANDBOX_URL")
			username := os.Getenv("ARM_USERNAME")
			password := os.Getenv("ARM_PASSWORD")
			var err error
			if username != "" && password != "" {
				browser.LoginInBrowser(username, password, browser.AzurePortal, sandboxUrl)
			} else {
				err = ps.LoginAzurePortalFromSandbox()
			}
			stopSpinner(err)
			return err
		},
	}

	return cmd
}
