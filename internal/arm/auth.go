package arm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func GetARMToken(clientId, clientSecret, tenant string) (string, error) {
	if strings.TrimSpace(clientId) == "" || strings.TrimSpace(clientSecret) == "" || strings.TrimSpace(tenant) == "" {
		return "", fmt.Errorf("client ID, client secret, and tenant are required")
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)

	form := url.Values{}
	form.Set("client_id", clientId)
	form.Set("scope", "https://management.azure.com/.default")
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "client_credentials")

	resp, err := http.Post(
		tokenURL,
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("request ARM token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ARM token response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("ARM token request failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}

	var tokenResp map[string]any
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode ARM token response: %w", err)
	}

	token, ok := tokenResp["access_token"].(string)
	if !ok || strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("access_token missing in ARM token response")
	}

	return token, nil
}

func GetArmToken2(clientId, clientSecret, tenant string) {
	// tenantID := tenant
	// clientID := clientId
	// clientSecret := clientSecret

	cred, err := azidentity.NewClientSecretCredential(
		tenant,
		clientId,
		clientSecret,
		nil,
	)

	if err != nil {
		panic(err)
	}

	token, err := cred.GetToken(
		context.Background(),
		policy.TokenRequestOptions{
			Scopes: []string{
				"https://management.azure.com/.default",
			},
		},
	)

	if err != nil {
		panic(err)
	}

	fmt.Println(token.Token)
}
