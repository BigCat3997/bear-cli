package ps

import (
	"reflect"
	"testing"
)

func TestNormalizeAndValidateCreateCredentialOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    PsCreateCredentialOptions
		wantErr bool
	}{
		{
			name: "normalizes supported values",
			opts: PsCreateCredentialOptions{
				UseClipboard:  true,
				CloudProvider: " AWS ",
				Scope:         " TERRAFORM ",
				Output:        " JSON ",
			},
		},
		{
			name: "html file takes precedence over clipboard",
			opts: PsCreateCredentialOptions{
				UseClipboard:  true,
				FilePath:      " sandbox.html ",
				CloudProvider: "azure",
				Scope:         "full",
				Output:        "env",
			},
		},
		{
			name: "accepts push configuration",
			opts: PsCreateCredentialOptions{
				UseClipboard:      true,
				CloudProvider:     "aws",
				Scope:             "terraform",
				Output:            "none",
				PushVariableGroup: true,
				ADOProject:        "project",
				VariableGroup:     "credentials",
			},
		},
		{
			name:    "rejects unknown provider",
			opts:    PsCreateCredentialOptions{UseClipboard: true, CloudProvider: "gcp", Scope: "full", Output: "env"},
			wantErr: true,
		},
		{
			name:    "rejects unknown scope",
			opts:    PsCreateCredentialOptions{UseClipboard: true, CloudProvider: "aws", Scope: "tf", Output: "env"},
			wantErr: true,
		},
		{
			name:    "rejects unknown output",
			opts:    PsCreateCredentialOptions{UseClipboard: true, CloudProvider: "aws", Scope: "full", Output: "yaml"},
			wantErr: true,
		},
		{
			name:    "requires an input source",
			opts:    PsCreateCredentialOptions{CloudProvider: "aws", Scope: "full", Output: "env"},
			wantErr: true,
		},
		{
			name: "push requires terraform scope",
			opts: PsCreateCredentialOptions{
				UseClipboard: true, CloudProvider: "aws", Scope: "full", Output: "none",
				PushVariableGroup: true, ADOProject: "project", VariableGroup: "credentials",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := prepareAndValidateCreateCredentialOptions(&tt.opts, createCredentialFlagState{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.name == "html file takes precedence over clipboard" && tt.opts.UseClipboard {
				t.Fatal("expected --html-path to disable clipboard input")
			}
		})
	}
}

func TestPublishToADODefaults(t *testing.T) {
	opts := PsCreateCredentialOptions{
		UseClipboard:  true,
		CloudProvider: "aws",
		Scope:         "full",
		Output:        "env",
		PublishToADO:  " <ADO_PROJECT> ",
		VariableGroup: defaultTerraformVariableGroup,
	}

	err := prepareAndValidateCreateCredentialOptions(&opts, createCredentialFlagState{publishToADO: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.PushVariableGroup {
		t.Fatal("expected publishing to be enabled")
	}
	if opts.ADOProject != "<ADO_PROJECT>" {
		t.Fatalf("ADO project = %q", opts.ADOProject)
	}
	if opts.Scope != "terraform" {
		t.Fatalf("scope = %q, want terraform", opts.Scope)
	}
	if opts.Output != "none" {
		t.Fatalf("output = %q, want none", opts.Output)
	}
}

func TestPublishToADORespectsExplicitScope(t *testing.T) {
	opts := PsCreateCredentialOptions{
		UseClipboard:  true,
		CloudProvider: "aws",
		Scope:         "full",
		Output:        "env",
		PublishToADO:  "project",
		VariableGroup: defaultTerraformVariableGroup,
	}

	err := prepareAndValidateCreateCredentialOptions(&opts, createCredentialFlagState{
		publishToADO: true,
		scope:        true,
	})
	if err == nil {
		t.Fatal("expected explicitly incompatible scope to be rejected")
	}
}

func TestPublishToJenkinsDefaults(t *testing.T) {
	opts := PsCreateCredentialOptions{
		UseClipboard:     true,
		CloudProvider:    " AWS ",
		Scope:            "full",
		Output:           "env",
		PublishToJenkins: " http://localhost:8080/ ",
		JenkinsUser:      " admin ",
		JenkinsAPIToken:  "token",
		JenkinsScope:     "global",
	}

	err := prepareAndValidateCreateCredentialOptions(&opts, createCredentialFlagState{publishToJenkins: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.PublishToJenkins != "http://localhost:8080" {
		t.Fatalf("Jenkins URL = %q", opts.PublishToJenkins)
	}
	if opts.Scope != "terraform" || opts.Output != "none" {
		t.Fatalf("scope/output = %q/%q, want terraform/none", opts.Scope, opts.Output)
	}
	if opts.JenkinsScope != "GLOBAL" || opts.JenkinsPrefix != "" {
		t.Fatalf("Jenkins scope/prefix = %q/%q", opts.JenkinsScope, opts.JenkinsPrefix)
	}
}

func TestPublishToJenkinsRequiresToken(t *testing.T) {
	opts := PsCreateCredentialOptions{
		UseClipboard:     true,
		CloudProvider:    "aws",
		Scope:            "terraform",
		Output:           "none",
		PublishToJenkins: "http://localhost:8080",
		JenkinsUser:      "admin",
		JenkinsScope:     "GLOBAL",
	}

	err := prepareAndValidateCreateCredentialOptions(&opts, createCredentialFlagState{publishToJenkins: true})
	if err == nil {
		t.Fatal("expected missing Jenkins API token to be rejected")
	}
}

func TestPublishToJenkinsRespectsExplicitScope(t *testing.T) {
	opts := PsCreateCredentialOptions{
		UseClipboard:     true,
		CloudProvider:    "aws",
		Scope:            "full",
		Output:           "none",
		PublishToJenkins: "http://localhost:8080",
		JenkinsUser:      "admin",
		JenkinsAPIToken:  "token",
		JenkinsScope:     "GLOBAL",
	}

	err := prepareAndValidateCreateCredentialOptions(&opts, createCredentialFlagState{publishToJenkins: true, scope: true})
	if err == nil {
		t.Fatal("expected explicitly incompatible scope to be rejected")
	}
}

func TestToAdoVariableInputsSortsAndSeparatesSecrets(t *testing.T) {
	normal, secret := toAdoVariableInputs(map[string]string{
		"REGION":            "us-east-1",
		"ACCESS_KEY_ID":     "key",
		"SECRET_ACCESS_KEY": "secret",
	})

	if want := []string{"ACCESS_KEY_ID=key", "REGION=us-east-1"}; !reflect.DeepEqual(normal, want) {
		t.Fatalf("normal variables = %v, want %v", normal, want)
	}
	if want := []string{"SECRET_ACCESS_KEY=secret"}; !reflect.DeepEqual(secret, want) {
		t.Fatalf("secret variables = %v, want %v", secret, want)
	}
}
