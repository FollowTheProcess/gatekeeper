package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// DynamoDBClient is an interface wrapping access to DDB.
type DynamoDBClient interface {
	// PutItem puts an item into DDB.
	PutItem(
		ctx context.Context,
		input *dynamodb.PutItemInput,
		options ...func(*dynamodb.Options),
	) (*dynamodb.PutItemOutput, error)
}
