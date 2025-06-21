package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

// NodeHandlers contains node-related HTTP handlers
type NodeHandlers struct {
	nodeService core.NodeService
}

// NewNodeHandlers creates a new NodeHandlers instance
func NewNodeHandlers(nodeService core.NodeService) *NodeHandlers {
	return &NodeHandlers{
		nodeService: nodeService,
	}
}

// GetNodeUtilization handles GET /api/v1/nodes/{nodeName}/utilization
func (h *NodeHandlers) GetNodeUtilization(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "nodeName")

	// Validate input
	if err := validateNodeParams(nodeName); err != nil {
		responses.WriteBadRequest(w, err)
		return
	}

	// Get node utilization
	utilization, err := h.nodeService.GetNodeUtilization(r.Context(), nodeName)
	if err != nil {
		handleServiceError(w, err)
		return
	}

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
