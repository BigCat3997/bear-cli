package config

import "testing"

func baseConfig() Config {
	cfg := Default()
	cfg.DefaultProfile = "dev"
	cfg.Profiles["dev"] = Profile{
		AWSRegion:       "us-west-2",
		AWSProfile:      "dev-aws",
		ADOOrganization: "dev-org",
		ADOProject:      "dev-proj",
	}
	cfg.Profiles["prod"] = Profile{AWSRegion: "eu-west-1", AWSProfile: "prod-aws"}
	return cfg
}

func TestResolveProfileSelection(t *testing.T) {
	tests := []struct {
		name      string
		overrides Overrides
		env       map[string]string
		want      string
	}{
		{"default profile from file", Overrides{}, nil, "dev"},
		{"env beats file", Overrides{}, map[string]string{"BEAR_PROFILE": "prod"}, "prod"},
		{"flag beats env", Overrides{Profile: strPtr("dev")}, map[string]string{"BEAR_PROFILE": "prod"}, "dev"},
		{"env value is trimmed", Overrides{}, map[string]string{"BEAR_PROFILE": "  prod  "}, "prod"},
		{"flag value is trimmed", Overrides{Profile: strPtr(" prod ")}, nil, "prod"},
		{"empty env clears default", Overrides{}, map[string]string{"BEAR_PROFILE": ""}, ""},
		{"empty flag clears env", Overrides{Profile: strPtr("")}, map[string]string{"BEAR_PROFILE": "prod"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(baseConfig(), tt.overrides, envFrom(tt.env))
			if got.SelectedProfile != tt.want {
				t.Fatalf("SelectedProfile = %q, want %q", got.SelectedProfile, tt.want)
			}
		})
	}
}

func TestResolveLoadsActiveProfile(t *testing.T) {
	got := Resolve(baseConfig(), Overrides{}, envFrom(nil))
	want := Profile{AWSRegion: "us-west-2", AWSProfile: "dev-aws", ADOOrganization: "dev-org", ADOProject: "dev-proj"}
	if got.ActiveProfile != want {
		t.Fatalf("ActiveProfile = %+v, want %+v", got.ActiveProfile, want)
	}
}

func TestResolveUnknownProfileLeavesActiveProfileEmpty(t *testing.T) {
	got := Resolve(baseConfig(), Overrides{Profile: strPtr("ghost")}, envFrom(nil))
	if got.SelectedProfile != "ghost" {
		t.Fatalf("SelectedProfile = %q", got.SelectedProfile)
	}
	if got.ActiveProfile != (Profile{}) {
		t.Fatalf("ActiveProfile = %+v, want zero", got.ActiveProfile)
	}
	// Validation is what rejects the unknown profile.
	if err := got.Validate(); err == nil {
		t.Fatal("Validate should reject a selected profile that does not exist")
	}
}

func TestResolveDoesNotMutateInputProfiles(t *testing.T) {
	cfg := baseConfig()
	Resolve(cfg, Overrides{AWSRegion: strPtr("ap-south-1")}, envFrom(map[string]string{"AWS_PROFILE": "x"}))
	if cfg.Profiles["dev"].AWSRegion != "us-west-2" || cfg.Profiles["dev"].AWSProfile != "dev-aws" {
		t.Fatalf("Resolve mutated the profiles map: %+v", cfg.Profiles["dev"])
	}
}

func TestResolveEnvironmentOverridesProfile(t *testing.T) {
	env := map[string]string{
		"BEAR_OUTPUT_FORMAT":    "json",
		"BEAR_COLOR":            "never",
		"AWS_REGION":            "ap-southeast-1",
		"AWS_PROFILE":           "env-aws",
		"BEAR_ADO_ORGANIZATION": "env-org",
		"BEAR_ADO_PROJECT":      "env-proj",
		"BEAR_GODADDY_BASE_URL": "https://env.example.com/",
	}
	got := Resolve(baseConfig(), Overrides{}, envFrom(env))

	if got.Output.Format != "json" || got.Output.Color != "never" {
		t.Errorf("output = %+v", got.Output)
	}
	want := Profile{AWSRegion: "ap-southeast-1", AWSProfile: "env-aws", ADOOrganization: "env-org", ADOProject: "env-proj"}
	if got.ActiveProfile != want {
		t.Errorf("ActiveProfile = %+v, want %+v", got.ActiveProfile, want)
	}
	if got.GoDaddy.BaseURL != "https://env.example.com" {
		t.Errorf("GoDaddy.BaseURL = %q (trailing slash should be trimmed)", got.GoDaddy.BaseURL)
	}
}

