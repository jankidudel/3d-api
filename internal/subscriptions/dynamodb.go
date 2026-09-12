package subscriptions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoClient interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

type DynamoStore struct {
	table  string
	client DynamoClient
}

func NewDynamoStore(table string, client DynamoClient) *DynamoStore {
	return &DynamoStore{table: table, client: client}
}

func (s *DynamoStore) Subscribe(ctx context.Context, email string, ipIdentifier string, now time.Time) error {
	emailKey := "email#" + emailIdentifier(email)
	existing, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      &s.table,
		Key:            map[string]types.AttributeValue{"subscriberKey": &types.AttributeValueMemberS{Value: emailKey}},
		ConsistentRead: boolp(true),
	})
	if err != nil {
		return err
	}
	if existing.Item != nil {
		return ErrAlreadySubscribed
	}

	day := now.Format("20060102")
	expiresAt := now.Add(48 * time.Hour)
	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{
				TableName:           &s.table,
				Item:                subscriberItem(emailKey, email, now),
				ConditionExpression: str("attribute_not_exists(subscriberKey)"),
			}},
			counterWrite(s.table, fmt.Sprintf("ip#%s#day#%s", ipIdentifier, day), MaxSubscriptionsPerIPDay, expiresAt),
			counterWrite(s.table, "global#day#"+day, MaxSubscriptionsPerDay, expiresAt),
		},
	})
	if err == nil {
		return nil
	}
	var cancelled *types.TransactionCanceledException
	if !errors.As(err, &cancelled) {
		return err
	}
	for index, reason := range cancelled.CancellationReasons {
		if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
			switch index {
			case 0:
				return ErrAlreadySubscribed
			case 1:
				return ErrDailyLimited
			case 2:
				return ErrGlobalLimited
			}
		}
	}
	return err
}

func subscriberItem(key string, email string, now time.Time) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"subscriberKey": &types.AttributeValueMemberS{Value: key},
		"email":         &types.AttributeValueMemberS{Value: email},
		"createdAt":     &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}
}

func counterWrite(table string, key string, limit int, expiresAt time.Time) types.TransactWriteItem {
	return types.TransactWriteItem{Update: &types.Update{
		TableName:           &table,
		Key:                 map[string]types.AttributeValue{"subscriberKey": &types.AttributeValueMemberS{Value: key}},
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
	}}
}

func emailIdentifier(email string) string {
	digest := sha256.Sum256([]byte(email))
	return hex.EncodeToString(digest[:])
}

func str(value string) *string { return &value }
func boolp(value bool) *bool   { return &value }
