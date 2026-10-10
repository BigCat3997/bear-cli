package ado

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type VariableValue struct {
	Value    *string `json:"value,omitempty"`
	IsSecret bool    `json:"isSecret,omitempty"`
}

type ProjectReference struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type VariableGroupProjectReference struct {
	Name             string           `json:"name,omitempty"`
	Description      string           `json:"description,omitempty"`
	ProjectReference ProjectReference `json:"projectReference"`
}

type VariableGroup struct {
	ID                             int                             `json:"id,omitempty"`
	Name                           string                          `json:"name"`
	Description                    string                          `json:"description,omitempty"`
	Type                           string                          `json:"type,omitempty"`
	Variables                      map[string]VariableValue        `json:"variables,omitempty"`
	VariableGroupProjectReferences []VariableGroupProjectReference `json:"variableGroupProjectReferences,omitempty"`
}

type variableGroupListResponse struct {
	Count int             `json:"count"`
	Value []VariableGroup `json:"value"`
}

type UpsertVariableGroupInput struct {
	Project         string
	Group           string
	Description     string
	Variables       []string
	SecretVariables []string
	Override        bool
}

type UpsertVariableGroupResult struct {
	Status string        `json:"status"`
	Group  VariableGroup `json:"group"`
}

func (c *AzureDevOpsClient) ListVariableGroups(ctx context.Context) (*http.Response, error) {
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/distributedtask/variablegroups?api-version=7.0", c.Org, c.Project)
	return c.DoRequest(ctx, "GET", url, nil)
}

func (c *AzureDevOpsClient) ListVariableGroupsData(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.ListVariableGroups(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeHTTPError(resp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode variable groups: %w", err)
	}

	return result, nil
}

func (c *AzureDevOpsClient) GetVariableGroups(ctx context.Context, groupName string) ([]VariableGroup, error) {
	endpoint := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/distributedtask/variablegroups?api-version=7.1", c.Org, c.Project)
	if groupName != "" {
		endpoint = fmt.Sprintf("%s&groupName=%s", endpoint, url.QueryEscape(groupName))
	}

	resp, err := c.DoRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeHTTPError(resp)
	}

	var result variableGroupListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode variable groups: %w", err)
	}

	return result.Value, nil
}

func (c *AzureDevOpsClient) FindVariableGroupByName(ctx context.Context, groupName string) (*VariableGroup, error) {
	groups, err := c.GetVariableGroups(ctx, groupName)
	if err != nil {
		return nil, err
	}

	for i := range groups {
		if groups[i].Name == groupName {
			return &groups[i], nil
		}
	}

	return nil, nil
}

func (c *AzureDevOpsClient) AddVariableGroup(ctx context.Context, group VariableGroup) (*VariableGroup, error) {
	body, err := json.Marshal(group)
	if err != nil {
		return nil, fmt.Errorf("marshal variable group: %w", err)
	}

	endpoint := fmt.Sprintf("https://dev.azure.com/%s/_apis/distributedtask/variablegroups?api-version=7.1", c.Org)
	resp, err := c.DoRequest(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeHTTPError(resp)
	}

	var result VariableGroup
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode created variable group: %w", err)
	}

	return &result, nil
}

func (c *AzureDevOpsClient) UpdateVariableGroup(ctx context.Context, groupID int, group VariableGroup) (*VariableGroup, error) {
	body, err := json.Marshal(group)
	if err != nil {
		return nil, fmt.Errorf("marshal variable group: %w", err)
	}

	endpoint := fmt.Sprintf("https://dev.azure.com/%s/_apis/distributedtask/variablegroups/%d?api-version=7.1", c.Org, groupID)
	resp, err := c.DoRequest(ctx, "PUT", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeHTTPError(resp)
	}

	var result VariableGroup
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode updated variable group: %w", err)
	}

	return &result, nil
}

