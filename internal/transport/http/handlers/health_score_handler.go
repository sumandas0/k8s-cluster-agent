package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

type HealthScoreHandler struct {
	service core.HealthScoreService
	logger  *slog.Logger
}

func NewHealthScoreHandler(service core.HealthScoreService, logger *slog.Logger) *HealthScoreHandler {
	return &HealthScoreHandler{
		service: service,
		logger:  logger.With(slog.String("handler", "health_score")),
	}
}

func (h *HealthScoreHandler) GetPodHealthScore(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	podName := chi.URLParam(r, "podName")
	requestID := middleware.GetReqID(r.Context())

	if namespace == "" || podName == "" {
		h.logger.Warn("invalid health score request",
			slog.String("namespace", namespace),
			slog.String("pod", podName),
			slog.String("error", "namespace and podName are required"),
			slog.String("request_id", requestID))
		responses.WriteBadRequest(w, errors.New("namespace and podName are required"))
		return
	}

	healthScore, err := h.service.CalculateHealthScore(r.Context(), namespace, podName)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to calculate health score", namespace, podName)
		return
	}

	h.logger.Debug("health score request successful",
		slog.String("namespace", namespace),
		slog.String("pod", podName),
		slog.String("request_id", requestID))

	responses.WriteJSON(w, responses.Success(healthScore))
}

func (h *HealthScoreHandler) handleServiceError(w http.ResponseWriter, r *http.Request, err error, operation, namespace, podName string) {
	requestID := middleware.GetReqID(r.Context())

	switch {
	case errors.Is(err, core.ErrPodNotFound):
		h.logger.Warn("pod not found",
			slog.String("operation", operation),
			slog.String("namespace", namespace),
			slog.String("pod", podName),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID))
		responses.WriteNotFound(w, "Pod not found")
	case errors.Is(err, context.DeadlineExceeded):
		h.logger.Warn("request timeout",
			slog.String("operation", operation),
			slog.String("namespace", namespace),
			slog.String("pod", podName),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID))
		responses.WriteTimeout(w, "Request timeout")
	default:
		h.logger.Error("internal server error",
			slog.String("operation", operation),
			slog.String("namespace", namespace),
			slog.String("pod", podName),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID))
		responses.WriteInternalError(w, "Internal server error")
	}
}

