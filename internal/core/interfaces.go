package core

import (
	"context"

	v1 "k8s.io/api/core/v1"

	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

type PodService interface {
	GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error)

	GetPodDescription(ctx context.Context, namespace, name string) (*models.PodDescription, error)

	GetPodScheduling(ctx context.Context, namespace, name string) (*models.PodScheduling, error)

	GetPodResources(ctx context.Context, namespace, name string) (*models.PodResources, error)

	GetPodFailureEvents(ctx context.Context, namespace, name string) (*models.PodFailureEvents, error)

	GetPodSchedulingExplanation(ctx context.Context, namespace, name string) (*models.SchedulingExplanation, error)
}

type NodeService interface {
	GetNodeUtilization(ctx context.Context, nodeName string) (*models.NodeUtilization, error)
}

type NamespaceService interface {
	GetNamespaceErrors(ctx context.Context, namespace string) (*models.NamespaceErrorReport, error)
}

type HealthScoreService interface {
	CalculateHealthScore(ctx context.Context, namespace, podName string) (*models.PodHealthScore, error)
}

type ClusterIssuesService interface {
	GetClusterIssues(ctx context.Context, namespace string, severityFilter string) (*models.ClusterIssues, error)
}

type Services struct {
	Pod           PodService
	Node          NodeService
	Namespace     NamespaceService
	HealthScore   HealthScoreService
	ClusterIssues ClusterIssuesService
}
