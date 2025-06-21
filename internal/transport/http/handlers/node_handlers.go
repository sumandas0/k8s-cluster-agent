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
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

// NodeHandlers contains node-related HTTP handlers
type NodeHandlers struct {
	nodeService core.NodeService
	logger      *slog.Logger
}

// NewNodeHandlers creates a new NodeHandlers instance
func NewNodeHandlers(nodeService core.NodeService, logger *slog.Logger) *NodeHandlers {
	return &NodeHandlers{
		nodeService: nodeService,
		logger:      logger,
	}
}

// GetNodeUtilization handles GET /api/v1/nodes/{nodeName}/utilization
func (h *NodeHandlers) GetNodeUtilization(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "nodeName")
	requestID := middleware.GetReqID(r.Context())

	// Validate input
	if err := validateNodeParams(nodeName); err != nil {
		h.logger.Warn("invalid node utilization request",
			"node", nodeName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	// Get node utilization
	utilization, err := h.nodeService.GetNodeUtilization(r.Context(), nodeName)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to get node utilization", nodeName)
		return
	}

	h.logger.Debug("node utilization request successful",
		"node", nodeName,
		"request_id", requestID,
	)

	// Write response
	responses.WriteJSON(w, responses.Success(utilization))
}

// validateNodeParams validates node-related request parameters
func validateNodeParams(nodeName string) error {
	if nodeName == "" {
		return fmt.Errorf("node name is required")
	}
	return nil
}

// handleServiceError maps service errors to HTTP responses and logs detailed error information
func (h *NodeHandlers) handleServiceError(w http.ResponseWriter, r *http.Request, err error, operation, nodeName string) {
	requestID := middleware.GetReqID(r.Context())

	switch {
	case errors.Is(err, core.ErrPodNotFound):
		h.logger.Warn("pod not found",
			"operation", operation,
			"node", nodeName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteNotFound(w, "Pod not found")
	case errors.Is(err, core.ErrNodeNotFound):
		h.logger.Warn("node not found",
			"operation", operation,
			"node", nodeName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteNotFound(w, "Node not found")
	case errors.Is(err, core.ErrMetricsNotAvailable):
		h.logger.Warn("metrics server not available",
			"operation", operation,
			"node", nodeName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteServiceUnavailable(w, "Metrics server not available")
	case errors.Is(err, context.DeadlineExceeded):
		h.logger.Warn("request timeout",
			"operation", operation,
			"node", nodeName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteTimeout(w, "Request timeout")
	default:
		// Log the actual error for internal debugging but don't expose to client
		h.logger.Error("internal server error",
			"operation", operation,
			"node", nodeName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteInternalError(w, "Internal server error")
	}
}
