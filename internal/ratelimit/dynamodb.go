package ratelimit

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const maxWriteRetries = 8

type DynamoClient interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}
type DynamoStore struct {
	table  string
	client DynamoClient
}

func NewDynamoStore(table string, client DynamoClient) *DynamoStore {
	return &DynamoStore{table: table, client: client}
}

func (s *DynamoStore) Consume(ctx context.Context, id string, uploadID string, size int64, now time.Time) error {
	key := map[string]types.AttributeValue{"windowKey": &types.AttributeValueMemberS{Value: "ip#" + id}}
	for attempt := 0; attempt < maxWriteRetries; attempt++ {
		got, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: &s.table, Key: key, ConsistentRead: boolp(true)})
		if err != nil {
			return err
		}
		timestamps, sizes, ids, version := decode(got.Item)
		cutoff := now.Add(-RollingDay)
		kept := timestamps[:0]
		keptSizes := sizes[:0]
		keptIDs := ids[:0]
		for index, timestamp := range timestamps {
			if timestamp.After(cutoff) {
				kept = append(kept, timestamp)
				keptSizes = append(keptSizes, sizes[index])
				keptIDs = append(keptIDs, ids[index])
			}
		}
		for _, existingID := range keptIDs {
			if existingID == uploadID {
				return nil
			}
		}
		hourly := 0
		hourCutoff := now.Add(-RollingHour)
		for _, timestamp := range kept {
			if timestamp.After(hourCutoff) {
				hourly++
			}
		}
		if hourly >= MaxAcceptedJobsPerHour {
			return ErrLimited
		}
		if len(kept) >= MaxAcceptedJobsPerDay {
			return ErrDailyLimited
		}
		var totalBytes int64
		for _, acceptedSize := range keptSizes {
			totalBytes += acceptedSize
		}
		if totalBytes+size > MaxAcceptedBytesPerDay {
			return ErrByteLimited
		}
		kept = append(kept, now)
		keptSizes = append(keptSizes, size)
		keptIDs = append(keptIDs, uploadID)
		values := map[string]types.AttributeValue{
			":timestamps": &types.AttributeValueMemberL{Value: times(kept)},
			":sizes":      &types.AttributeValueMemberL{Value: integers(keptSizes)},
			":ids":        &types.AttributeValueMemberL{Value: strings(keptIDs)},
			":expires":    &types.AttributeValueMemberN{Value: strconv.FormatInt(kept[len(kept)-1].Add(RollingDay).Unix(), 10)},
			":next":       &types.AttributeValueMemberN{Value: strconv.FormatInt(version+1, 10)},
		}
		condition := "attribute_not_exists(#version)"
		if got.Item != nil {
			condition = "#version = :version"
			values[":version"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(version, 10)}
		}
		_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{TableName: &s.table, Key: key, UpdateExpression: str("SET #timestamps = :timestamps, #sizes = :sizes, #ids = :ids, expiresAt = :expires, #version = :next"), ConditionExpression: str(condition), ExpressionAttributeNames: map[string]string{"#timestamps": "timestamps", "#sizes": "sizes", "#ids": "ids", "#version": "version"}, ExpressionAttributeValues: values})
		if err == nil {
			return nil
		}
		var conditional *types.ConditionalCheckFailedException
		if !errors.As(err, &conditional) {
			return err
		}
	}
	return errors.New("rate limit contention exceeded retry budget")
}
func decode(item map[string]types.AttributeValue) ([]time.Time, []int64, []string, int64) {
	var result []time.Time
	var version int64
	if v, ok := item["timestamps"].(*types.AttributeValueMemberL); ok {
		for _, x := range v.Value {
			if n, ok := x.(*types.AttributeValueMemberN); ok {
				if seconds, err := strconv.ParseInt(n.Value, 10, 64); err == nil {
					result = append(result, time.Unix(seconds, 0).UTC())
				}
			}
		}
	}
	if v, ok := item["version"].(*types.AttributeValueMemberN); ok {
		version, _ = strconv.ParseInt(v.Value, 10, 64)
	}
	var sizes []int64
	if v, ok := item["sizes"].(*types.AttributeValueMemberL); ok {
		for _, x := range v.Value {
			if n, ok := x.(*types.AttributeValueMemberN); ok {
				value, _ := strconv.ParseInt(n.Value, 10, 64)
				sizes = append(sizes, value)
			}
		}
	}
	var ids []string
	if v, ok := item["ids"].(*types.AttributeValueMemberL); ok {
		for _, x := range v.Value {
			if value, ok := x.(*types.AttributeValueMemberS); ok {
				ids = append(ids, value.Value)
			}
		}
	}
	if len(sizes) != len(result) || len(ids) != len(result) {
		return nil, nil, nil, version
	}
	return result, sizes, ids, version
}
func integers(values []int64) []types.AttributeValue {
	result := make([]types.AttributeValue, 0, len(values))
	for _, value := range values {
		result = append(result, &types.AttributeValueMemberN{Value: strconv.FormatInt(value, 10)})
	}
	return result
}
func strings(values []string) []types.AttributeValue {
	result := make([]types.AttributeValue, 0, len(values))
	for _, value := range values {
		result = append(result, &types.AttributeValueMemberS{Value: value})
	}
	return result
}
func times(values []time.Time) []types.AttributeValue {
	result := make([]types.AttributeValue, 0, len(values))
	for _, value := range values {
		result = append(result, &types.AttributeValueMemberN{Value: strconv.FormatInt(value.Unix(), 10)})
	}
	return result
}
func str(value string) *string { return &value }
func boolp(value bool) *bool   { return &value }
