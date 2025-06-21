package responses

import (
	"encoding/json"
	"net/http"
	"time"
)

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Error    ErrorDetail `json:"error"`
	Metadata Metadata    `json:"metadata"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// WriteError writes a generic error response
func WriteError(w http.ResponseWriter, statusCode int, code, message, details string) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		Metadata: Metadata{
			Timestamp: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// WriteBadRequest writes a 400 Bad Request error
func WriteBadRequest(w http.ResponseWriter, err error) {
	WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request", err.Error())
}

// WriteNotFound writes a 404 Not Found error
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", message, "")
}

// WriteInternalError writes a 500 Internal Server Error
func WriteInternalError(w http.ResponseWriter, message string) {
	// Don't expose internal error details to clients
	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message, "")
}

// WriteTimeout writes a 504 Gateway Timeout error
func WriteTimeout(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusGatewayTimeout, "REQUEST_TIMEOUT", message, "")
}

// WriteServiceUnavailable writes a 503 Service Unavailable error
func WriteServiceUnavailable(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message, "")
}
