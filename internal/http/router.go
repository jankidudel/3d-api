package apihttp

import "net/http"

func NewRouter(jobsHandler *JobsHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /jobs", jobsHandler.Create)
	mux.HandleFunc("GET /jobs/{id}", jobsHandler.GetByID)
	mux.HandleFunc("GET /openapi.yaml", openAPIHandler)
	mux.HandleFunc("GET /swagger/{$}", swaggerUIHandler)
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	return mux
}
