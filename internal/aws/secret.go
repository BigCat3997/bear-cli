package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"
)

// CreateSecretInput contains the parameters for creating a secret
type CreateSecretInput struct {
	Name        string
	SecretValue string
	Description string
	Tags        []string
	Override    bool
	Region      string
	Profile     string
}

// CreateSecretResult contains the result of creating a secret
type CreateSecretResult struct {
	Name      string
	ARN       string
	VersionID string
}

// GetSecretInput contains the parameters for retrieving a secret
type GetSecretInput struct {
	Name    string
	Region  string
	Profile string
}

// GetSecretResult contains the result of retrieving a secret
type GetSecretResult struct {
	Name        string
	SecretValue string
	ARN         string
	VersionID   string
}

// DeleteSecretInput contains the parameters for deleting a secret
type DeleteSecretInput struct {
	Name         string
	RecoveryDays int64
	ForceDelete  bool
	Region       string
	Profile      string
}

// DeleteSecretResult contains the result of deleting a secret
type DeleteSecretResult struct {
	Name         string
	ARN          string
	DeletionDate string
}

// CreateSecret creates a new secret in AWS Secrets Manager
func CreateSecret(ctx context.Context, input CreateSecretInput) (*CreateSecretResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("secret name is required")
	}
	if strings.TrimSpace(input.SecretValue) == "" {
		return nil, errors.New("secret value is required")
	}

	client, err := newSecretsManagerClient(ctx, input.Region, input.Profile)
	if err != nil {
		return nil, fmt.Errorf("create secrets manager client: %w", err)
	}

	// Parse tags if provided
	var secretTags []types.Tag
	for _, tag := range input.Tags {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) == 2 {
			secretTags = append(secretTags, types.Tag{
				Key:   aws.String(parts[0]),
				Value: aws.String(parts[1]),
			})
		}
	}

	createSecretInput := &secretsmanager.CreateSecretInput{
		Name:         aws.String(input.Name),
		SecretString: aws.String(input.SecretValue),
	}

	if input.Description != "" {
		createSecretInput.Description = aws.String(input.Description)
	}

	if len(secretTags) > 0 {
		createSecretInput.Tags = secretTags
	}

	output, err := client.CreateSecret(ctx, createSecretInput)
	if err != nil {
		var oe *smithy.OperationError
		if errors.As(err, &oe) {
			if strings.Contains(oe.Error(), "ResourceExistsException") {
				if !input.Override {
					return nil, fmt.Errorf("secret %q already exists (use --override to update value)", input.Name)
				}

				updateOutput, updateErr := client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
					SecretId:     aws.String(input.Name),
					SecretString: aws.String(input.SecretValue),
				})
				if updateErr != nil {
					return nil, fmt.Errorf("override existing secret value: %w", updateErr)
				}

				return &CreateSecretResult{
					Name:      input.Name,
					ARN:       aws.ToString(updateOutput.ARN),
					VersionID: aws.ToString(updateOutput.VersionId),
				}, nil
			}
		}
		return nil, fmt.Errorf("create secret: %w", err)
	}

	result := &CreateSecretResult{
		Name:      aws.ToString(output.Name),
		ARN:       aws.ToString(output.ARN),
		VersionID: aws.ToString(output.VersionId),
	}

	return result, nil
}

// GetSecret retrieves a secret value from AWS Secrets Manager
func GetSecret(ctx context.Context, input GetSecretInput) (*GetSecretResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("secret name is required")
	}

	client, err := newSecretsManagerClient(ctx, input.Region, input.Profile)
	if err != nil {
		return nil, fmt.Errorf("create secrets manager client: %w", err)
	}

	output, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(input.Name),
	})
	if err != nil {
		var oe *smithy.OperationError
		if errors.As(err, &oe) {
			if strings.Contains(oe.Error(), "ResourceNotFoundException") {
				return nil, fmt.Errorf("secret %q not found", input.Name)
			}
		}
		return nil, fmt.Errorf("get secret: %w", err)
	}

	secretValue := ""
	if output.SecretString != nil {
		secretValue = aws.ToString(output.SecretString)
	}

	result := &GetSecretResult{
		Name:        aws.ToString(output.Name),
		SecretValue: secretValue,
		ARN:         aws.ToString(output.ARN),
		VersionID:   aws.ToString(output.VersionId),
	}

	return result, nil
}

// DeleteSecret deletes a secret from AWS Secrets Manager
func DeleteSecret(ctx context.Context, input DeleteSecretInput) (*DeleteSecretResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("secret name is required")
	}

	client, err := newSecretsManagerClient(ctx, input.Region, input.Profile)
	if err != nil {
		return nil, fmt.Errorf("create secrets manager client: %w", err)
	}

	deleteInput := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(input.Name),
	}

	// If force delete is set, delete immediately (RecoveryWindowInDays = 0)
	if input.ForceDelete {
		deleteInput.ForceDeleteWithoutRecovery = aws.Bool(true)
	} else if input.RecoveryDays > 0 {
		deleteInput.RecoveryWindowInDays = aws.Int64(input.RecoveryDays)
	}

	output, err := client.DeleteSecret(ctx, deleteInput)
	if err != nil {
		var oe *smithy.OperationError
		if errors.As(err, &oe) {
			if strings.Contains(oe.Error(), "ResourceNotFoundException") {
				return nil, fmt.Errorf("secret %q not found", input.Name)
			}
		}
		return nil, fmt.Errorf("delete secret: %w", err)
	}

	deletionDate := ""
	if output.DeletionDate != nil {
		deletionDate = output.DeletionDate.String()
	}

	result := &DeleteSecretResult{
		Name:         aws.ToString(output.Name),
		ARN:          aws.ToString(output.ARN),
		DeletionDate: deletionDate,
	}

	return result, nil
}

// newSecretsManagerClient creates a new Secrets Manager client with optional region and profile
func newSecretsManagerClient(ctx context.Context, region string, profile string) (*secretsmanager.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	if region != "" {
		cfg.Region = region
	}

	if profile != "" {
		cfg2, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
		if err != nil {
			return nil, fmt.Errorf("load aws profile: %w", err)
		}
		cfg = cfg2
	}

	return secretsmanager.NewFromConfig(cfg), nil
}
