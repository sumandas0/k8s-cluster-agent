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

type NodeHandlers struct {
	nodeService core.NodeService
	logger      *slog.Logger
}

func NewNodeHandlers(nodeService core.NodeService, logger *slog.Logger) *NodeHandlers {
	return &NodeHandlers{
		nodeService: nodeService,
		logger:      logger,
	}
}

func (h *NodeHandlers) GetNodeUtilization(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "nodeName")
	requestID := middleware.GetReqID(r.Context())

	if err := validateNodeParams(nodeName); err != nil {
		h.logger.Warn("invalid node utilization request",
			"node", nodeName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteBadRequest(w, err)
		return
	}

	utilization, err := h.nodeService.GetNodeUtilization(r.Context(), nodeName)
	if err != nil {
		h.handleServiceError(w, r, err, "failed to get node utilization", nodeName)
		return
	}

	h.logger.Debug("node utilization request successful",
		"node", nodeName,
		"request_id", requestID,
	)

	responses.WriteJSON(w, responses.Success(utilization))
}

func validateNodeParams(nodeName string) error {
	if nodeName == "" {
		return fmt.Errorf("node name is required")
	}
	return nil
}

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
		h.logger.Error("internal server error",
			"operation", operation,
			"node", nodeName,
			"error", err.Error(),
			"request_id", requestID,
		)
		responses.WriteInternalError(w, "Internal server error")
	}
}
