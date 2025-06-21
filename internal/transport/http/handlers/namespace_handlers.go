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

// NamespaceHandlers contains namespace-related HTTP handlers
type NamespaceHandlers struct {
	namespaceService core.NamespaceService
	logger           *slog.Logger
}

// NewNamespaceHandlers creates a new NamespaceHandlers instance
func NewNamespaceHandlers(namespaceService core.NamespaceService, logger *slog.Logger) *NamespaceHandlers {
	return &NamespaceHandlers{
		namespaceService: namespaceService,
		logger:           logger,
	}
}

// GetNamespaceErrors handles GET /api/v1/namespace/{namespace}/error
func (h *NamespaceHandlers) GetNamespaceErrors(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	requestID := middleware.GetReqID(r.Context())

	// Validate input
	if err := validateNamespace(namespace); err != nil {
		h.logger.Warn("invalid namespace error request",
			"namespace", namespace,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	// Get namespace error report
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

	// Write successful response
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

// validateNamespace validates namespace parameter
func validateNamespace(namespace string) error {
	if namespace == "" {
		return errors.New("namespace is required")
	}
	if len(namespace) > 253 {
		return errors.New("namespace name is too long")
	}
	// Note: Full RFC 1123 validation could be added here
	return nil
}

