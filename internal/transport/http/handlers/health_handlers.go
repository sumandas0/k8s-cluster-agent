package handlers

import (
	"encoding/json"
	"net/http"
)

// HealthResponse represents a health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// HandleHealth handles GET /healthz
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleReadiness handles GET /readyz
func HandleReadiness(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would check if all dependencies are ready
	// For now, we'll just return OK
	response := HealthResponse{
		Status: "ready",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
