package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

// nodeService implements the NodeService interface
type nodeService struct {
	k8sClient     kubernetes.Interface
	metricsClient metricsclientset.Interface
	logger        *slog.Logger
}

// NewNodeService creates a new NodeService instance
func NewNodeService(k8sClient kubernetes.Interface, metricsClient metricsclientset.Interface, logger *slog.Logger) core.NodeService {
	return &nodeService{
		k8sClient:     k8sClient,
		metricsClient: metricsClient,
		logger:        logger,
	}
}

// GetNodeUtilization returns the current resource utilization for a node
func (s *nodeService) GetNodeUtilization(ctx context.Context, nodeName string) (*models.NodeUtilization, error) {
	// First, check if the node exists
	node, err := s.k8sClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, core.ErrNodeNotFound
		}
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	// Check if metrics are available
	if !s.checkMetricsAvailable(ctx) {
		return nil, core.ErrMetricsNotAvailable
	}

	// Get node metrics
	nodeMetrics, err := s.metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, core.ErrMetricsNotAvailable
		}
		return nil, fmt.Errorf("failed to get metrics for node %s: %w", nodeName, err)
	}

	// Get capacity from the node
	cpuCapacity := node.Status.Capacity[v1.ResourceCPU]
	memoryCapacity := node.Status.Capacity[v1.ResourceMemory]

	// Get usage from metrics
	cpuUsage := nodeMetrics.Usage[v1.ResourceCPU]
	memoryUsage := nodeMetrics.Usage[v1.ResourceMemory]

	// Calculate percentages
	cpuPercentage := calculatePercentage(&cpuUsage, &cpuCapacity)
	memoryPercentage := calculatePercentage(&memoryUsage, &memoryCapacity)

	return &models.NodeUtilization{
		NodeName:         nodeName,
		CPUUsage:         cpuUsage.String(),
		CPUCapacity:      cpuCapacity.String(),
		CPUPercentage:    cpuPercentage,
		MemoryUsage:      memoryUsage.String(),
		MemoryCapacity:   memoryCapacity.String(),
		MemoryPercentage: memoryPercentage,
		Timestamp:        time.Now(),
	}, nil
}

// checkMetricsAvailable checks if the metrics server is available
func (s *nodeService) checkMetricsAvailable(ctx context.Context) bool {
	if s.metricsClient == nil {
		return false
	}

	// Try to list metrics to check availability
	_, err := s.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{Limit: 1})

	return err == nil
}

// calculatePercentage calculates the percentage of usage vs capacity
func calculatePercentage(usage, capacity *resource.Quantity) float64 {
	if capacity.IsZero() {
		return 0
	}

	// Convert to float64 for percentage calculation
	usageFloat := float64(usage.MilliValue())
	capacityFloat := float64(capacity.MilliValue())

	if capacityFloat == 0 {
		return 0
	}

	return (usageFloat / capacityFloat) * 100
}
