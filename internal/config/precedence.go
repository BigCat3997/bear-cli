package config

import (
	"os"
	"strings"
)

// EnvLookup makes environment-variable resolution replaceable in tests.
type EnvLookup func(key string) (string, bool)

// Overrides represents values supplied through command-line flags.
//
// Pointer fields distinguish an explicitly supplied empty value from a flag
// that was not supplied.
type Overrides struct {
	Profile *string

	OutputFormat *string
	ColorMode    *string

	AWSRegion       *string
	AWSProfile      *string
	ADOOrganization *string
	ADOProject      *string

	GoDaddyBaseURL *string
}

// Resolve applies environment variables and command-line overrides.
//
// Precedence, from highest to lowest:
//
//	command-line flags
//	environment variables
//	selected TOML profile
//	TOML global values
//	built-in defaults
func Resolve(
	cfg Config,
	overrides Overrides,
	lookupEnv EnvLookup,
) Config {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	selectedProfile := resolveProfileName(cfg, overrides, lookupEnv)

	cfg.SelectedProfile = selectedProfile
	if selectedProfile != "" {
		if profile, found := cfg.Profiles[selectedProfile]; found {
			cfg.ActiveProfile = profile
		}
	}

	applyEnvironment(&cfg, lookupEnv)
	applyOverrides(&cfg, overrides)

	normalize(&cfg)

	return cfg
}

func resolveProfileName(
	cfg Config,
	overrides Overrides,
	lookupEnv EnvLookup,
) string {
	if overrides.Profile != nil {
		return strings.TrimSpace(*overrides.Profile)
	}

	if value, found := lookupEnv("BEAR_PROFILE"); found {
		return strings.TrimSpace(value)
	}

	return strings.TrimSpace(cfg.DefaultProfile)
}

func applyEnvironment(cfg *Config, lookupEnv EnvLookup) {
	setFromEnvironment(
		&cfg.Output.Format,
		"BEAR_OUTPUT_FORMAT",
		lookupEnv,
	)

	setFromEnvironment(
		&cfg.Output.Color,
		"BEAR_COLOR",
		lookupEnv,
	)

	// AWS_REGION is compatible with the AWS SDK and CLI.
	setFromEnvironment(
		&cfg.ActiveProfile.AWSRegion,
		"AWS_REGION",
		lookupEnv,
	)

	// AWS_DEFAULT_REGION is used only when AWS_REGION is not set.
	if _, found := lookupEnv("AWS_REGION"); !found {
		setFromEnvironment(
			&cfg.ActiveProfile.AWSRegion,
			"AWS_DEFAULT_REGION",
			lookupEnv,
		)
	}

	setFromEnvironment(
		&cfg.ActiveProfile.AWSProfile,
		"AWS_PROFILE",
		lookupEnv,
	)

	setFromEnvironment(
		&cfg.ActiveProfile.ADOOrganization,
		"BEAR_ADO_ORGANIZATION",
		lookupEnv,
	)

	setFromEnvironment(
		&cfg.ActiveProfile.ADOProject,
		"BEAR_ADO_PROJECT",
		lookupEnv,
	)

	setFromEnvironment(
		&cfg.GoDaddy.BaseURL,
		"BEAR_GODADDY_BASE_URL",
		lookupEnv,
	)

	// Empty Jenkins environment variables should behave like unset variables so
	// values from config.toml remain available as fallbacks.
	setFromNonEmptyEnvironment(&cfg.Jenkins.URL, "BEAR_JENKINS_URL", lookupEnv)
	setFromNonEmptyEnvironment(&cfg.Jenkins.UserID, "BEAR_JENKINS_USER_ID", lookupEnv)
	setFromNonEmptyEnvironment(&cfg.Jenkins.APIToken, "BEAR_JENKINS_API_TOKEN", lookupEnv)
}

func applyOverrides(cfg *Config, overrides Overrides) {
	setFromOverride(&cfg.Output.Format, overrides.OutputFormat)
	setFromOverride(&cfg.Output.Color, overrides.ColorMode)

	setFromOverride(
		&cfg.ActiveProfile.AWSRegion,
		overrides.AWSRegion,
	)

	setFromOverride(
		&cfg.ActiveProfile.AWSProfile,
		overrides.AWSProfile,
	)

	setFromOverride(
		&cfg.ActiveProfile.ADOOrganization,
		overrides.ADOOrganization,
	)

	setFromOverride(
		&cfg.ActiveProfile.ADOProject,
		overrides.ADOProject,
	)

	setFromOverride(
		&cfg.GoDaddy.BaseURL,
		overrides.GoDaddyBaseURL,
	)
}

func setFromEnvironment(
	target *string,
	key string,
	lookupEnv EnvLookup,
) {
	value, found := lookupEnv(key)
	if !found {
		return
	}

	*target = strings.TrimSpace(value)
}

func setFromNonEmptyEnvironment(
	target *string,
	key string,
	lookupEnv EnvLookup,
) {
	value, found := lookupEnv(key)
	value = strings.TrimSpace(value)
	if !found || value == "" {
		return
	}

	*target = value
}

func setFromOverride(target *string, override *string) {
	if override == nil {
		return
	}

	*target = strings.TrimSpace(*override)
}

func normalize(cfg *Config) {
	cfg.DefaultProfile = strings.TrimSpace(cfg.DefaultProfile)
	cfg.SelectedProfile = strings.TrimSpace(cfg.SelectedProfile)

	cfg.Output.Format = strings.ToLower(
		strings.TrimSpace(cfg.Output.Format),
	)

	cfg.Output.Color = strings.ToLower(
		strings.TrimSpace(cfg.Output.Color),
	)

	cfg.ActiveProfile.AWSRegion = strings.TrimSpace(
		cfg.ActiveProfile.AWSRegion,
	)

	cfg.ActiveProfile.AWSProfile = strings.TrimSpace(
		cfg.ActiveProfile.AWSProfile,
	)

	cfg.ActiveProfile.ADOOrganization = strings.TrimSpace(
		cfg.ActiveProfile.ADOOrganization,
	)

	cfg.ActiveProfile.ADOProject = strings.TrimSpace(
		cfg.ActiveProfile.ADOProject,
	)

	cfg.GoDaddy.BaseURL = strings.TrimRight(
		strings.TrimSpace(cfg.GoDaddy.BaseURL),
		"/",
	)

	cfg.Jenkins.URL = strings.TrimRight(strings.TrimSpace(cfg.Jenkins.URL), "/")
	cfg.Jenkins.UserID = strings.TrimSpace(cfg.Jenkins.UserID)
	cfg.Jenkins.APIToken = strings.TrimSpace(cfg.Jenkins.APIToken)
}
