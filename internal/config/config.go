package config

import (
	"fmt"
	"strings"
)

const (
	DefaultOutputFormat = "table"
	DefaultColorMode    = "auto"
	DefaultAWSRegion    = "us-east-1"
	DefaultGoDaddyURL   = "https://api.godaddy.com"
)

// Config represents config.toml.
//
// SelectedProfile and ActiveProfile are runtime values. They are resolved after
// reading the configuration file and applying environment and flag overrides.
type Config struct {
	DefaultProfile string             `toml:"default_profile"`
	Output         OutputConfig       `toml:"output"`
	Profiles       map[string]Profile `toml:"profiles"`
	GoDaddy        GoDaddyConfig      `toml:"godaddy"`
	Jenkins        JenkinsConfig      `toml:"jenkins"`
	Terraform      TerraformConfig    `toml:"terraform"`
	AWS            AWSConfig          `toml:"aws"`

	SelectedProfile string  `toml:"-"`
	ActiveProfile   Profile `toml:"-"`
}

type OutputConfig struct {
	Format string `toml:"format"`
	Color  string `toml:"color"`
}

type Profile struct {
	AWSRegion       string `toml:"aws_region"`
	AWSProfile      string `toml:"aws_profile"`
	ADOOrganization string `toml:"ado_organization"`
	ADOProject      string `toml:"ado_project"`
}

type GoDaddyConfig struct {
	BaseURL string `toml:"base_url"`
}

type JenkinsConfig struct {
	URL      string `toml:"jenkins_url"`
	UserID   string `toml:"jenkins_user_id"`
	APIToken string `toml:"jenkins_api_token"`
}

// AWSConfig configures AWS-related menu actions.
type AWSConfig struct {
	SecretSets []SecretSet `toml:"secret_sets"`
}

// SecretSet is a group of key files published together to AWS Secrets Manager.
type SecretSet struct {
	Title   string      `toml:"title"`
	Tooltip string      `toml:"tooltip"`
	Secrets []KeySecret `toml:"secrets"`
}

// KeySecret describes one secret. Set PathEnv to read the file location from an
// environment variable, or Path for a fixed location (may start with ~/).
type KeySecret struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	PathEnv     string `toml:"path_env"`
	Path        string `toml:"path"`
}

// TerraformConfig configures Terraform-related commands, such as the working
// directory that "terraform clean" operates on by default.
type TerraformConfig struct {
	Path string `toml:"path"`
}

// Default returns a configuration containing safe built-in defaults.
func Default() Config {
	return Config{
		Output: OutputConfig{
			Format: DefaultOutputFormat,
			Color:  DefaultColorMode,
		},
		Profiles: make(map[string]Profile),
		GoDaddy: GoDaddyConfig{
			BaseURL: DefaultGoDaddyURL,
		},
	}
}

// Profile returns a named profile.
func (c Config) Profile(name string) (Profile, bool) {
	profile, found := c.Profiles[name]
	return profile, found
}

// Validate checks configuration values after all precedence rules have been
// applied.
func (c Config) Validate() error {
	switch strings.ToLower(c.Output.Format) {
	case "json", "table", "env":
	default:
		return fmt.Errorf(
			"invalid output format %q: expected json, table, or env",
			c.Output.Format,
		)
	}

	switch strings.ToLower(c.Output.Color) {
	case "auto", "always", "never":
	default:
		return fmt.Errorf(
			"invalid color mode %q: expected auto, always, or never",
			c.Output.Color,
		)
	}

	if c.SelectedProfile != "" {
		if _, found := c.Profiles[c.SelectedProfile]; !found {
			return fmt.Errorf(
				"profile %q does not exist in the configuration",
				c.SelectedProfile,
			)
		}
	}

	if strings.TrimSpace(c.GoDaddy.BaseURL) == "" {
		return fmt.Errorf("GoDaddy base URL must not be empty")
	}

	return nil
}