func TestResolveFlagsOverrideEnvironment(t *testing.T) {
	env := map[string]string{
		"BEAR_OUTPUT_FORMAT":    "json",
		"BEAR_COLOR":            "never",
		"AWS_REGION":            "env-region",
		"AWS_PROFILE":           "env-aws",
		"BEAR_ADO_ORGANIZATION": "env-org",
		"BEAR_ADO_PROJECT":      "env-proj",
		"BEAR_GODADDY_BASE_URL": "https://env.example.com",
	}
	overrides := Overrides{
		OutputFormat:    strPtr("env"),
		ColorMode:       strPtr("always"),
		AWSRegion:       strPtr("flag-region"),
		AWSProfile:      strPtr("flag-aws"),
		ADOOrganization: strPtr("flag-org"),
		ADOProject:      strPtr("flag-proj"),
		GoDaddyBaseURL:  strPtr("https://flag.example.com/"),
	}
	got := Resolve(baseConfig(), overrides, envFrom(env))

	if got.Output.Format != "env" || got.Output.Color != "always" {
		t.Errorf("output = %+v", got.Output)
	}
	want := Profile{AWSRegion: "flag-region", AWSProfile: "flag-aws", ADOOrganization: "flag-org", ADOProject: "flag-proj"}
	if got.ActiveProfile != want {
		t.Errorf("ActiveProfile = %+v, want %+v", got.ActiveProfile, want)
	}
	if got.GoDaddy.BaseURL != "https://flag.example.com" {
		t.Errorf("GoDaddy.BaseURL = %q", got.GoDaddy.BaseURL)
	}
}

func TestResolveExplicitEmptyFlagOverridesValue(t *testing.T) {
	got := Resolve(baseConfig(), Overrides{AWSRegion: strPtr("")}, envFrom(nil))
	if got.ActiveProfile.AWSRegion != "" {
		t.Fatalf("AWSRegion = %q, want empty (explicit empty flag wins)", got.ActiveProfile.AWSRegion)
	}
}

func TestResolveAWSRegionEnvironment(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"neither set keeps profile", nil, "us-west-2"},
		{"AWS_REGION", map[string]string{"AWS_REGION": "r1"}, "r1"},
		{"AWS_DEFAULT_REGION fallback", map[string]string{"AWS_DEFAULT_REGION": "r2"}, "r2"},
		{"AWS_REGION beats AWS_DEFAULT_REGION", map[string]string{"AWS_REGION": "r1", "AWS_DEFAULT_REGION": "r2"}, "r1"},
		{"set-but-empty AWS_REGION blocks AWS_DEFAULT_REGION", map[string]string{"AWS_REGION": "", "AWS_DEFAULT_REGION": "r2"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(baseConfig(), Overrides{}, envFrom(tt.env))
			if got.ActiveProfile.AWSRegion != tt.want {
				t.Fatalf("AWSRegion = %q, want %q", got.ActiveProfile.AWSRegion, tt.want)
			}
		})
	}
}

func TestResolveNormalizes(t *testing.T) {
	cfg := baseConfig()
	cfg.DefaultProfile = "  dev  "
	cfg.Jenkins = JenkinsConfig{URL: "  http://j:8080//  ", UserID: " u ", APIToken: " t "}

	got := Resolve(cfg, Overrides{
		OutputFormat:    strPtr("  JSON "),
		ColorMode:       strPtr(" Auto "),
		AWSRegion:       strPtr(" us-east-2 "),
		ADOOrganization: strPtr(" org "),
		GoDaddyBaseURL:  strPtr(" https://x.test/// "),
	}, envFrom(nil))

	if got.DefaultProfile != "dev" || got.SelectedProfile != "dev" {
		t.Errorf("profile names not trimmed: %q / %q", got.DefaultProfile, got.SelectedProfile)
	}
	if got.Output.Format != "json" || got.Output.Color != "auto" {
		t.Errorf("output not lowercased/trimmed: %+v", got.Output)
	}
	if got.ActiveProfile.AWSRegion != "us-east-2" || got.ActiveProfile.ADOOrganization != "org" {
		t.Errorf("active profile not trimmed: %+v", got.ActiveProfile)
	}
	if got.GoDaddy.BaseURL != "https://x.test" {
		t.Errorf("GoDaddy.BaseURL = %q", got.GoDaddy.BaseURL)
	}
	if got.Jenkins != (JenkinsConfig{URL: "http://j:8080", UserID: "u", APIToken: "t"}) {
		t.Errorf("Jenkins = %+v", got.Jenkins)
	}
}

func TestResolveJenkinsEnvironment(t *testing.T) {
	cfg := baseConfig()
	cfg.Jenkins = JenkinsConfig{URL: "http://file", UserID: "file", APIToken: "file"}

	got := Resolve(cfg, Overrides{}, envFrom(map[string]string{
		"BEAR_JENKINS_URL":       "http://env/",
		"BEAR_JENKINS_USER_ID":   "  ",
		"BEAR_JENKINS_API_TOKEN": "env-token",
	}))
	want := JenkinsConfig{URL: "http://env", UserID: "file", APIToken: "env-token"}
	if got.Jenkins != want {
		t.Fatalf("Jenkins = %+v, want %+v", got.Jenkins, want)
	}
}

func TestResolveNilLookupUsesProcessEnvironment(t *testing.T) {
	t.Setenv("BEAR_OUTPUT_FORMAT", "json")
	got := Resolve(baseConfig(), Overrides{}, nil)
	if got.Output.Format != "json" {
		t.Fatalf("Format = %q, want json from os environment", got.Output.Format)
	}
}

func TestSetFromNonEmptyEnvironment(t *testing.T) {
	target := "keep"
	setFromNonEmptyEnvironment(&target, "K", envFrom(map[string]string{"K": ""}))
	if target != "keep" {
		t.Fatalf("empty value must not overwrite, got %q", target)
	}
	setFromNonEmptyEnvironment(&target, "K", envFrom(map[string]string{"K": " new "}))
	if target != "new" {
		t.Fatalf("target = %q, want new", target)
	}
}
