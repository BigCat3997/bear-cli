package jenkins

import (
	"bear_cli/internal/config"

	"github.com/spf13/cobra"
)

type connectionOptions struct {
	URL      string
	User     string
	APIToken string
	Config   config.JenkinsConfig
}

// NewCommand creates the Jenkins command tree. A constructor is used instead
// of a package-global command so repeated callers do not share mutable flag
// state.
func NewCommand(cfg config.JenkinsConfig) *cobra.Command {
	connection := &connectionOptions{Config: cfg}
	cmd := &cobra.Command{
		Use:   "jenkins",
		Short: "Manage Jenkins credentials and agents",
		Long:  "Manage Jenkins through its HTTP endpoints using API-token authentication.",
	}

	cmd.PersistentFlags().StringVar(&connection.URL, "url", "", "Jenkins controller URL (default: BEAR_JENKINS_URL, then config)")
	cmd.PersistentFlags().StringVar(&connection.User, "user", "", "Jenkins username (default: BEAR_JENKINS_USER_ID, then config)")
	cmd.PersistentFlags().StringVar(&connection.APIToken, "api-token", "", "Jenkins API token (prefer BEAR_JENKINS_API_TOKEN)")

	cmd.AddCommand(newCredentialCommand(connection))
	cmd.AddCommand(newAgentCommand(connection))
	return cmd
}

func (options connectionOptions) resolved() config.JenkinsConfig {
	resolved := options.Config
	if options.URL != "" {
		resolved.URL = options.URL
	}
	if options.User != "" {
		resolved.UserID = options.User
	}
	if options.APIToken != "" {
		resolved.APIToken = options.APIToken
	}
	return resolved
}
