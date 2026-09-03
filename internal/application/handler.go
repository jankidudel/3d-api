package application

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	apihttp "3d-api/internal/http"
	"3d-api/internal/jobs"
	"3d-api/internal/ratelimit"
	"3d-api/internal/uploads"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// @todo: review, why is it named "New*" ?
func NewHandler(ctx context.Context, logger *slog.Logger) (http.Handler, error) {
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		return nil, errors.New("S3_BUCKET is required")
	}
	table := os.Getenv("RATE_LIMIT_TABLE")
	secret := os.Getenv("RATE_LIMIT_SECRET")
	if table == "" || secret == "" {
		return nil, errors.New("RATE_LIMIT_TABLE and RATE_LIMIT_SECRET are required")
	}
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(awsConfig)
	limiter := ratelimit.New(ratelimit.NewDynamoStore(table, dynamodb.NewFromConfig(awsConfig)), secret)
	uploadSigner := uploads.NewS3Presigner(bucket, s3.NewPresignClient(s3Client))
	repository := jobs.NewS3Repository(bucket, s3Client)
	service := jobs.NewService(repository, uploadSigner, 15*time.Minute)
	return apihttp.NewRouter(
		apihttp.NewJobsHandler(service, limiter, logger, uploadSigner),
		os.Getenv("ENABLE_API_DOCUMENTATION") != "false",
	), nil
}
