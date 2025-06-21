package responses

import (
	"encoding/json"
	"net/http"
	"time"
)

type SuccessResponse struct {
	Data     interface{} `json:"data"`
	Metadata Metadata    `json:"metadata"`
}

type Metadata struct {
	RequestID string    `json:"requestId"`
	Timestamp time.Time `json:"timestamp"`
}

func Success(data interface{}) SuccessResponse {
	return SuccessResponse{
		Data: data,
		Metadata: Metadata{
			Timestamp: time.Now(),
		},
	}
}

func WriteJSON(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func WriteJSONWithStatus(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
