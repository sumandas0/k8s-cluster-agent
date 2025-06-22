package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	_ "github.com/sumandas0/k8s-cluster-agent/internal/core/models"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

type PodHandlers struct {
	podService core.PodService
	logger     *slog.Logger
}

func NewPodHandlers(podService core.PodService, logger *slog.Logger) *PodHandlers {
	return &PodHandlers{
		podService: podService,
		logger:     logger,
	}
}

// GetPodDescribe returns a full description of a pod
// @Summary Get pod description
// @Description Returns comprehensive pod information including status, containers, volumes, and conditions
// @Tags Pods
// @Accept json
// @Produce json
// @Param namespace path string true "Namespace name"
// @Param podName path string true "Pod name"
// @Success 200 {object} responses.SuccessResponse{data=map[string]interface{}} "Pod description"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid parameters"
// @Failure 404 {object} responses.ErrorResponse "Pod not found"
// @Failure 408 {object} responses.ErrorResponse "Request timeout"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /pods/{namespace}/{podName}/describe [get]
func (h *PodHandlers) GetPodDescribe(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")
	requestID := middleware.GetReqID(r.Context())

	if err := validatePodParams(namespace, podName); err != nil {
		h.logger.Warn("invalid pod describe request",
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	description, err := h.podService.GetPodDescription(r.Context(), namespace, podName)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to get pod description", namespace, podName)
		return
	}

	h.logger.Debug("pod describe request successful",
		"namespace", namespace,
		"pod", podName,
		"request_id", requestID,
	)

	responses.WriteJSON(w, responses.Success(description))
}

// GetPodScheduling returns detailed pod scheduling information
// @Summary Get pod scheduling information
// @Description Returns comprehensive scheduling analysis including placement decisions, failure reasons, and constraints
// @Tags Pods
// @Accept json
// @Produce json
// @Param namespace path string true "Namespace name"
// @Param podName path string true "Pod name"
// @Success 200 {object} responses.SuccessResponse{data=models.PodScheduling} "Pod scheduling information"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid parameters"
// @Failure 404 {object} responses.ErrorResponse "Pod not found"
// @Failure 408 {object} responses.ErrorResponse "Request timeout"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /pods/{namespace}/{podName}/scheduling [get]
func (h *PodHandlers) GetPodScheduling(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")
	requestID := middleware.GetReqID(r.Context())

	if err := validatePodParams(namespace, podName); err != nil {
		h.logger.Warn("invalid pod scheduling request",
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	scheduling, err := h.podService.GetPodScheduling(r.Context(), namespace, podName)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to get pod scheduling", namespace, podName)
		return
	}

	h.logger.Debug("pod scheduling request successful",
		"namespace", namespace,
		"pod", podName,
		"request_id", requestID,
	)

	responses.WriteJSON(w, responses.Success(scheduling))
}

// GetPodResources returns resource requirements and usage for a pod
// @Summary Get pod resource information
// @Description Returns detailed resource requirements (CPU, memory) for all containers in the pod
// @Tags Pods
// @Accept json
// @Produce json
// @Param namespace path string true "Namespace name"
// @Param podName path string true "Pod name"
// @Success 200 {object} responses.SuccessResponse{data=models.PodResources} "Pod resource information"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid parameters"
// @Failure 404 {object} responses.ErrorResponse "Pod not found"
// @Failure 408 {object} responses.ErrorResponse "Request timeout"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /pods/{namespace}/{podName}/resources [get]
func (h *PodHandlers) GetPodResources(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")
	requestID := middleware.GetReqID(r.Context())

	if err := validatePodParams(namespace, podName); err != nil {
		h.logger.Warn("invalid pod resources request",
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	resources, err := h.podService.GetPodResources(r.Context(), namespace, podName)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to get pod resources", namespace, podName)
		return
	}

	h.logger.Debug("pod resources request successful",
		"namespace", namespace,
		"pod", podName,
		"request_id", requestID,
	)

	responses.WriteJSON(w, responses.Success(resources))
}

// GetPodFailureEvents returns analyzed failure events for a pod
// @Summary Get pod failure events analysis
// @Description Returns categorized and analyzed failure events with insights and recommendations
// @Tags Pods
// @Accept json
// @Produce json
// @Param namespace path string true "Namespace name"
// @Param podName path string true "Pod name"
// @Success 200 {object} responses.SuccessResponse{data=models.PodFailureEvents} "Pod failure events analysis"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid parameters"
// @Failure 404 {object} responses.ErrorResponse "Pod not found"
// @Failure 408 {object} responses.ErrorResponse "Request timeout"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /pods/{namespace}/{podName}/failure-events [get]
func (h *PodHandlers) GetPodFailureEvents(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")
	requestID := middleware.GetReqID(r.Context())

	if err := validatePodParams(namespace, podName); err != nil {
		h.logger.Warn("invalid pod failure events request",
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	failureEvents, err := h.podService.GetPodFailureEvents(r.Context(), namespace, podName)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to get pod failure events", namespace, podName)
		return
	}

	h.logger.Debug("pod failure events request successful",
		"namespace", namespace,
		"pod", podName,
		"total_events", failureEvents.TotalEvents,
		"failure_events", len(failureEvents.FailureEvents),
		"critical_events", failureEvents.CriticalEvents,
		"request_id", requestID,
	)

	responses.WriteJSON(w, responses.Success(failureEvents))
}

// GetPodSchedulingExplanation returns detailed scheduling explanation
// @Summary Get detailed pod scheduling explanation
// @Description Returns Elasticsearch-style detailed explanations for pod scheduling decisions with per-node analysis
// @Tags Pods
// @Accept json
// @Produce json
// @Param namespace path string true "Namespace name"
// @Param podName path string true "Pod name"
// @Success 200 {object} responses.SuccessResponse{data=models.SchedulingExplanation} "Pod scheduling explanation"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid parameters"
// @Failure 404 {object} responses.ErrorResponse "Pod not found"
// @Failure 408 {object} responses.ErrorResponse "Request timeout"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /pods/{namespace}/{podName}/scheduling/explain [get]
func (h *PodHandlers) GetPodSchedulingExplanation(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")
	requestID := middleware.GetReqID(r.Context())

	if err := validatePodParams(namespace, podName); err != nil {
		h.logger.Warn("invalid pod scheduling explanation request",
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	explanation, err := h.podService.GetPodSchedulingExplanation(r.Context(), namespace, podName)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to get pod scheduling explanation", namespace, podName)
		return
	}

	h.logger.Debug("pod scheduling explanation request successful",
		"namespace", namespace,
		"pod", podName,
		"request_id", requestID,
	)

	responses.WriteJSON(w, responses.Success(explanation))
}

func validatePodParams(namespace, podName string) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if podName == "" {
		return fmt.Errorf("pod name is required")
	}
	return nil
}

func (h *PodHandlers) handleServiceError(w http.ResponseWriter, r *http.Request, err error, operation, namespace, podName string) {
	requestID := middleware.GetReqID(r.Context())

	switch {
	case errors.Is(err, core.ErrPodNotFound):
		h.logger.Warn("pod not found",
			"operation", operation,
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteNotFound(w, "Pod not found")
	case errors.Is(err, core.ErrNodeNotFound):
		h.logger.Warn("node not found",
			"operation", operation,
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteNotFound(w, "Node not found")
	case errors.Is(err, core.ErrMetricsNotAvailable):
		h.logger.Warn("metrics server not available",
			"operation", operation,
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteServiceUnavailable(w, "Metrics server not available")
	case errors.Is(err, context.DeadlineExceeded):
		h.logger.Warn("request timeout",
			"operation", operation,
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteTimeout(w, "Request timeout")
	default:
		h.logger.Error("internal server error",
			"operation", operation,
			"namespace", namespace,
			"pod", podName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteInternalError(w, "Internal server error")
	}
}
