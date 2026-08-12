package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apihttp "3d-api/internal/http"
	"3d-api/internal/jobs"
	"3d-api/internal/uploads"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		log.Fatal("S3_BUCKET is required")
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("load AWS configuration: %v", err)
	}
	uploadSigner := uploads.NewS3Presigner(bucket, s3.NewPresignClient(s3.NewFromConfig(awsConfig)))
	repository := jobs.NewMemoryRepository()
	service := jobs.NewService(repository, uploadSigner, 15*time.Minute)
	handler := apihttp.NewRouter(apihttp.NewJobsHandler(service))

	server := &http.Server{
		Addr:              ":" + port(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("API listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func port() string {
	if value := os.Getenv("PORT"); value != "" {
		return value
	}
	return "8080"
}
