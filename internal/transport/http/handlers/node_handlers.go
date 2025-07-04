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

// GetNodeUtilization returns resource utilization metrics for a node
// @Summary Get node utilization metrics
// @Description Returns CPU and memory utilization metrics for the specified node (requires metrics server)
// @Tags Nodes
// @Accept json
// @Produce json
// @Param nodeName path string true "Node name"
// @Success 200 {object} responses.SuccessResponse{data=models.NodeUtilization} "Node utilization metrics"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid parameters"
// @Failure 404 {object} responses.ErrorResponse "Node not found"
// @Failure 408 {object} responses.ErrorResponse "Request timeout"
// @Failure 503 {object} responses.ErrorResponse "Metrics server not available"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /nodes/{nodeName}/utilization [get]
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
