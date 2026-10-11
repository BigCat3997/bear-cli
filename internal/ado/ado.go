package ado

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-resty/resty/v2"
)

type AzureDevOpsCred struct {
	Org     string `json:"org"`
	PAT     string `json:"pat"`
	Project string `json:"project"`
}

type AzureDevOpsClient struct {
	AzureDevOpsCred
	RestyClient *resty.Client
}

// DoRequest performs an HTTP request to the Azure DevOps API with basic authentication.
// Username is left empty since PAT is used as password for authentication.
func (c *AzureDevOpsClient) DoRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req := c.RestyClient.R().
		SetContext(ctx).
		SetBasicAuth("", c.PAT).
		SetHeader("Content-Type", "application/json").
		SetDoNotParseResponse(true)

	if body != nil {
		payload, err := io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		req.SetBody(payload)
	}

	resp, err := req.Execute(method, url)
	if err != nil {
		return nil, err
	}

	if resp.RawResponse == nil {
		return nil, fmt.Errorf("empty HTTP response")
	}

	return resp.RawResponse, nil
}

func decodeHTTPError(resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("empty HTTP response")
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	return fmt.Errorf("HTTP %s: %s", resp.Status, message)
}

func NewAzureDevOpsClient(org, project, pat string) *AzureDevOpsClient {
	return &AzureDevOpsClient{
		AzureDevOpsCred: AzureDevOpsCred{
			Org:     org,
			PAT:     pat,
			Project: project,
		},
		RestyClient: resty.New(),
	}
}

func LoadBootstrapConfigPath() (string, error) {
	credPath, exist := os.LookupEnv("BEAR_ADO_CREDPATH")

	if !exist {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "bear", "ado", "credentials.json"), nil
	}
	return credPath, nil
}

func Bootstrap(org, pat string) error {
	if org == "" || pat == "" {
		return fmt.Errorf("org and pat are required")
	}

	configPath, err := LoadBootstrapConfigPath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(AzureDevOpsCred{Org: org, PAT: pat, Project: ""}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	fmt.Printf("Azure DevOps credentials saved to %s\n", configPath)
	return nil
}
