package ratelimit

import (
	"context"
	"errors"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoClient interface {
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}
type DynamoStore struct {
	table  string
	client DynamoClient
}

func NewDynamoStore(table string, client DynamoClient) *DynamoStore {
	return &DynamoStore{table: table, client: client}
}
func (s *DynamoStore) Consume(ctx context.Context, windows []Window) error {
	items := make([]types.TransactWriteItem, 0, len(windows))
	for _, window := range windows {
		items = append(items, types.TransactWriteItem{Update: &types.Update{
			TableName: &s.table, Key: map[string]types.AttributeValue{"windowKey": &types.AttributeValueMemberS{Value: window.Key}},
			UpdateExpression: str("ADD #count :one SET expiresAt = :expires"), ConditionExpression: str("attribute_not_exists(#count) OR #count < :limit"),
			ExpressionAttributeNames: map[string]string{"#count": "count"}, ExpressionAttributeValues: map[string]types.AttributeValue{
				":one": &types.AttributeValueMemberN{Value: "1"}, ":limit": &types.AttributeValueMemberN{Value: strconv.Itoa(window.Limit)}, ":expires": &types.AttributeValueMemberN{Value: strconv.FormatInt(window.ExpiresAt.Unix(), 10)},
			},
		}})
	}
	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: items})
	var canceled *types.TransactionCanceledException
	if errors.As(err, &canceled) {
		for index, reason := range canceled.CancellationReasons {
			if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" && index < len(windows) {
				return &LimitedError{RetryAt: windows[index].RetryAt}
			}
		}
		return ErrLimited
	}
	return err
}
func str(value string) *string { return &value }
