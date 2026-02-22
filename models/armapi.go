package models

type ARMCredential struct {
	SubscriptionID string `json:"subscriptionId"`
	TenantID       string `json:"tenantId"`
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`
}
