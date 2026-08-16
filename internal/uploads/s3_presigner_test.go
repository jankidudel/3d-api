package uploads

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3PresignerCreatesSizeLimitedPostRequest(t *testing.T) {
	config := aws.Config{
		Region:      "eu-west-2",
		Credentials: credentials.NewStaticCredentialsProvider("access-key", "secret-key", ""),
	}
	presigner := NewS3Presigner("quality-check-uploads", s3.NewPresignClient(s3.NewFromConfig(config)))

	presignedURL, fields, err := presigner.PresignPost(
		context.Background(),
		"jobs/job-1/input.stl",
		117884,
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("PresignPost() error = %v", err)
	}

	parsed, err := url.Parse(presignedURL)
	if err != nil {
		t.Fatalf("parse presigned URL: %v", err)
	}
	if parsed.Host != "quality-check-uploads.s3.eu-west-2.amazonaws.com" {
		t.Errorf("host = %q", parsed.Host)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		t.Errorf("path = %q, want bucket root", parsed.Path)
	}
	if fields["key"] != "jobs/job-1/input.stl" {
		t.Errorf("key = %q", fields["key"])
	}
	policyJSON, err := base64.StdEncoding.DecodeString(fields["policy"])
	if err != nil {
		t.Fatalf("decode policy: %v", err)
	}
	var policy struct {
		Conditions []any `json:"conditions"`
	}
	if err := json.Unmarshal(policyJSON, &policy); err != nil {
		t.Fatalf("parse policy: %v", err)
	}
	wantCondition := []any{"content-length-range", float64(1), float64(117884)}
	found := false
	for _, condition := range policy.Conditions {
		if values, ok := condition.([]any); ok && len(values) == 3 &&
			values[0] == wantCondition[0] && values[1] == wantCondition[1] && values[2] == wantCondition[2] {
			found = true
		}
	}
	if !found {
		t.Errorf("policy does not contain content-length-range condition: %s", policyJSON)
	}
}
