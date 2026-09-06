package processingqueue

import (
	"context"
	"encoding/json"
	"testing"

	"3d-api/internal/jobs"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type fakeSQSClient struct {
	input *sqs.SendMessageInput
}

func (c *fakeSQSClient) SendMessage(_ context.Context, input *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	c.input = input
	return &sqs.SendMessageOutput{}, nil
}

func TestSQSDispatcherSendsOriginalInputAsProcessorEvent(t *testing.T) {
	client := &fakeSQSClient{}
	dispatcher := NewSQSDispatcher("uploads", "https://sqs.example/queue", client)
	job := jobs.Job{ID: "job-1", FileFormat: jobs.FileFormatSTL, Operation: jobs.OperationQualityCheck}

	if err := dispatcher.Dispatch(context.Background(), job, 42); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if aws.ToString(client.input.QueueUrl) != "https://sqs.example/queue" {
		t.Errorf("queue URL = %q", aws.ToString(client.input.QueueUrl))
	}
	var message eventMessage
	if err := json.Unmarshal([]byte(aws.ToString(client.input.MessageBody)), &message); err != nil {
		t.Fatalf("decode message: %v", err)
	}
	if len(message.Records) != 1 || message.Records[0].S3.Bucket.Name != "uploads" || message.Records[0].S3.Object.Key != "jobs%2Fjob-1%2Finput%2Finput.stl" || message.Records[0].S3.Object.Size != 42 {
		t.Errorf("message = %+v", message)
	}
}
