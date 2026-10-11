package arm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type CreateStorageAccountParams struct {
	SubscriptionID string
	ResourceGroup  string
	AccountName    string
	Location       string
}

type CreateStorageContainerParams struct {
	SubscriptionID string
	ResourceGroup  string
	AccountName    string
	ContainerName  string
}

type storageAccountCreateRequest struct {
	Location   string                      `json:"location"`
	Kind       string                      `json:"kind"`
	SKU        storageAccountSKU           `json:"sku"`
	Properties storageAccountARMProperties `json:"properties"`
}

type storageAccountSKU struct {
	Name string `json:"name"`
}

type storageAccountARMProperties struct {
	AllowBlobPublicAccess bool `json:"allowBlobPublicAccess"`
}

type storageContainerCreateRequest struct {
	Properties storageContainerProperties `json:"properties"`
}

type storageContainerProperties struct {
	PublicAccess string `json:"publicAccess"`
}

type listStorageAccountKeysResponse struct {
	Keys []storageAccountKey `json:"keys"`
}

type storageAccountKey struct {
	KeyName string `json:"keyName"`
	Value   string `json:"value"`
}

func CreateStorageAccount(token string, params CreateStorageAccountParams) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("ARM token is required")
	}
	if strings.TrimSpace(params.SubscriptionID) == "" {
		return fmt.Errorf("subscription ID is required")
	}
	if strings.TrimSpace(params.ResourceGroup) == "" {
		return fmt.Errorf("resource group is required")
	}
	if strings.TrimSpace(params.AccountName) == "" {
		return fmt.Errorf("storage account name is required")
	}
	if strings.TrimSpace(params.Location) == "" {
		return fmt.Errorf("location is required")
	}

	payload := storageAccountCreateRequest{
		Location: params.Location,
		Kind:     "StorageV2",
		SKU: storageAccountSKU{
			Name: "Standard_LRS",
		},
		Properties: storageAccountARMProperties{
			AllowBlobPublicAccess: false,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal storage account payload: %w", err)
	}

	url := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s?api-version=2023-05-01",
		params.SubscriptionID,
		params.ResourceGroup,
		params.AccountName,
	)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create storage account request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("create storage account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusConflict {
		return nil
	}

	responseBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("create storage account failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(responseBody)))
}

func CreateStorageContainer(token string, params CreateStorageContainerParams) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("ARM token is required")
	}
	if strings.TrimSpace(params.SubscriptionID) == "" {
		return fmt.Errorf("subscription ID is required")
	}
	if strings.TrimSpace(params.ResourceGroup) == "" {
		return fmt.Errorf("resource group is required")
	}
	if strings.TrimSpace(params.AccountName) == "" {
		return fmt.Errorf("storage account name is required")
	}
	if strings.TrimSpace(params.ContainerName) == "" {
		return fmt.Errorf("container name is required")
	}

	payload := storageContainerCreateRequest{
		Properties: storageContainerProperties{
			PublicAccess: "None",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal storage container payload: %w", err)
	}

	url := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers/%s?api-version=2023-05-01",
		params.SubscriptionID,
		params.ResourceGroup,
		params.AccountName,
		params.ContainerName,
	)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create storage container request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("create storage container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusConflict {
		return nil
	}

	responseBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("create storage container failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(responseBody)))
}

func GetStorageAccountConnectionString(token string, params CreateStorageContainerParams) (string, error) {
	url := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/listKeys?api-version=2023-05-01",
		params.SubscriptionID,
		params.ResourceGroup,
		params.AccountName,
	)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("list storage account keys request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", fmt.Errorf("list storage account keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("list storage account keys failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read storage account keys response: %w", err)
	}

	var payload listStorageAccountKeysResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return "", fmt.Errorf("unmarshal storage account keys response: %w", err)
	}

	if len(payload.Keys) == 0 || strings.TrimSpace(payload.Keys[0].Value) == "" {
		return "", fmt.Errorf("no storage account keys returned")
	}

	accountKey := payload.Keys[0].Value
	connectionString := fmt.Sprintf(
		"DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net",
		params.AccountName,
		accountKey,
	)

	return connectionString, nil
}
