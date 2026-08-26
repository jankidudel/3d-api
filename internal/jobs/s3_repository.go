package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const jobsPrefix = "jobs/"

type S3RepositoryClient interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

type S3Repository struct {
	bucket string
	client S3RepositoryClient
}

type jobMetadata struct {
	ID            string     `json:"id"`
	FileName      string     `json:"fileName"`
	FileFormat    FileFormat `json:"fileFormat"`
	FileSizeBytes int64      `json:"fileSizeBytes"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func NewS3Repository(bucket string, client S3RepositoryClient) *S3Repository {
	return &S3Repository{bucket: bucket, client: client}
}

func (r *S3Repository) Create(ctx context.Context, job Job) error {
	metadata, err := json.Marshal(jobMetadata{
		ID: job.ID, FileName: job.FileName, FileSizeBytes: job.FileSizeBytes, FileFormat: job.FileFormat, CreatedAt: job.CreatedAt,
	})
	if err != nil {
		return err
	}
	_, err = r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(jobObjectKey(job.ID, "metadata.json")),
		Body:        bytes.NewReader(metadata),
		ContentType: aws.String("application/json"),
	})
	return err
}

func (r *S3Repository) GetByID(ctx context.Context, id string) (Job, error) {
	metadataObject, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket), Key: aws.String(jobObjectKey(id, "metadata.json")),
	})
	if isNotFound(err) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	defer metadataObject.Body.Close()
	var metadata jobMetadata
	if err := json.NewDecoder(metadataObject.Body).Decode(&metadata); err != nil {
		return Job{}, err
	}

	job := Job{
		ID: metadata.ID, FileName: metadata.FileName, FileSizeBytes: metadata.FileSizeBytes, FileFormat: metadata.FileFormat,
		Status: StatusCreated, CreatedAt: metadata.CreatedAt,
	}
	reportObject, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket), Key: aws.String(jobObjectKey(id, "report.json")),
	})
	if err == nil {
		defer reportObject.Body.Close()
		job.Report, err = io.ReadAll(reportObject.Body)
		if err != nil {
			return Job{}, err
		}
		if !json.Valid(job.Report) {
			return Job{}, errors.New("stored job report is not valid JSON")
		}
		job.Status = StatusCompleted
		return job, nil
	}
	if !isNotFound(err) {
		return Job{}, err
	}
	failureObject, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket), Key: aws.String(jobObjectKey(id, "failure.json")),
	})
	if err == nil {
		defer failureObject.Body.Close()
		job.Failure, err = io.ReadAll(failureObject.Body)
		if err != nil {
			return Job{}, err
		}
		if !json.Valid(job.Failure) {
			return Job{}, errors.New("stored job failure is not valid JSON")
		}
		job.Status = StatusFailed
		return job, nil
	}
	if !isNotFound(err) {
		return Job{}, err
	}

	format := job.FileFormat
	if format == "" {
		format = FileFormatSTL
	}
	_, err = r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket), Key: aws.String(jobObjectKey(id, "input."+string(format))),
	})
	if err == nil {
		job.Status = StatusProcessing
	} else if !isNotFound(err) {
		return Job{}, err
	}
	return job, nil
}

func jobObjectKey(id string, name string) string {
	return jobsPrefix + id + "/" + name
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var apiError smithy.APIError
	return errors.As(err, &apiError) &&
		(apiError.ErrorCode() == "NoSuchKey" || apiError.ErrorCode() == "NotFound")
}
