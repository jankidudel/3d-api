package uploads

import (
	"context"
	"time"

	"3d-api/internal/jobs"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Presigner struct {
	bucket string
	client *s3.PresignClient
}

func NewS3Presigner(bucket string, client *s3.PresignClient) *S3Presigner {
	return &S3Presigner{bucket: bucket, client: client}
}

func (p *S3Presigner) PresignPost(
	ctx context.Context,
	objectKey string,
	fileSizeBytes int64,
	lifetime time.Duration,
) (string, map[string]string, error) {
	request, err := p.client.PresignPostObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(objectKey),
	}, func(options *s3.PresignPostOptions) {
		options.Expires = lifetime
		options.Conditions = []any{
			[]any{"content-length-range", 1, jobs.MaxUploadSizeBytes},
		}
	})
	if err != nil {
		return "", nil, err
	}

	return request.URL, request.Values, nil
}
