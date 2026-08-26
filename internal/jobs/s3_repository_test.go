package jobs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type fakeS3RepositoryClient struct {
	objects map[string][]byte
}

func newFakeS3RepositoryClient() *fakeS3RepositoryClient {
	return &fakeS3RepositoryClient{objects: make(map[string][]byte)}
}

func (c *fakeS3RepositoryClient) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	c.objects[aws.ToString(input.Key)] = body
	return &s3.PutObjectOutput{}, nil
}

func (c *fakeS3RepositoryClient) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	body, ok := c.objects[aws.ToString(input.Key)]
	if !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func (c *fakeS3RepositoryClient) HeadObject(_ context.Context, input *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if _, ok := c.objects[aws.ToString(input.Key)]; !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.HeadObjectOutput{}, nil
}

func TestS3RepositoryTracksCreatedProcessingAndCompleted(t *testing.T) {
	ctx := context.Background()
	client := newFakeS3RepositoryClient()
	repository := NewS3Repository("bucket", client)
	job := Job{
		ID: "0123456789abcdef0123456789abcdef", FileName: "model.stl", FileSizeBytes: 42,
		Status: StatusCreated, CreatedAt: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC),
	}
	if err := repository.Create(ctx, job); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	created, err := repository.GetByID(ctx, job.ID)
	if err != nil || created.Status != StatusCreated {
		t.Fatalf("created job = %+v, error = %v", created, err)
	}

	client.objects[jobObjectKey(job.ID, "input.stl")] = []byte("mesh")
	processing, err := repository.GetByID(ctx, job.ID)
	if err != nil || processing.Status != StatusProcessing {
		t.Fatalf("processing job = %+v, error = %v", processing, err)
	}

	client.objects[jobObjectKey(job.ID, "report.json")] = []byte(`{"triangleCount":12}`)
	completed, err := repository.GetByID(ctx, job.ID)
	if err != nil || completed.Status != StatusCompleted {
		t.Fatalf("completed job = %+v, error = %v", completed, err)
	}
	if string(completed.Report) != `{"triangleCount":12}` {
		t.Errorf("report = %s", completed.Report)
	}

	delete(client.objects, jobObjectKey(job.ID, "report.json"))
	client.objects[jobObjectKey(job.ID, "failure.json")] = []byte(`{"error":{"code":"processing_failed"}}`)
	failed, err := repository.GetByID(ctx, job.ID)
	if err != nil || failed.Status != StatusFailed {
		t.Fatalf("failed job = %+v, error = %v", failed, err)
	}
	if len(failed.Failure) == 0 {
		t.Error("failed job does not include failure details")
	}
}

func TestS3RepositoryReturnsNotFoundForMissingMetadata(t *testing.T) {
	_, err := NewS3Repository("bucket", newFakeS3RepositoryClient()).GetByID(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestS3RepositoryTracksThreeMfInputUsingStoredFormat(t *testing.T) {
	ctx := context.Background()
	client := newFakeS3RepositoryClient()
	repository := NewS3Repository("bucket", client)
	job := Job{
		ID: "0123456789abcdef0123456789abcdef", FileName: "model.3mf", FileSizeBytes: 42,
		FileFormat: FileFormat3MF, Status: StatusCreated, CreatedAt: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC),
	}
	if err := repository.Create(ctx, job); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	client.objects[jobObjectKey(job.ID, "input.3mf")] = []byte("3mf")
	stored, err := repository.GetByID(ctx, job.ID)
	if err != nil || stored.Status != StatusProcessing || stored.FileFormat != FileFormat3MF {
		t.Fatalf("stored job = %+v, error = %v", stored, err)
	}
}
