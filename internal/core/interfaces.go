package core

import (
	"context"

	v1 "k8s.io/api/core/v1"

	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

// PodService provides operations for querying pod information
// from the Kubernetes API. It handles both standard pod data
// and metrics when available.
type PodService interface {
	// GetPod returns the full pod specification and status.
	// Returns ErrNotFound if the pod doesn't exist.
	GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error)

	// GetPodScheduling returns scheduling-specific information for a pod.
	// Returns ErrNotFound if the pod doesn't exist.
	GetPodScheduling(ctx context.Context, namespace, name string) (*models.PodScheduling, error)

	// GetPodResources returns aggregated resource requirements for a pod.
	// Returns ErrNotFound if the pod doesn't exist.
	GetPodResources(ctx context.Context, namespace, name string) (*models.PodResources, error)
}

// NodeService provides operations for querying node information
// from the Kubernetes API, including metrics when available.
type NodeService interface {
	// GetNodeUtilization returns the current resource utilization for a node.
	// Returns ErrNotFound if the node doesn't exist.
	// Returns ErrMetricsNotAvailable if metrics server is not available.
	GetNodeUtilization(ctx context.Context, nodeName string) (*models.NodeUtilization, error)
}

// Services aggregates all service interfaces for easy dependency injection
type Services struct {
	Pod  PodService
	Node NodeService
}
