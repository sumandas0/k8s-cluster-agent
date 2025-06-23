package responses

import (
	"encoding/json"
	"net/http"
	"time"
)

type SuccessResponse[T any] struct {
	Data     T        `json:"data"`
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	RequestID string    `json:"requestId"`
	Timestamp time.Time `json:"timestamp"`
}

func Success[T any](data T) SuccessResponse[T] {
	return SuccessResponse[T]{
		Data: data,
		Metadata: Metadata{
			Timestamp: time.Now(),
		},
	}
}

func WriteJSON[T any](w http.ResponseWriter, response T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func WriteJSONWithStatus[T any](w http.ResponseWriter, status int, response T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
