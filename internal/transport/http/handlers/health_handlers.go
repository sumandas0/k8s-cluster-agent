package handlers

import (
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status"`
}

// HandleHealth returns the health status of the service
// @Summary Health check endpoint
// @Description Returns the health status of the K8s Cluster Agent service
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Router /healthz [get]
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleReadiness returns the readiness status of the service
// @Summary Readiness check endpoint
// @Description Returns the readiness status of the K8s Cluster Agent service
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse "Service is ready"
// @Router /readyz [get]
func HandleReadiness(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ready",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
