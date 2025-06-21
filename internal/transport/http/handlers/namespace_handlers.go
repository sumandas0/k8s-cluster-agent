package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

type NamespaceHandlers struct {
	namespaceService core.NamespaceService
	logger           *slog.Logger
}

func NewNamespaceHandlers(namespaceService core.NamespaceService, logger *slog.Logger) *NamespaceHandlers {
	return &NamespaceHandlers{
		namespaceService: namespaceService,
		logger:           logger,
	}
}

func (h *NamespaceHandlers) GetNamespaceErrors(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	requestID := middleware.GetReqID(r.Context())

	if err := validateNamespace(namespace); err != nil {
		h.logger.Warn("invalid namespace error request",
			"namespace", namespace,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	report, err := h.namespaceService.GetNamespaceErrors(r.Context(), namespace)
	if err != nil {
		h.logger.Error("failed to get namespace errors",
			"namespace", namespace,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteInternalError(w, "Failed to analyze namespace errors")
		return
	}

	response := responses.SuccessResponse{
		Data: report,
		Metadata: responses.Metadata{
			RequestID: requestID,
			Timestamp: report.AnalysisTime,
		},
	}
	responses.WriteJSON(w, response)

	h.logger.Info("namespace error analysis served",
		"namespace", namespace,
		"total_pods", report.TotalPodsAnalyzed,
		"problematic_pods", report.ProblematicPodsCount,
		"request_id", requestID,
	)
}

func validateNamespace(namespace string) error {
	if namespace == "" {
		return errors.New("namespace is required")
	}
	if len(namespace) > 253 {
		return errors.New("namespace name is too long")
	}
	return nil
}

