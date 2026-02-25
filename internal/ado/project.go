package ado

import (
	"context"
	"fmt"
	"net/http"
)

func (c *AzureDevOpsClient) ListProjects(ctx context.Context) (*http.Response, error) {
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/projects?api-version=7.0", c.Org)
	return c.DoRequest(ctx, "GET", url, nil)
}
