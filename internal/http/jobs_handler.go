package apihttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"3d-api/internal/jobs"
	"3d-api/internal/ratelimit"
	"3d-api/internal/response"
)

type JobsHandler struct {
	service *jobs.Service
	limiter *ratelimit.Limiter
	logger  *slog.Logger
}

type createJobRequest struct {
	FileName      string `json:"fileName"`
	FileSizeBytes int64  `json:"fileSizeBytes"`
}

type jobResponse struct {
	ID            string          `json:"id"`
	FileName      string          `json:"fileName"`
	FileSizeBytes int64           `json:"fileSizeBytes"`
	Status        jobs.Status     `json:"status"`
	CreatedAt     time.Time       `json:"createdAt"`
	Report        json.RawMessage `json:"report,omitempty"`
	Failure       json.RawMessage `json:"failure,omitempty"`
}

type createJobResponse struct {
	jobResponse
	Upload uploadResponse `json:"upload"`
}

type uploadResponse struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

func NewJobsHandler(service *jobs.Service, limiter *ratelimit.Limiter, logger *slog.Logger) *JobsHandler {
	return &JobsHandler{service: service, limiter: limiter, logger: logger}
}

func (h *JobsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request createJobRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	input := jobs.CreateInput{
		FileName:      request.FileName,
		FileSizeBytes: request.FileSizeBytes,
	}
	if err := jobs.ValidateCreateInput(input); err != nil {
		if errors.Is(err, jobs.ErrFileTooLarge) {
			response.CodedError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", jobs.FileTooLargeMessage, nil)
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.limiter != nil {
		retryAfter, err := h.limiter.Allow(r.Context(), r.RemoteAddr)
		if errors.Is(err, ratelimit.ErrLimited) {
			seconds := max(1, int(retryAfter.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			response.CodedError(w, http.StatusTooManyRequests, "RATE_LIMITED", "You have reached the free upload limit. Please try again later.", &seconds)
			return
		}
		if err != nil {
			h.logger.ErrorContext(r.Context(), "rate limit failed", "error", err)
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	result, err := h.service.Create(r.Context(), input)
	if err != nil {
		if isValidationError(err) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.ErrorContext(r.Context(), "create job failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusCreated, createJobResponse{
		jobResponse: toJobResponse(result.Job),
		Upload: uploadResponse{
			URL:       result.Upload.URL,
			Method:    result.Upload.Method,
			Fields:    result.Upload.Fields,
			ExpiresAt: result.Upload.ExpiresAt,
		},
	})
}

func (h *JobsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	job, err := h.service.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "job not found")
			return
		}
		h.logger.ErrorContext(r.Context(), "get job failed", "job_id", r.PathValue("id"), "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, toJobResponse(job))
}

func isValidationError(err error) bool {
	return errors.Is(err, jobs.ErrFileNameRequired) ||
		errors.Is(err, jobs.ErrFileNameNotSTL) ||
		errors.Is(err, jobs.ErrFileSizeInvalid)
}

func toJobResponse(job jobs.Job) jobResponse {
	return jobResponse{
		ID:            job.ID,
		FileName:      job.FileName,
		FileSizeBytes: job.FileSizeBytes,
		Status:        job.Status,
		CreatedAt:     job.CreatedAt,
		Report:        job.Report,
		Failure:       job.Failure,
	}
}
