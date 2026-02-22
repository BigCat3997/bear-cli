package armapi

import (
	"encoding/json"
	"io"
	"net/http"
)

// Retrieves the tenant ID associated with the Azure subscription using the provided ARM token.
func GetTenantId(token string) string {
	req, _ := http.NewRequest(
		"GET",
		"https://management.azure.com/subscriptions?api-version=2025-04-01",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	subResp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer subResp.Body.Close()

	subBody, _ := io.ReadAll(subResp.Body)

	var data map[string]any
	if err := json.Unmarshal(subBody, &data); err != nil {
		panic(err)
	}

	if valueArr, ok := data["value"].([]any); ok && len(valueArr) > 0 {
		if firstSub, ok := valueArr[0].(map[string]any); ok {
			if tenantID, ok := firstSub["tenantId"].(string); ok {
				return tenantID
			}
		}
	}
	return ""
}
