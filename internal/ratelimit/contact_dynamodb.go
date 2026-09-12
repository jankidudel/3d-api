package ratelimit

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ContactDynamoStore struct {
	table  string
	client DynamoClient
}

func NewContactDynamoStore(table string, client DynamoClient) *ContactDynamoStore {
	return &ContactDynamoStore{table: table, client: client}
}

func (s *ContactDynamoStore) Increment(ctx context.Context, key string, limit int, expiresAt time.Time) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:           &s.table,
		Key:                 map[string]types.AttributeValue{"windowKey": &types.AttributeValueMemberS{Value: key}},
		UpdateExpression:    str("SET expiresAt = :expires ADD #count :one"),
		ConditionExpression: str("attribute_not_exists(#count) OR #count < :limit"),
		ExpressionAttributeNames: map[string]string{
			"#count": "count",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":expires": &types.AttributeValueMemberN{Value: strconv.FormatInt(expiresAt.Unix(), 10)},
			":one":     &types.AttributeValueMemberN{Value: "1"},
			":limit":   &types.AttributeValueMemberN{Value: strconv.Itoa(limit)},
		},
	})
	if err == nil {
		return nil
	}
	var conditional *types.ConditionalCheckFailedException
	if errors.As(err, &conditional) {
		return ErrLimited
	}
	return err
}
