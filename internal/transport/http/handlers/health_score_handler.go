package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	_ "github.com/sumandas0/k8s-cluster-agent/internal/core/models"
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

// GetPodHealthScore calculates and returns a comprehensive health score for a pod
// @Summary Get pod health score
// @Description Returns a health score (0-100) with detailed component analysis including restarts, container states, events, and uptime
// @Tags Pods
// @Accept json
// @Produce json
// @Param namespace path string true "Namespace name"
// @Param podName path string true "Pod name"
// @Success 200 {object} responses.SuccessResponse{data=models.PodHealthScore} "Pod health score with detailed analysis"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid parameters"
// @Failure 404 {object} responses.ErrorResponse "Pod not found"
// @Failure 408 {object} responses.ErrorResponse "Request timeout"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /pods/{namespace}/{podName}/health-score [get]
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
