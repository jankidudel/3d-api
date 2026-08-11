package apihttp

import (
	"net/http"

	"3d-api/internal/response"
)

type healthResponse struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
