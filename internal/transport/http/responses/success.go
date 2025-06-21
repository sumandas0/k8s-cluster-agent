package responses

import (
	"encoding/json"
	"net/http"
	"time"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Data     interface{} `json:"data"`
	Metadata Metadata    `json:"metadata"`
}

// Metadata contains request metadata
type Metadata struct {
	RequestID string    `json:"requestId"`
	Timestamp time.Time `json:"timestamp"`
}

// Success creates a new success response
func Success(data interface{}) SuccessResponse {
	return SuccessResponse{
		Data: data,
		Metadata: Metadata{
			Timestamp: time.Now(),
		},
	}
}

// WriteJSON writes a JSON response with proper headers
func WriteJSON(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// If encoding fails, we've already written the header, so just log
		// In a real app, you'd want proper error logging here
		return
	}
}

// WriteJSONWithStatus writes a JSON response with a specific status code
func WriteJSONWithStatus(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
