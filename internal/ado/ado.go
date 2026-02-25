package ado

import (
	"context"
	"io"
	"net/http"
)

type AzureDevOpsClient struct {
	Org        string
	Project    string
	PAT        string
	HTTPClient *http.Client
}

// NewAzureDevOpsClient creates a new AzureDevOpsClient with the given org, project, and PAT.
func NewAzureDevOpsClient(org, project, pat string) *AzureDevOpsClient {
	return &AzureDevOpsClient{
		Org:        org,
		Project:    project,
		PAT:        pat,
		HTTPClient: &http.Client{},
	}
}

func (c *AzureDevOpsClient) DoRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("", c.PAT) // PAT as password, username can be empty
	req.Header.Set("Content-Type", "application/json")
	return c.HTTPClient.Do(req)
}
