package processingqueue

import (
	"context"
	"encoding/json"
	"net/url"

	"3d-api/internal/jobs"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface {
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type SQSDispatcher struct {
	bucket   string
	queueURL string
	client   SQSClient
}

type eventMessage struct {
	Records []eventRecord `json:"Records"`
}

type eventRecord struct {
	S3 eventS3 `json:"s3"`
}

type eventS3 struct {
	Bucket eventBucket `json:"bucket"`
	Object eventObject `json:"object"`
}

type eventBucket struct {
	Name string `json:"name"`
}

type eventObject struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

func NewSQSDispatcher(bucket string, queueURL string, client SQSClient) *SQSDispatcher {
	return &SQSDispatcher{bucket: bucket, queueURL: queueURL, client: client}
}

func (d *SQSDispatcher) Dispatch(ctx context.Context, job jobs.Job, size int64) error {
	inputName, err := job.FileFormat.InputObjectName(job.Operation)
	if err != nil {
		return err
	}
	message, err := json.Marshal(eventMessage{Records: []eventRecord{{S3: eventS3{
		Bucket: eventBucket{Name: d.bucket},
		Object: eventObject{
			Key:  url.QueryEscape("jobs/" + job.ID + "/" + inputName),
			Size: size,
		},
	}}}})
	if err != nil {
		return err
	}
	_, err = d.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(d.queueURL),
		MessageBody: aws.String(string(message)),
	})
	return err
}
