package models

import "time"

type CredentialScope string

const (
	ScopeFull      CredentialScope = "full"
	ScopeTerraform CredentialScope = "terraform"
)

func ParseCredentialScope(s string) CredentialScope {
	switch s {
	case "terraform":
		return ScopeTerraform
	default:
		return ScopeFull
	}
}

type SandboxCredential interface {
	Provider() string
	IsExpired() bool
	ExpiresAt() time.Time
	ToEnvMap() map[string]string
	ToTerraformEnvMap() map[string]string
	ToScopedEnvMap(scope CredentialScope) map[string]string
}

type PsAwsCredential struct {
	SandboxCredential
	AWSCredential

	SandboxURL string `json:"sandboxUrl"`
	User       string `json:"user"`
	Password   string `json:"password"`
}

func (a *PsAwsCredential) Provider() string {
	return "aws"
}

func (c *PsAwsCredential) ToEnvMap() map[string]string {
	return map[string]string{
		"AWS_ACCESS_KEY_ID":     c.AccessKeyId,
		"AWS_SECRET_ACCESS_KEY": c.SecretAccessKey,
		"AWS_REGION":            c.Region,
		"AWS_SANDBOX_URL":       c.SandboxURL,
		"AWS_USERNAME":          c.User,
		"AWS_PASSWORD":          c.Password,
	}
}

func (c *PsAwsCredential) ToTerraformEnvMap() map[string]string {
	return map[string]string{
		"AWS_ACCESS_KEY_ID":     c.AccessKeyId,
		"AWS_SECRET_ACCESS_KEY": c.SecretAccessKey,
		"AWS_REGION":            c.Region,
	}
}

func (c *PsAwsCredential) ToScopedEnvMap(scope CredentialScope) map[string]string {
	if scope == ScopeTerraform {
		return c.ToTerraformEnvMap()
	}
	return c.ToEnvMap()
}

type PsAzureCredential struct {
	SandboxCredential
	ARMCredential

	SandboxURL                    string `json:"sandboxUrl"`
	TenantName                    string `json:"tenantName"`
	User                          string `json:"user"`
	Password                      string `json:"password"`
	ResourceGroup                 string `json:"resourceGroup"`
	ResourceProviderRegistrations string `json:"resourceProviderRegistrations"`
}

func (a *PsAzureCredential) Provider() string {
	return "azure"
}

func (c *PsAzureCredential) ToEnvMap() map[string]string {
	return map[string]string{
		"ARM_SUBSCRIPTION_ID":                 c.SubscriptionID,
		"ARM_TENANT_ID":                       c.TenantID,
		"ARM_CLIENT_ID":                       c.ClientID,
		"ARM_CLIENT_SECRET":                   c.ClientSecret,
		"ARM_SANDBOX_URL":                     c.SandboxURL,
		"ARM_TENANT_NAME":                     c.TenantName,
		"ARM_USERNAME":                        c.User,
		"ARM_PASSWORD":                        c.Password,
		"ARM_RESOURCE_GROUP":                  c.ResourceGroup,
		"ARM_RESOURCE_PROVIDER_REGISTRATIONS": c.ResourceProviderRegistrations,
	}
}

func (c *PsAzureCredential) ToTerraformEnvMap() map[string]string {
	return map[string]string{
		"ARM_SUBSCRIPTION_ID":                 c.SubscriptionID,
		"ARM_TENANT_ID":                       c.TenantID,
		"ARM_CLIENT_ID":                       c.ClientID,
		"ARM_CLIENT_SECRET":                   c.ClientSecret,
		"ARM_RESOURCE_PROVIDER_REGISTRATIONS": c.ResourceProviderRegistrations,
	}
}

func (c *PsAzureCredential) ToScopedEnvMap(scope CredentialScope) map[string]string {
	if scope == ScopeTerraform {
		return c.ToTerraformEnvMap()
	}
	return c.ToEnvMap()
}
