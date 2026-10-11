package aws

import (
	"context"
	"fmt"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func CreateTerraformLocksTable(
	ctx context.Context,
	tableName string,
) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: awsSdk.String(tableName),

		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: awsSdk.String("LockId"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},

		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: awsSdk.String("LockId"),
				KeyType:       types.KeyTypeHash,
			},
		},

		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		return fmt.Errorf("create dynamodb table: %w", err)
	}
	return nil
}
