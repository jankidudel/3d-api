package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestAdapterRoutesAPIGatewayV2Request(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" || r.URL.Query().Get("check") != "true" {
			t.Errorf("request URL = %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	response, err := (adapter{handler: handler}).Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/api/v1/health", RawQueryString: "check=true",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: http.MethodGet},
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusOK || response.Body != `{"status":"ok"}` {
		t.Fatalf("response = %+v", response)
	}
	if response.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q", response.Headers["Content-Type"])
	}
}
