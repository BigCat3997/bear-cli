package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fullConfig = `
default_profile = "sandbox"

[output]
format = "JSON"
color = "never"

[profiles.sandbox]
aws_region = "us-east-1"
aws_profile = "sandbox"
ado_organization = "org"
ado_project = "proj"

[profiles.prod]
aws_region = "eu-west-1"

[godaddy]
base_url = "https://api.godaddy.test/"

[terraform]
path = "/infra"

[[aws.secret_sets]]
title = "Keys"
tooltip = "tip"

[[aws.secret_sets.secrets]]
name = "a"
description = "desc"
path_env = "KEY_PATH"

[[aws.secret_sets.secrets]]
name = "b"
path = "~/keys/b.pem"
`

func TestLoadFullFile(t *testing.T) {
	path := writeConfigFile(t, fullConfig)

	res, err := Load(LoadOptions{Path: path, LookupEnv: envFrom(nil)})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !res.FileFound || res.Path != path {
		t.Errorf("FileFound=%v Path=%q", res.FileFound, res.Path)
	}

	cfg := res.Config
	if cfg.SelectedProfile != "sandbox" || cfg.ActiveProfile.AWSRegion != "us-east-1" || cfg.ActiveProfile.ADOProject != "proj" {
		t.Errorf("profile not resolved: %q %+v", cfg.SelectedProfile, cfg.ActiveProfile)
	}
	if cfg.Output.Format != "json" || cfg.Output.Color != "never" {
		t.Errorf("output = %+v", cfg.Output)
	}
	if cfg.GoDaddy.BaseURL != "https://api.godaddy.test" {
		t.Errorf("GoDaddy.BaseURL = %q", cfg.GoDaddy.BaseURL)
	}
	if cfg.Terraform.Path != "/infra" {
		t.Errorf("Terraform.Path = %q", cfg.Terraform.Path)
	}
	if len(cfg.Profiles) != 2 {
		t.Errorf("Profiles = %v", cfg.Profiles)
	}
	if len(cfg.AWS.SecretSets) != 1 {
		t.Fatalf("SecretSets = %+v", cfg.AWS.SecretSets)
	}
	set := cfg.AWS.SecretSets[0]
	if set.Title != "Keys" || set.Tooltip != "tip" || len(set.Secrets) != 2 {
		t.Fatalf("secret set = %+v", set)
	}
	if set.Secrets[0] != (KeySecret{Name: "a", Description: "desc", PathEnv: "KEY_PATH"}) {
		t.Errorf("secret[0] = %+v", set.Secrets[0])
	}
	if set.Secrets[1] != (KeySecret{Name: "b", Path: "~/keys/b.pem"}) {
		t.Errorf("secret[1] = %+v", set.Secrets[1])
	}
}

func TestLoadEmptyFileUsesDefaults(t *testing.T) {
	path := writeConfigFile(t, "")

	res, err := Load(LoadOptions{Path: path, LookupEnv: envFrom(nil)})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !res.FileFound {
		t.Error("FileFound should be true for an existing empty file")
	}
	if res.Config.Output.Format != DefaultOutputFormat || res.Config.GoDaddy.BaseURL != DefaultGoDaddyURL {
		t.Errorf("defaults not preserved: %+v", res.Config)
	}
	if res.Config.Profiles == nil {
		t.Error("Profiles must be non-nil")
	}
}

func TestLoadFilePartialKeepsOtherDefaults(t *testing.T) {
	path := writeConfigFile(t, "[output]\nformat = \"env\"\n")

	res, err := Load(LoadOptions{Path: path, LookupEnv: envFrom(nil)})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Config.Output.Format != "env" {
		t.Errorf("Format = %q", res.Config.Output.Format)
	}
	if res.Config.Output.Color != DefaultColorMode {
		t.Errorf("Color = %q, want default %q", res.Config.Output.Color, DefaultColorMode)
	}
}

func TestLoadMissingDefaultFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	res, err := Load(LoadOptions{LookupEnv: envFrom(nil)})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.FileFound {
		t.Error("FileFound should be false")
	}
	if filepath.Base(res.Path) != ConfigFilename {
		t.Errorf("Path = %q", res.Path)
	}
	if res.Config.Output.Format != DefaultOutputFormat {
		t.Errorf("expected defaults, got %+v", res.Config.Output)
	}
}

func TestLoadReadsDefaultFileFromWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ConfigFilename), []byte("[output]\nformat = \"json\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	res, err := Load(LoadOptions{LookupEnv: envFrom(nil)})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !res.FileFound || res.Config.Output.Format != "json" {
		t.Fatalf("FileFound=%v Format=%q", res.FileFound, res.Config.Output.Format)
	}
}

func TestLoadMissingExplicitFileIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.toml")

	t.Run("flag path", func(t *testing.T) {
		_, err := Load(LoadOptions{Path: missing, LookupEnv: envFrom(nil)})
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("BEAR_CONFIG", func(t *testing.T) {
		_, err := Load(LoadOptions{LookupEnv: envFrom(map[string]string{"BEAR_CONFIG": missing})})
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("empty BEAR_CONFIG is not explicit", func(t *testing.T) {
		t.Chdir(t.TempDir())
		_, err := Load(LoadOptions{LookupEnv: envFrom(map[string]string{"BEAR_CONFIG": ""})})
		if err != nil {
			t.Fatalf("empty BEAR_CONFIG should fall back to default path, got %v", err)
		}
	})
}

func TestLoadFlagPathBeatsBearConfig(t *testing.T) {
	flagPath := writeConfigFile(t, "[output]\nformat = \"json\"\n")
	envPath := writeConfigFile(t, "[output]\nformat = \"env\"\n")

	res, err := Load(LoadOptions{Path: flagPath, LookupEnv: envFrom(map[string]string{"BEAR_CONFIG": envPath})})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Path != flagPath || res.Config.Output.Format != "json" {
		t.Fatalf("Path=%q Format=%q", res.Path, res.Config.Output.Format)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	path := writeConfigFile(t, "this is = = not toml")

	_, err := Load(LoadOptions{Path: path, LookupEnv: envFrom(nil)})
	if err == nil || !strings.Contains(err.Error(), "decode configuration file") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadWrongTypeInTOML(t *testing.T) {
	path := writeConfigFile(t, "[output]\nformat = 42\n")

	_, err := Load(LoadOptions{Path: path, LookupEnv: envFrom(nil)})
	if err == nil || !strings.Contains(err.Error(), "decode configuration file") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadPathIsDirectory(t *testing.T) {
	_, err := Load(LoadOptions{Path: t.TempDir(), LookupEnv: envFrom(nil)})
	if err == nil || !strings.Contains(err.Error(), "read configuration file") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadValidationFailuresMentionPath(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		env      map[string]string
		wantErr  string
	}{
		{"bad format in file", "[output]\nformat = \"xml\"\n", nil, "invalid output format"},
		{"bad color in file", "[output]\ncolor = \"neon\"\n", nil, "invalid color mode"},
		{"bad format from env", "", map[string]string{"BEAR_OUTPUT_FORMAT": "xml"}, "invalid output format"},
		{"unknown default profile", "default_profile = \"ghost\"\n", nil, `profile "ghost" does not exist`},
		{"unknown profile from env", "", map[string]string{"BEAR_PROFILE": "ghost"}, `profile "ghost" does not exist`},
		{"empty godaddy url in file", "[godaddy]\nbase_url = \"\"\n", nil, "GoDaddy base URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.contents)
			_, err := Load(LoadOptions{Path: path, LookupEnv: envFrom(tt.env)})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("error should mention the file path %q: %v", path, err)
			}
		})
	}
}

func TestLoadAppliesOverridesAndEnvironmentOverFile(t *testing.T) {
	path := writeConfigFile(t, fullConfig)
	env := envFrom(map[string]string{"BEAR_PROFILE": "prod", "BEAR_COLOR": "always"})

	res, err := Load(LoadOptions{
		Path:      path,
		LookupEnv: env,
		Overrides: Overrides{OutputFormat: strPtr("table")},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg := res.Config
	if cfg.SelectedProfile != "prod" || cfg.ActiveProfile.AWSRegion != "eu-west-1" {
		t.Errorf("profile = %q %+v", cfg.SelectedProfile, cfg.ActiveProfile)
	}
	if cfg.Output.Color != "always" {
		t.Errorf("env should beat file for color, got %q", cfg.Output.Color)
	}
	if cfg.Output.Format != "table" {
		t.Errorf("flag should beat file for format, got %q", cfg.Output.Format)
	}
}

func TestLoadExpandsTildeInPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "c.toml"), []byte("[output]\nformat = \"json\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := Load(LoadOptions{Path: "~/c.toml", LookupEnv: envFrom(nil)})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Path != filepath.Join(home, "c.toml") || res.Config.Output.Format != "json" {
		t.Fatalf("Path=%q Format=%q", res.Path, res.Config.Output.Format)
	}
}

func TestPathWasExplicit(t *testing.T) {
	tests := []struct {
		name string
		path string
		env  map[string]string
		want bool
	}{
		{"nothing", "", nil, false},
		{"flag", "x.toml", nil, true},
		{"env", "", map[string]string{"BEAR_CONFIG": "x.toml"}, true},
		{"empty env", "", map[string]string{"BEAR_CONFIG": ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathWasExplicit(tt.path, envFrom(tt.env)); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadNilLookupEnvUsesProcessEnvironment(t *testing.T) {
	path := writeConfigFile(t, "")
	t.Setenv("BEAR_COLOR", "never")

	res, err := Load(LoadOptions{Path: path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Config.Output.Color != "never" {
		t.Fatalf("Color = %q", res.Config.Output.Color)
	}
}
