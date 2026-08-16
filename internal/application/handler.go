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
	"3d-api/internal/uploads"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewHandler(ctx context.Context, logger *slog.Logger) (http.Handler, error) {
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		return nil, errors.New("S3_BUCKET is required")
	}
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(awsConfig)
	uploadSigner := uploads.NewS3Presigner(bucket, s3.NewPresignClient(s3Client))
	repository := jobs.NewS3Repository(bucket, s3Client)
	service := jobs.NewService(repository, uploadSigner, 15*time.Minute)
	return apihttp.NewRouter(apihttp.NewJobsHandler(service, logger)), nil
}
