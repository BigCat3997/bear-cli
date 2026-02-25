package ado

import (
	"context"
	"fmt"
	"net/http"
)

func (c *AzureDevOpsClient) ListVariableGroups(ctx context.Context) (*http.Response, error) {
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/distributedtask/variablegroups?api-version=7.0-preview.2", c.Org, c.Project)
	return c.DoRequest(ctx, "GET", url, nil)
}
