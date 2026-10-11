package config

import (
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Output.Format != DefaultOutputFormat || cfg.Output.Color != DefaultColorMode {
		t.Errorf("unexpected output defaults: %+v", cfg.Output)
	}
	if cfg.GoDaddy.BaseURL != DefaultGoDaddyURL {
		t.Errorf("GoDaddy.BaseURL = %q, want %q", cfg.GoDaddy.BaseURL, DefaultGoDaddyURL)
	}
	if cfg.Profiles == nil {
		t.Error("Profiles must be non-nil so callers can write to it")
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default() must be valid, got %v", err)
	}
}

func TestProfileLookup(t *testing.T) {
	cfg := Default()
	cfg.Profiles["dev"] = Profile{AWSRegion: "eu-west-1"}

	got, found := cfg.Profile("dev")
	if !found || got.AWSRegion != "eu-west-1" {
		t.Errorf("Profile(dev) = %+v, %v", got, found)
	}
	if _, found := cfg.Profile("missing"); found {
		t.Error("Profile(missing) should not be found")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // empty means valid
	}{
		{"defaults", func(*Config) {}, ""},
		{"format json", func(c *Config) { c.Output.Format = "json" }, ""},
		{"format env", func(c *Config) { c.Output.Format = "env" }, ""},
		{"format is case insensitive", func(c *Config) { c.Output.Format = "JSON" }, ""},
		{"format invalid", func(c *Config) { c.Output.Format = "yaml" }, "invalid output format"},
		{"format empty", func(c *Config) { c.Output.Format = "" }, "invalid output format"},
		{"color always", func(c *Config) { c.Output.Color = "always" }, ""},
		{"color never", func(c *Config) { c.Output.Color = "never" }, ""},
		{"color case insensitive", func(c *Config) { c.Output.Color = "NEVER" }, ""},
		{"color invalid", func(c *Config) { c.Output.Color = "rainbow" }, "invalid color mode"},
		{"color empty", func(c *Config) { c.Output.Color = "" }, "invalid color mode"},
		{"no selected profile is fine", func(c *Config) { c.SelectedProfile = "" }, ""},
		{"selected profile exists", func(c *Config) {
			c.Profiles["dev"] = Profile{}
			c.SelectedProfile = "dev"
		}, ""},
		{"selected profile missing", func(c *Config) { c.SelectedProfile = "ghost" }, `profile "ghost" does not exist`},
		{"godaddy url empty", func(c *Config) { c.GoDaddy.BaseURL = "" }, "GoDaddy base URL"},
		{"godaddy url blank", func(c *Config) { c.GoDaddy.BaseURL = "   " }, "GoDaddy base URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
