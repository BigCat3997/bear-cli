package armapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func GetARMToken(clientID, clientSecret, tenant string) string {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", "https://management.azure.com/.default")
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "client_credentials")

	resp, err := http.Post(
		tokenURL,
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tokenResp map[string]any
	json.Unmarshal(body, &tokenResp)
	token := tokenResp["access_token"].(string)

	return token
}
