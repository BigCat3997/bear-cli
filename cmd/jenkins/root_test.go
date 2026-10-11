package jenkins

import (
	"testing"

	"bear_cli/internal/config"
)

func TestConnectionUsesResolvedJenkinsConfig(t *testing.T) {
	want := config.JenkinsConfig{URL: "http://jenkins.test", UserID: "config-user", APIToken: "config-token"}
	options := connectionOptions{Config: want}
	if got := options.resolved(); got != want {
		t.Fatalf("resolved = %#v, want %#v", got, want)
	}
}

func TestExplicitJenkinsFlagsOverrideResolvedConfig(t *testing.T) {
	options := connectionOptions{Config: config.JenkinsConfig{
		URL: "http://from-config", UserID: "config-user", APIToken: "config-token",
	}, URL: "http://from-flag"}
	if got := options.resolved().URL; got != "http://from-flag" {
		t.Fatalf("URL = %q", got)
	}
}

func TestAPITokenIsNotExposedAsFlagDefault(t *testing.T) {
	cmd := NewCommand(config.JenkinsConfig{APIToken: "do-not-print-this"})
	flag := cmd.PersistentFlags().Lookup("api-token")
	if flag == nil || flag.DefValue != "" {
		t.Fatalf("API token flag default = %v", flag)
	}
}
