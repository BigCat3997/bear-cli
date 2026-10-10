package ado

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type projectListResponse struct {
	Count int       `json:"count"`
	Value []Project `json:"value"`
}

func (c *AzureDevOpsClient) ListProjects(ctx context.Context) (*http.Response, error) {
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/projects?api-version=7.0", c.Org)
	return c.DoRequest(ctx, "GET", url, nil)
}

func (c *AzureDevOpsClient) ListProjectsData(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeHTTPError(resp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode projects: %w", err)
	}

	return result, nil
}

func (c *AzureDevOpsClient) FindProjectByName(ctx context.Context, name string) (*Project, error) {
	resp, err := c.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeHTTPError(resp)
	}

	var result projectListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode projects: %w", err)
	}

	for i := range result.Value {
		if result.Value[i].Name == name {
			return &result.Value[i], nil
		}
	}

	return nil, fmt.Errorf("project %q not found", name)
}
