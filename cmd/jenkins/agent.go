package jenkins

import (
	"fmt"

	"bear_cli/internal/config"
	jenkinsservice "bear_cli/internal/jenkins"

	"github.com/spf13/cobra"
)

type sshAgentOptions struct {
	Name                    string
	Description             string
	RemoteFS                string
	Labels                  string
	Host                    string
	CredentialsID           string
	Executors               int
	Mode                    string
	Port                    int
	JavaPath                string
	JVMOptions              string
	PrefixStartAgentCommand string
	SuffixStartAgentCommand string
	LaunchTimeout           int
	MaxRetries              int
	RetryWait               int
	HostKeyStrategy         string
	Override                bool
}

func newAgentCommand(connection *connectionOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "agent", Short: "Manage Jenkins agents"}
	cmd.AddCommand(newCreateSSHAgentCommand(connection))
	return cmd
}

func newCreateSSHAgentCommand(connection *connectionOptions) *cobra.Command {
	opts := &sshAgentOptions{}
	cmd := &cobra.Command{
		Use:   "create-ssh",
		Short: "Create or update a permanent Jenkins SSH agent",
		Long:  "Create a permanent Jenkins agent launched by the SSH Build Agents plugin. The credential ID must already exist in Jenkins as an SSH username credential.",
		Example: `  BEAR_JENKINS_URL=http://localhost:8080 \
  BEAR_JENKINS_USER_ID=admin BEAR_JENKINS_API_TOKEN=... \
  bear jenkins agent create-ssh \
    --name linux-01 --host <AGENT_HOST> \
    --credentials-id linux-agent-key \
    --remote-fs /home/jenkins --labels "linux terraform"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveSSHAgentConnection(connection)
			if err != nil {
				return err
			}
			client := jenkinsservice.NewClient(resolved.URL, resolved.UserID, resolved.APIToken)
			result, err := client.CreateSSHAgent(cmd.Context(), jenkinsservice.SSHAgentInput{
				Name: opts.Name, Description: opts.Description, RemoteFS: opts.RemoteFS, Labels: opts.Labels,
				Host: opts.Host, CredentialsID: opts.CredentialsID, Executors: opts.Executors, Mode: opts.Mode,
				Port: opts.Port, JavaPath: opts.JavaPath, JVMOptions: opts.JVMOptions,
				PrefixStartAgentCommand: opts.PrefixStartAgentCommand, SuffixStartAgentCommand: opts.SuffixStartAgentCommand,
				LaunchTimeoutSeconds: opts.LaunchTimeout, MaxNumRetries: opts.MaxRetries,
				RetryWaitTimeSeconds: opts.RetryWait, HostKeyVerificationStrategy: opts.HostKeyStrategy,
				Override: opts.Override,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Jenkins SSH agent %q %s for host %s\n", result.Name, result.Status, result.Host)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Jenkins agent name")
	cmd.Flags().StringVar(&opts.Host, "host", "", "SSH hostname or IP address")
	cmd.Flags().IntVar(&opts.Port, "ssh-port", 22, "SSH port")
	cmd.Flags().StringVar(&opts.CredentialsID, "credentials-id", "", "Existing Jenkins SSH credential ID")
	cmd.Flags().StringVar(&opts.RemoteFS, "remote-fs", "", "Absolute workspace path on the agent")
	cmd.Flags().IntVar(&opts.Executors, "executors", 1, "Number of concurrent executors")
	cmd.Flags().StringVar(&opts.Labels, "labels", "", "Space-separated Jenkins labels")
	cmd.Flags().StringVar(&opts.Mode, "mode", "NORMAL", "Scheduling mode: NORMAL or EXCLUSIVE")
	cmd.Flags().StringVar(&opts.Description, "description", "Managed by Bear CLI", "Agent description")
	cmd.Flags().StringVar(&opts.HostKeyStrategy, "host-key-strategy", "known-hosts", "SSH host-key strategy: known-hosts, manually-trusted, or non-verifying")
	cmd.Flags().StringVar(&opts.JavaPath, "java-path", "", "Java executable path on the agent (auto-detected when empty)")
	cmd.Flags().StringVar(&opts.JVMOptions, "jvm-options", "", "JVM options used to start the Jenkins agent")
	cmd.Flags().StringVar(&opts.PrefixStartAgentCommand, "prefix-start-agent-command", "", "Command prefix added before the remote agent start command")
	cmd.Flags().StringVar(&opts.SuffixStartAgentCommand, "suffix-start-agent-command", "", "Command suffix added after the remote agent start command")
	cmd.Flags().IntVar(&opts.LaunchTimeout, "launch-timeout", 60, "SSH launch timeout in seconds")
	cmd.Flags().IntVar(&opts.MaxRetries, "max-retries", 10, "Maximum SSH connection retries")
	cmd.Flags().IntVar(&opts.RetryWait, "retry-wait", 15, "Seconds between SSH connection retries")
	cmd.Flags().BoolVar(&opts.Override, "override", false, "Update the agent when its name already exists")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("host")
	_ = cmd.MarkFlagRequired("credentials-id")
	_ = cmd.MarkFlagRequired("remote-fs")
	return cmd
}

// resolveSSHAgentConnection reloads Bear's selected config file when this
// command was constructed without one. Existing values (including flags) keep
// precedence, and only missing Jenkins fields are filled from the file.
func resolveSSHAgentConnection(connection *connectionOptions) (config.JenkinsConfig, error) {
	resolved := connection.resolved()
	if resolved.URL != "" && resolved.UserID != "" && resolved.APIToken != "" {
		return resolved, nil
	}

	loaded, err := config.Load(config.LoadOptions{})
	if err != nil {
		return config.JenkinsConfig{}, fmt.Errorf("load Jenkins config: %w", err)
	}

	fallback := loaded.Config.Jenkins
	if resolved.URL == "" {
		resolved.URL = fallback.URL
	}
	if resolved.UserID == "" {
		resolved.UserID = fallback.UserID
	}
	if resolved.APIToken == "" {
		resolved.APIToken = fallback.APIToken
	}

	return resolved, nil
}
