package apihttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"3d-api/internal/jobs"
)

func testRouter() http.Handler {
	service := jobs.NewService(jobs.NewMemoryRepository(), httpStubUploadSigner{}, 15*time.Minute)
	return NewRouter(NewJobsHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil))))
}

type httpStubUploadSigner struct{}

func (httpStubUploadSigner) PresignPut(
	context.Context,
	string,
	int64,
	time.Duration,
) (string, string, error) {
	return "https://uploads.example/presigned", http.MethodPut, nil
}

type failingUploadSigner struct {
	err error
}

func (s failingUploadSigner) PresignPut(
	context.Context,
	string,
	int64,
	time.Duration,
) (string, string, error) {
	return "", "", s.err
}

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
	}
	if response.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Errorf("body = %q", response.Body.String())
	}
}

func TestCreateAndGetJob(t *testing.T) {
	router := testRouter()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"fileName":"rotor.stl","fileSizeBytes":117884}`))
	createResponse := httptest.NewRecorder()

	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d; body = %s", createResponse.Code, http.StatusCreated, createResponse.Body.String())
	}
	var created createJobResponse
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}
	if created.ID == "" || created.FileName != "rotor.stl" || created.FileSizeBytes != 117884 || created.Status != jobs.StatusCreated {
		t.Errorf("POST body = %+v", created)
	}
	if created.Upload.URL != "https://uploads.example/presigned" || created.Upload.Method != http.MethodPut {
		t.Errorf("POST upload = %+v", created.Upload)
	}
	if want := created.CreatedAt.Add(15 * time.Minute); !created.Upload.ExpiresAt.Equal(want) {
		t.Errorf("POST upload expiry = %v, want %v", created.Upload.ExpiresAt, want)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+created.ID, nil)
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getResponse.Code, http.StatusOK)
	}
	var got jobResponse
	if err := json.NewDecoder(getResponse.Body).Decode(&got); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if got != created.jobResponse {
		t.Errorf("GET body = %+v, want %+v", got, created.jobResponse)
	}
}

func TestCreateLogsInternalError(t *testing.T) {
	var logs strings.Builder
	signingErr := errors.New("credentials unavailable")
	service := jobs.NewService(jobs.NewMemoryRepository(), failingUploadSigner{err: signingErr}, 15*time.Minute)
	handler := NewRouter(NewJobsHandler(service, slog.New(slog.NewTextHandler(&logs, nil))))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"fileName":"rotor.stl","fileSizeBytes":117884}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(logs.String(), "create job failed") || !strings.Contains(logs.String(), signingErr.Error()) {
		t.Errorf("logs = %q, want operation and underlying error", logs.String())
	}
	if strings.Contains(response.Body.String(), signingErr.Error()) {
		t.Errorf("response leaks internal error: %q", response.Body.String())
	}
}

func TestGetMissingJob(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if response.Body.String() != "{\"error\":\"job not found\"}\n" {
		t.Errorf("body = %q", response.Body.String())
	}
}

func TestCreateRejectsMalformedJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"fileName":`))
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantBody string
	}{
		{name: "missing filename", body: `{"fileSizeBytes":1}`, wantBody: `{"error":"fileName is required"}` + "\n"},
		{name: "non STL filename", body: `{"fileName":"rotor.obj","fileSizeBytes":1}`, wantBody: `{"error":"fileName must end in .stl"}` + "\n"},
		{name: "zero size", body: `{"fileName":"rotor.stl","fileSizeBytes":0}`, wantBody: `{"error":"fileSizeBytes must be greater than 0"}` + "\n"},
		{name: "negative size", body: `{"fileName":"rotor.stl","fileSizeBytes":-1}`, wantBody: `{"error":"fileSizeBytes must be greater than 0"}` + "\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(test.body))
			response := httptest.NewRecorder()

			testRouter().ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			if response.Body.String() != test.wantBody {
				t.Errorf("body = %q, want %q", response.Body.String(), test.wantBody)
			}
		})
	}
}

func TestCreateRejectsUnknownFields(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"fileName":"rotor.stl","fileSizeBytes":1,"unexpected":true}`))
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateRejectsOversizedFile(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"fileName":"rotor.stl","fileSizeBytes":26214401}`))
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestCreateRejectsMultipleJSONValues(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"fileName":"rotor.stl","fileSizeBytes":1} {}`))
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestRouterRejectsUnsupportedMethod(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/api/v1/jobs", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestOpenAPIDocument(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Content-Type") != "application/yaml" {
		t.Errorf("Content-Type = %q, want application/yaml", response.Header().Get("Content-Type"))
	}
	document := response.Body.String()
	if !strings.HasPrefix(document, "openapi: 3.1.0\n") {
		t.Error("OpenAPI document does not declare version 3.1.0")
	}
	for _, path := range []string{"  /health:", "  /jobs:", "  /jobs/{id}:"} {
		if !strings.Contains(document, path) {
			t.Errorf("OpenAPI document does not contain %q", path)
		}
	}
	if !strings.Contains(document, "  - url: /api/v1\n") {
		t.Error("OpenAPI document does not declare the versioned server URL")
	}
}

func TestSwaggerUI(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/swagger/", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Body.String(), `url: "/api/v1/openapi.yaml"`) {
		t.Error("Swagger UI does not reference the OpenAPI document")
	}
	if !strings.Contains(response.Body.String(), "tryItOutEnabled: true") {
		t.Error("Swagger UI does not enable Try it out by default")
	}
}

func TestSwaggerRedirect(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/swagger", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMovedPermanently)
	}
	if location := response.Header().Get("Location"); location != "/api/swagger/" {
		t.Errorf("Location = %q, want /api/swagger/", location)
	}
}

func TestUnversionedEndpointIsNotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
