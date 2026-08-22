package response

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Error string `json:"error"`
}

type CodedErrorBody struct {
	Code              string `json:"code"`
	Message           string `json:"message"`
	RetryAfterSeconds *int   `json:"retryAfterSeconds,omitempty"`
}

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorBody{Error: message})
}

func CodedError(w http.ResponseWriter, status int, code string, message string, retryAfterSeconds *int) {
	JSON(w, status, CodedErrorBody{Code: code, Message: message, RetryAfterSeconds: retryAfterSeconds})
}
