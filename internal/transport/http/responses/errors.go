package responses

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type ErrorResponse struct {
	Error    ErrorDetail `json:"error"`
	Metadata Metadata    `json:"metadata"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

func WriteBadRequest(w http.ResponseWriter, err error) {
	WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request", err.Error())
}

func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", message, "")
}

func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message, "")
}

func WriteTimeout(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusGatewayTimeout, "REQUEST_TIMEOUT", message, "")
}

func WriteServiceUnavailable(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message, "")
}