func (c *AzureDevOpsClient) UpsertVariableGroup(ctx context.Context, input UpsertVariableGroupInput) (*UpsertVariableGroupResult, error) {
	variables, err := parseVariableFlags(input.Variables, input.SecretVariables, input.Override)
	if err != nil {
		return nil, err
	}

	project, err := c.FindProjectByName(ctx, input.Project)
	if err != nil {
		return nil, fmt.Errorf("resolve project: %w", err)
	}

	existing, err := c.FindVariableGroupByName(ctx, input.Group)
	if err != nil {
		return nil, fmt.Errorf("lookup variable group: %w", err)
	}

	projectRefs := []VariableGroupProjectReference{
		{
			Name:        input.Group,
			Description: input.Description,
			ProjectReference: ProjectReference{
				ID:   project.ID,
				Name: project.Name,
			},
		},
	}

	result := &UpsertVariableGroupResult{Status: "created"}

	if existing == nil {
		created, err := c.AddVariableGroup(ctx, VariableGroup{
			Name:                           input.Group,
			Description:                    input.Description,
			Type:                           "Vsts",
			Variables:                      variables,
			VariableGroupProjectReferences: projectRefs,
		})
		if err != nil {
			return nil, fmt.Errorf("create variable group: %w", err)
		}
		result.Group = *created
		return result, nil
	}

	merged, err := mergeVariables(existing.Variables, variables, input.Override)
	if err != nil {
		return nil, fmt.Errorf("merge variables: %w", err)
	}

	description := existing.Description
	if input.Description != "" {
		description = input.Description
	}

	typ := existing.Type
	if typ == "" {
		typ = "Vsts"
	}

	if len(existing.VariableGroupProjectReferences) > 0 {
		projectRefs = existing.VariableGroupProjectReferences
	}

	updated, err := c.UpdateVariableGroup(ctx, existing.ID, VariableGroup{
		Name:                           existing.Name,
		Description:                    description,
		Type:                           typ,
		Variables:                      merged,
		VariableGroupProjectReferences: projectRefs,
	})
	if err != nil {
		return nil, fmt.Errorf("update variable group: %w", err)
	}

	result.Status = "updated"
	result.Group = *updated
	return result, nil
}

func parseVariableFlags(entries []string, secretEntries []string, allowOverride bool) (map[string]VariableValue, error) {
	variables := make(map[string]VariableValue, len(entries)+len(secretEntries))

	for _, entry := range entries {
		if err := addParsedVariable(variables, entry, false, allowOverride, "--variable"); err != nil {
			return nil, err
		}
	}

	for _, entry := range secretEntries {
		if err := addParsedVariable(variables, entry, true, allowOverride, "--secret-variable"); err != nil {
			return nil, err
		}
	}

	return variables, nil
}

func addParsedVariable(variables map[string]VariableValue, entry string, isSecret bool, allowOverride bool, flagName string) error {
	parts := strings.SplitN(entry, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid %s value %q, expected key=value", flagName, entry)
	}

	key := strings.TrimSpace(parts[0])
	if key == "" {
		return fmt.Errorf("invalid %s value %q, key cannot be empty", flagName, entry)
	}

	value := parts[1]
	if _, exists := variables[key]; exists && !allowOverride {
		return fmt.Errorf("variable %q was provided multiple times; rerun with --override to keep the last value", key)
	}

	valueCopy := value
	variables[key] = VariableValue{Value: &valueCopy, IsSecret: isSecret}
	return nil
}

func mergeVariables(existing map[string]VariableValue, incoming map[string]VariableValue, allowOverride bool) (map[string]VariableValue, error) {
	merged := make(map[string]VariableValue, len(existing)+len(incoming))

	for key, value := range existing {
		merged[key] = value
	}

	for key, value := range incoming {
		if _, exists := merged[key]; exists && !allowOverride {
			return nil, fmt.Errorf("variable group already contains variable %q; rerun with --override to replace it", key)
		}
		merged[key] = value
	}

	return merged, nil
}
