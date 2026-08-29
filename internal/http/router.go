package apihttp

import "net/http"

const apiPrefix = "/api/v1"
const swaggerPrefix = "/api/swagger"

func NewRouter(jobsHandler *JobsHandler, documentationEnabled bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+apiPrefix+"/health", healthHandler)
	mux.HandleFunc("POST "+apiPrefix+"/jobs", jobsHandler.Create)
	mux.HandleFunc("GET "+apiPrefix+"/jobs/{id}", jobsHandler.GetByID)
	if documentationEnabled {
		mux.HandleFunc("GET "+apiPrefix+"/openapi.yaml", openAPIHandler)
		mux.HandleFunc("GET "+swaggerPrefix+"/{$}", swaggerUIHandler)
		mux.HandleFunc("GET "+swaggerPrefix, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, swaggerPrefix+"/", http.StatusMovedPermanently)
		})
	}
	return mux
}
