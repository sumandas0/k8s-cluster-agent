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

	// GetPodDescription returns comprehensive pod information similar to kubectl describe pod.
	// Returns ErrNotFound if the pod doesn't exist.
	GetPodDescription(ctx context.Context, namespace, name string) (*models.PodDescription, error)

	// GetPodScheduling returns scheduling-specific information for a pod.
	// Returns ErrNotFound if the pod doesn't exist.
	GetPodScheduling(ctx context.Context, namespace, name string) (*models.PodScheduling, error)

	// GetPodResources returns aggregated resource requirements for a pod.
	// Returns ErrNotFound if the pod doesn't exist.
	GetPodResources(ctx context.Context, namespace, name string) (*models.PodResources, error)

	// GetPodFailureEvents returns analyzed failure events for a pod.
	// Returns ErrNotFound if the pod doesn't exist.
	GetPodFailureEvents(ctx context.Context, namespace, name string) (*models.PodFailureEvents, error)
}

// NodeService provides operations for querying node information
// from the Kubernetes API, including metrics when available.
type NodeService interface {
	// GetNodeUtilization returns the current resource utilization for a node.
	// Returns ErrNotFound if the node doesn't exist.
	// Returns ErrMetricsNotAvailable if metrics server is not available.
	GetNodeUtilization(ctx context.Context, nodeName string) (*models.NodeUtilization, error)
}

// NamespaceService provides operations for namespace-level analysis
type NamespaceService interface {
	// GetNamespaceErrors analyzes all pods in a namespace for issues.
	// Only analyzes pods owned by Deployments and StatefulSets.
	// Returns a comprehensive report of problematic pods and their issues.
	GetNamespaceErrors(ctx context.Context, namespace string) (*models.NamespaceErrorReport, error)
}

// Services aggregates all service interfaces for easy dependency injection
type Services struct {
	Pod       PodService
	Node      NodeService
	Namespace NamespaceService
}
