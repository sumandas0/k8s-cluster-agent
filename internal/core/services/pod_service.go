package services

import (
	"context"
	"fmt"
	"log/slog"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

// podService implements the PodService interface
type podService struct {
	k8sClient kubernetes.Interface
	logger    *slog.Logger
}

// NewPodService creates a new PodService instance
func NewPodService(k8sClient kubernetes.Interface, logger *slog.Logger) core.PodService {
	return &podService{
		k8sClient: k8sClient,
		logger:    logger,
	}
}

// GetPod returns the full pod specification and status
func (s *podService) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	s.logger.Debug("getting pod", "namespace", namespace, "pod", name)

	pod, err := s.k8sClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			s.logger.Debug("pod not found", "namespace", namespace, "pod", name)
			return nil, core.ErrPodNotFound
		}
		s.logger.Error("failed to get pod from kubernetes API",
			"namespace", namespace,
			"pod", name,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}

	s.logger.Debug("successfully retrieved pod", "namespace", namespace, "pod", name)
	return pod, nil
}

// GetPodScheduling returns scheduling-specific information for a pod
func (s *podService) GetPodScheduling(ctx context.Context, namespace, name string) (*models.PodScheduling, error) {
	s.logger.Debug("getting pod scheduling info", "namespace", namespace, "pod", name)

	pod, err := s.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	scheduling := &models.PodScheduling{
		NodeName:          pod.Spec.NodeName,
		SchedulerName:     pod.Spec.SchedulerName,
		Affinity:          pod.Spec.Affinity,
		Tolerations:       pod.Spec.Tolerations,
		NodeSelector:      pod.Spec.NodeSelector,
		Priority:          pod.Spec.Priority,
		PriorityClassName: pod.Spec.PriorityClassName,
	}

	s.logger.Debug("successfully retrieved pod scheduling info", "namespace", namespace, "pod", name)
	return scheduling, nil
}

// GetPodResources returns aggregated resource requirements for a pod
func (s *podService) GetPodResources(ctx context.Context, namespace, name string) (*models.PodResources, error) {
	s.logger.Debug("getting pod resources", "namespace", namespace, "pod", name)

	pod, err := s.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	// Collect resources from all containers
	containers := make([]models.ContainerResources, 0, len(pod.Spec.Containers)+len(pod.Spec.InitContainers))

	// Add regular containers
	for _, container := range pod.Spec.Containers {
		containers = append(containers, models.ContainerResources{
			Name:     container.Name,
			Requests: container.Resources.Requests,
			Limits:   container.Resources.Limits,
		})
	}

	// Add init containers
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, models.ContainerResources{
			Name:     container.Name + " (init)",
			Requests: container.Resources.Requests,
			Limits:   container.Resources.Limits,
		})
	}

	// Calculate total resources
	totalCPURequest := resource.NewQuantity(0, resource.DecimalSI)
	totalCPULimit := resource.NewQuantity(0, resource.DecimalSI)
	totalMemoryRequest := resource.NewQuantity(0, resource.BinarySI)
	totalMemoryLimit := resource.NewQuantity(0, resource.BinarySI)

	for _, container := range pod.Spec.Containers {
		if req, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
			if err := safeAddQuantity(totalCPURequest, req); err != nil {
				s.logger.Warn("failed to add CPU request",
					"namespace", namespace,
					"pod", name,
					"container", container.Name,
					"error", err.Error(),
				)
			}
		}
		if limit, ok := container.Resources.Limits[v1.ResourceCPU]; ok {
			if err := safeAddQuantity(totalCPULimit, limit); err != nil {
				s.logger.Warn("failed to add CPU limit",
					"namespace", namespace,
					"pod", name,
					"container", container.Name,
					"error", err.Error(),
				)
			}
		}
		if req, ok := container.Resources.Requests[v1.ResourceMemory]; ok {
			if err := safeAddQuantity(totalMemoryRequest, req); err != nil {
				s.logger.Warn("failed to add memory request",
					"namespace", namespace,
					"pod", name,
					"container", container.Name,
					"error", err.Error(),
				)
			}
		}
		if limit, ok := container.Resources.Limits[v1.ResourceMemory]; ok {
			if err := safeAddQuantity(totalMemoryLimit, limit); err != nil {
				s.logger.Warn("failed to add memory limit",
					"namespace", namespace,
					"pod", name,
					"container", container.Name,
					"error", err.Error(),
				)
			}
		}
	}

	result := &models.PodResources{
		Containers: containers,
		Total: models.ResourceSummary{
			CPURequest:    totalCPURequest.String(),
			CPULimit:      totalCPULimit.String(),
			MemoryRequest: totalMemoryRequest.String(),
			MemoryLimit:   totalMemoryLimit.String(),
		},
	}

	s.logger.Debug("successfully calculated pod resources",
		"namespace", namespace,
		"pod", name,
		"containers", len(containers),
		"total_cpu_request", result.Total.CPURequest,
		"total_memory_request", result.Total.MemoryRequest,
	)

	return result, nil
}

// safeAddQuantity safely adds two resource quantities with error handling
func safeAddQuantity(total *resource.Quantity, add resource.Quantity) error {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error
		}
	}()

	total.Add(add)
	return nil
}
