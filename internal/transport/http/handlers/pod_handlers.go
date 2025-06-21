package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

// PodHandlers contains pod-related HTTP handlers
type PodHandlers struct {
	podService core.PodService
}

// NewPodHandlers creates a new PodHandlers instance
func NewPodHandlers(podService core.PodService) *PodHandlers {
	return &PodHandlers{
		podService: podService,
	}
}

// GetPodDescribe handles GET /api/v1/pods/{namespace}/{podName}/describe
func (h *PodHandlers) GetPodDescribe(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")

	// Validate input
	if err := validatePodParams(namespace, podName); err != nil {
		responses.WriteBadRequest(w, err)
		return
	}

	// Get pod details
	pod, err := h.podService.GetPod(r.Context(), namespace, podName)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Write response
	responses.WriteJSON(w, responses.Success(pod))
}

// GetPodScheduling handles GET /api/v1/pods/{namespace}/{podName}/scheduling
func (h *PodHandlers) GetPodScheduling(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")

	// Validate input
	if err := validatePodParams(namespace, podName); err != nil {
		responses.WriteBadRequest(w, err)
		return
	}

	// Get pod scheduling info
	scheduling, err := h.podService.GetPodScheduling(r.Context(), namespace, podName)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Write response
	responses.WriteJSON(w, responses.Success(scheduling))
}

// GetPodResources handles GET /api/v1/pods/{namespace}/{podName}/resources
func (h *PodHandlers) GetPodResources(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")

	// Validate input
	if err := validatePodParams(namespace, podName); err != nil {
		responses.WriteBadRequest(w, err)
		return
	}

	// Get pod resources
	resources, err := h.podService.GetPodResources(r.Context(), namespace, podName)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Write response
	responses.WriteJSON(w, responses.Success(resources))
}

// validatePodParams validates pod-related request parameters
func validatePodParams(namespace, podName string) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if podName == "" {
		return fmt.Errorf("pod name is required")
	}
	return nil
}

// handleServiceError maps service errors to HTTP responses
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, core.ErrPodNotFound):
		responses.WriteNotFound(w, "Pod not found")
	case errors.Is(err, core.ErrNodeNotFound):
		responses.WriteNotFound(w, "Node not found")
	case errors.Is(err, core.ErrMetricsNotAvailable):
		responses.WriteServiceUnavailable(w, "Metrics server not available")
	case errors.Is(err, context.DeadlineExceeded):
		responses.WriteTimeout(w, "Request timeout")
	default:
		responses.WriteInternalError(w, "Internal server error")
	}
}
