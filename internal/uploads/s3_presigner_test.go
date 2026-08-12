package uploads

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3PresignerCreatesPutRequest(t *testing.T) {
	config := aws.Config{
		Region:      "eu-west-2",
		Credentials: credentials.NewStaticCredentialsProvider("access-key", "secret-key", ""),
	}
	presigner := NewS3Presigner("quality-check-uploads", s3.NewPresignClient(s3.NewFromConfig(config)))

	presignedURL, method, err := presigner.PresignPut(
		context.Background(),
		"jobs/job-1/input.stl",
		117884,
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("PresignPut() error = %v", err)
	}
	if method != http.MethodPut {
		t.Errorf("method = %q, want PUT", method)
	}

	parsed, err := url.Parse(presignedURL)
	if err != nil {
		t.Fatalf("parse presigned URL: %v", err)
	}
	if parsed.Host != "quality-check-uploads.s3.eu-west-2.amazonaws.com" {
		t.Errorf("host = %q", parsed.Host)
	}
	if parsed.Path != "/jobs/job-1/input.stl" {
		t.Errorf("path = %q", parsed.Path)
	}
	if expires := parsed.Query().Get("X-Amz-Expires"); expires != "900" {
		t.Errorf("X-Amz-Expires = %q, want 900", expires)
	}
}
