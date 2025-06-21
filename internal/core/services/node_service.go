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

type nodeService struct {
	k8sClient     kubernetes.Interface
	metricsClient metricsclientset.Interface
	logger        *slog.Logger
}

func NewNodeService(k8sClient kubernetes.Interface, metricsClient metricsclientset.Interface, logger *slog.Logger) core.NodeService {
	return &nodeService{
		k8sClient:     k8sClient,
		metricsClient: metricsClient,
		logger:        logger,
	}
}

func (s *nodeService) GetNodeUtilization(ctx context.Context, nodeName string) (*models.NodeUtilization, error) {
	s.logger.Debug("getting node utilization", "node", nodeName)

	node, err := s.k8sClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			s.logger.Debug("node not found", "node", nodeName)
			return nil, core.ErrNodeNotFound
		}
		s.logger.Error("failed to get node from kubernetes API",
			"node", nodeName,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	if !s.checkMetricsAvailable(ctx) {
		s.logger.Warn("metrics server not available", "node", nodeName)
		return nil, core.ErrMetricsNotAvailable
	}

	nodeMetrics, err := s.metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			s.logger.Warn("node metrics not found", "node", nodeName)
			return nil, core.ErrMetricsNotAvailable
		}
		s.logger.Error("failed to get node metrics",
			"node", nodeName,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to get metrics for node %s: %w", nodeName, err)
	}

	cpuCapacity := node.Status.Capacity[v1.ResourceCPU]
	memoryCapacity := node.Status.Capacity[v1.ResourceMemory]

	cpuUsage := nodeMetrics.Usage[v1.ResourceCPU]
	memoryUsage := nodeMetrics.Usage[v1.ResourceMemory]

	cpuPercentage := calculatePercentage(&cpuUsage, &cpuCapacity)
	memoryPercentage := calculatePercentage(&memoryUsage, &memoryCapacity)

	result := &models.NodeUtilization{
		NodeName:         nodeName,
		CPUUsage:         cpuUsage.String(),
		CPUCapacity:      cpuCapacity.String(),
		CPUPercentage:    cpuPercentage,
		MemoryUsage:      memoryUsage.String(),
		MemoryCapacity:   memoryCapacity.String(),
		MemoryPercentage: memoryPercentage,
		Timestamp:        time.Now(),
	}

	s.logger.Debug("successfully retrieved node utilization",
		"node", nodeName,
		"cpu_percentage", cpuPercentage,
		"memory_percentage", memoryPercentage,
	)

	return result, nil
}

func (s *nodeService) checkMetricsAvailable(ctx context.Context) bool {
	if s.metricsClient == nil {
		s.logger.Debug("metrics client is nil")
		return false
	}

	_, err := s.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		s.logger.Debug("metrics server check failed", "error", err.Error())
		return false
	}

	return true
}

func calculatePercentage(usage, capacity *resource.Quantity) float64 {
	if usage == nil || capacity == nil {
		return 0
	}

	if capacity.IsZero() {
		return 0
	}

	usageFloat := float64(usage.MilliValue())
	capacityFloat := float64(capacity.MilliValue())

	if capacityFloat == 0 {
		return 0
	}

	percentage := (usageFloat / capacityFloat) * 100

	if percentage < 0 {
		return 0
	}
	if percentage > 100 {
		return 100
	}

	return percentage
}
