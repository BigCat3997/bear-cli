package godaddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type Client struct {
	APIKey     string
	APISecret  string
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(apiKey, apiSecret, baseURL string) *Client {
	trimmedBaseURL := strings.TrimSpace(baseURL)
	if trimmedBaseURL == "" {
		trimmedBaseURL = "https://api.godaddy.com"
	}

	return &Client{
		APIKey:     strings.TrimSpace(apiKey),
		APISecret:  strings.TrimSpace(apiSecret),
		BaseURL:    strings.TrimRight(trimmedBaseURL, "/"),
		HTTPClient: &http.Client{},
	}
}

func (c *Client) ListDomains(ctx context.Context, limit, offset int) ([]map[string]any, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}

	endpoint := "/v1/domains"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	body, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var domains []map[string]any
	if err := json.Unmarshal(body, &domains); err != nil {
		return nil, fmt.Errorf("decode list domains response: %w", err)
	}

	return domains, nil
}

func (c *Client) ListNameServers(ctx context.Context, domain string) ([]string, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	endpoint := path.Join("/v1/domains", url.PathEscape(domain))
	body, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var details struct {
		NameServers []string `json:"nameServers"`
	}
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("decode domain details response: %w", err)
	}

	return details.NameServers, nil
}

func (c *Client) ListRecords(ctx context.Context, domain, recordType, recordName string) ([]map[string]any, error) {
	domain = strings.TrimSpace(domain)
	recordType = strings.TrimSpace(recordType)
	recordName = strings.TrimSpace(recordName)

	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	endpoint := path.Join("/v1/domains", url.PathEscape(domain), "records")
	if recordType != "" {
		endpoint = path.Join(endpoint, url.PathEscape(recordType))
	}
	if recordName != "" {
		if recordType == "" {
			return nil, fmt.Errorf("record type is required when record name is set")
		}
		endpoint = path.Join(endpoint, url.PathEscape(recordName))
	}

	body, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var records []map[string]any
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("decode list records response: %w", err)
	}

	return records, nil
}

func (c *Client) ChangeNameServers(ctx context.Context, domain string, nameServers []string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	if len(nameServers) == 0 {
		return fmt.Errorf("at least one nameserver is required")
	}

	for i := range nameServers {
		nameServers[i] = strings.TrimSpace(nameServers[i])
		if nameServers[i] == "" {
			return fmt.Errorf("nameserver at index %d is empty", i)
		}
	}

	endpoint := path.Join("/v1/domains", url.PathEscape(domain))
	payload := map[string]any{
		"nameServers": nameServers,
	}

	_, err := c.doRequest(ctx, http.MethodPatch, endpoint, payload)
	return err
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, payload any) ([]byte, error) {
	if c.APIKey == "" || c.APISecret == "" {
		return nil, fmt.Errorf("api key and api secret are required")
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request payload: %w", err)
		}
		body = bytes.NewReader(data)
	}

	requestURL := c.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "sso-key "+c.APIKey+":"+c.APISecret)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("godaddy api error (%d): %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return responseBody, nil
}
