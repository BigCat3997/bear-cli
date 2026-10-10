package arm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Retrieves the tenant ID associated with the Azure subscription using the provided ARM token.
func GetTenantId(token string) string {
	tenantID, _ := GetTenantID(context.Background(), token)
	return tenantID
}

// GetTenantID retrieves the tenant ID without panicking on network or response
// errors, which allows callers to present failures normally.
func GetTenantID(ctx context.Context, token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("ARM token is required")
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://management.azure.com/subscriptions?api-version=2025-04-01",
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("create subscriptions request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	subResp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("list Azure subscriptions: %w", err)
	}
	defer subResp.Body.Close()

	subBody, err := io.ReadAll(subResp.Body)
	if err != nil {
		return "", fmt.Errorf("read subscriptions response: %w", err)
	}
	if subResp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("list Azure subscriptions failed: status=%s body=%s", subResp.Status, strings.TrimSpace(string(subBody)))
	}

	var data map[string]any
	if err := json.Unmarshal(subBody, &data); err != nil {
		return "", fmt.Errorf("decode subscriptions response: %w", err)
	}

	if valueArr, ok := data["value"].([]any); ok && len(valueArr) > 0 {
		if firstSub, ok := valueArr[0].(map[string]any); ok {
			if tenantID, ok := firstSub["tenantId"].(string); ok {
				return tenantID, nil
			}
		}
	}
	return "", fmt.Errorf("tenant ID missing from subscriptions response")
}
