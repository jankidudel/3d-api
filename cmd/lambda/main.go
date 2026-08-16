package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"3d-api/internal/application"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type adapter struct {
	handler http.Handler
}

func (a adapter) Handle(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	body := []byte(event.Body)
	if event.IsBase64Encoded {
		var err error
		body, err = base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{}, err
		}
	}
	path := event.RawPath
	if path == "" {
		path = event.RequestContext.HTTP.Path
	}
	request, err := http.NewRequestWithContext(
		ctx,
		event.RequestContext.HTTP.Method,
		path+rawQuery(event.RawQueryString),
		bytes.NewReader(body),
	)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	for name, value := range event.Headers {
		request.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	a.handler.ServeHTTP(recorder, request)
	result := recorder.Result()
	defer result.Body.Close()
	responseBody := recorder.Body.String()
	return events.APIGatewayV2HTTPResponse{
		StatusCode: result.StatusCode,
		Headers:    flattenHeaders(result.Header),
		Body:       responseBody,
	}, nil
}

func rawQuery(query string) string {
	if query == "" {
		return ""
	}
	return "?" + query
}

func flattenHeaders(headers http.Header) map[string]string {
	flattened := make(map[string]string, len(headers))
	for name := range headers {
		flattened[name] = headers.Get(name)
	}
	return flattened
}

func main() {
	handler, err := application.NewHandler(context.Background(), slog.Default())
	if err != nil {
		log.Fatal(errors.New("initialize Lambda API: " + err.Error()))
	}
	lambda.Start(adapter{handler: handler}.Handle)
}
