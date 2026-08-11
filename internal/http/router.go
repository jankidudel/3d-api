package apihttp

import "net/http"

func NewRouter(jobsHandler *JobsHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /jobs", jobsHandler.Create)
	mux.HandleFunc("GET /jobs/{id}", jobsHandler.GetByID)
	return mux
}
