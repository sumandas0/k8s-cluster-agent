package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

type ClusterIssuesHandler struct {
	service core.ClusterIssuesService
	logger  *slog.Logger
}

func NewClusterIssuesHandler(service core.ClusterIssuesService, logger *slog.Logger) *ClusterIssuesHandler {
	return &ClusterIssuesHandler{
		service: service,
		logger:  logger.With(slog.String("handler", "cluster_issues")),
	}
}

func (h *ClusterIssuesHandler) GetClusterIssues(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetReqID(r.Context())

	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "all"
	}

	severity := r.URL.Query().Get("severity")

	clusterIssues, err := h.service.GetClusterIssues(r.Context(), namespace, severity)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to get cluster issues", namespace, severity)
		return
	}

	h.logger.Debug("cluster issues request successful",
		slog.String("namespace", namespace),
		slog.String("severity", severity),
		slog.String("request_id", requestID))

	responses.WriteJSON(w, responses.Success(clusterIssues))
}

func (h *ClusterIssuesHandler) handleServiceError(w http.ResponseWriter, r *http.Request, err error, operation, namespace, severity string) {
	requestID := middleware.GetReqID(r.Context())

	switch {
	case err == context.DeadlineExceeded:
		h.logger.Warn("request timeout",
			slog.String("operation", operation),
			slog.String("namespace", namespace),
			slog.String("severity", severity),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID))
		responses.WriteTimeout(w, "Request timeout")
	default:
		h.logger.Error("internal server error",
			slog.String("operation", operation),
			slog.String("namespace", namespace),
			slog.String("severity", severity),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID))
		responses.WriteInternalError(w, "Internal server error")
	}
}

