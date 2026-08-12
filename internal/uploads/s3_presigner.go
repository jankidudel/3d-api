package uploads

import (
	"context"
	"time"

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

func (p *S3Presigner) PresignPut(
	ctx context.Context,
	objectKey string,
	fileSizeBytes int64,
	lifetime time.Duration,
) (string, string, error) {
	request, err := p.client.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(p.bucket),
		Key:           aws.String(objectKey),
		ContentLength: aws.Int64(fileSizeBytes),
	}, func(options *s3.PresignOptions) {
		options.Expires = lifetime
	})
	if err != nil {
		return "", "", err
	}

	return request.URL, request.Method, nil
}
