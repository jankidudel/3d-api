package application

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"3d-api/internal/contact"
	apihttp "3d-api/internal/http"
	"3d-api/internal/jobs"
	"3d-api/internal/processingqueue"
	"3d-api/internal/ratelimit"
	"3d-api/internal/subscriptions"
	"3d-api/internal/uploads"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// @todo: review, why is it named "New*" ?
func NewHandler(ctx context.Context, logger *slog.Logger) (http.Handler, error) {
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		return nil, errors.New("S3_BUCKET is required")
	}
	table := os.Getenv("RATE_LIMIT_TABLE")
	secret := os.Getenv("RATE_LIMIT_SECRET")
	queueURL := os.Getenv("PROCESSING_QUEUE_URL")
	if table == "" || secret == "" || queueURL == "" {
		return nil, errors.New("RATE_LIMIT_TABLE, RATE_LIMIT_SECRET and PROCESSING_QUEUE_URL are required")
	}
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(awsConfig)
	dynamoClient := dynamodb.NewFromConfig(awsConfig)
	limiter := ratelimit.New(ratelimit.NewDynamoStore(table, dynamoClient), secret)
	uploadSigner := uploads.NewS3Presigner(bucket, s3.NewPresignClient(s3Client))
	repository := jobs.NewS3Repository(bucket, s3Client)
	dispatcher := processingqueue.NewSQSDispatcher(bucket, queueURL, sqs.NewFromConfig(awsConfig))
	service := jobs.NewService(repository, uploadSigner, 15*time.Minute, dispatcher)
	contactHandler := apihttp.NewContactHandler(nil, nil, nil, logger)
	subscriptionHandler := apihttp.NewSubscriptionHandler(nil, nil, nil, logger)
	contactEmail := os.Getenv("CONTACT_EMAIL")
	turnstileSecret := os.Getenv("TURNSTILE_SECRET")
	subscriberTable := os.Getenv("SUBSCRIBER_TABLE")
	if contactEmail != "" && turnstileSecret != "" {
		sender := contact.NewSESSender(sesv2.NewFromConfig(awsConfig), contactEmail)
		captcha := contact.NewTurnstileVerifier(turnstileSecret, http.DefaultClient)
		contactHandler = apihttp.NewContactHandler(
			sender,
			captcha,
			ratelimit.NewContactLimiter(ratelimit.NewContactDynamoStore(table, dynamoClient), secret),
			logger,
		)
		if subscriberTable != "" {
			subscriptionHandler = apihttp.NewSubscriptionHandler(
				sender,
				captcha,
				subscriptions.NewService(subscriptions.NewDynamoStore(subscriberTable, dynamoClient), secret),
				logger,
			)
		}
	}
	return apihttp.NewRouter(
		apihttp.NewJobsHandler(service, limiter, logger, uploadSigner),
		contactHandler,
		subscriptionHandler,
		os.Getenv("ENABLE_API_DOCUMENTATION") != "false",
	), nil
}
