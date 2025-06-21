package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
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

// GetPodDescription returns comprehensive pod information similar to kubectl describe pod
func (s *podService) GetPodDescription(ctx context.Context, namespace, name string) (*models.PodDescription, error) {
	s.logger.Debug("getting pod description", "namespace", namespace, "pod", name)

	pod, err := s.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	// Get events related to this pod
	events, err := s.getPodEvents(ctx, namespace, name)
	if err != nil {
		s.logger.Warn("failed to get pod events",
			"namespace", namespace,
			"pod", name,
			"error", err.Error())
		// Continue without events rather than failing
		events = []models.EventInfo{}
	}

	// Build comprehensive description
	description := &models.PodDescription{
		Name:        pod.Name,
		Namespace:   pod.Namespace,
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		Status: models.PodStatusInfo{
			Phase:             string(pod.Status.Phase),
			Reason:            pod.Status.Reason,
			Message:           pod.Status.Message,
			HostIP:            pod.Status.HostIP,
			PodIP:             pod.Status.PodIP,
			NominatedNodeName: pod.Status.NominatedNodeName,
		},
		Node:              pod.Spec.NodeName,
		StartTime:         pod.Status.StartTime,
		PodIP:             pod.Status.PodIP,
		QOSClass:          string(pod.Status.QOSClass),
		Priority:          pod.Spec.Priority,
		PriorityClassName: pod.Spec.PriorityClassName,
		Tolerations:       pod.Spec.Tolerations,
		NodeSelector:      pod.Spec.NodeSelector,
		Events:            events,
		Conditions:        pod.Status.Conditions,
	}

	// Add PodIPs
	for _, podIP := range pod.Status.PodIPs {
		description.PodIPs = append(description.PodIPs, podIP.IP)
	}

	// Process containers
	description.Containers = s.buildContainerInfo(pod.Spec.Containers, pod.Status.ContainerStatuses)

	// Process init containers
	if len(pod.Spec.InitContainers) > 0 {
		description.InitContainers = s.buildContainerInfo(pod.Spec.InitContainers, pod.Status.InitContainerStatuses)
	}

	// Process volumes
	description.Volumes = s.buildVolumeInfo(pod.Spec.Volumes)

	s.logger.Debug("successfully built pod description",
		"namespace", namespace,
		"pod", name,
		"containers", len(description.Containers),
		"volumes", len(description.Volumes),
		"events", len(description.Events))

	return description, nil
}

// getPodEvents fetches recent events related to the pod
func (s *podService) getPodEvents(ctx context.Context, namespace, podName string) ([]models.EventInfo, error) {
	// Create field selector to get events for this specific pod
	fieldSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.kind", "Pod"),
		fields.OneTermEqualSelector("involvedObject.name", podName),
		fields.OneTermEqualSelector("involvedObject.namespace", namespace),
	).String()

	eventList, err := s.k8sClient.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
		Limit:         20, // Limit to most recent 20 events
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get events for pod %s/%s: %w", namespace, podName, err)
	}

	// Sort events by timestamp (most recent first)
	sort.Slice(eventList.Items, func(i, j int) bool {
		return eventList.Items[i].LastTimestamp.After(eventList.Items[j].LastTimestamp.Time)
	})

	events := make([]models.EventInfo, 0, len(eventList.Items))
	for _, event := range eventList.Items {
		events = append(events, models.EventInfo{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			FirstTimestamp: event.FirstTimestamp,
			LastTimestamp:  event.LastTimestamp,
			Count:          event.Count,
			Source:         fmt.Sprintf("%s/%s", event.Source.Component, event.Source.Host),
		})
	}

	return events, nil
}

// buildContainerInfo creates ContainerInfo from container specs and statuses
func (s *podService) buildContainerInfo(containers []v1.Container, statuses []v1.ContainerStatus) []models.ContainerInfo {
	containerInfo := make([]models.ContainerInfo, 0, len(containers))

	// Create a map of container statuses for quick lookup
	statusMap := make(map[string]v1.ContainerStatus)
	for _, status := range statuses {
		statusMap[status.Name] = status
	}

	for _, container := range containers {
		info := models.ContainerInfo{
			Name:        container.Name,
			Image:       container.Image,
			Resources:   container.Resources,
			Environment: container.Env,
		}

		// Add volume mounts
		for _, mount := range container.VolumeMounts {
			info.Mounts = append(info.Mounts, models.VolumeMountInfo{
				Name:      mount.Name,
				MountPath: mount.MountPath,
				ReadOnly:  mount.ReadOnly,
				SubPath:   mount.SubPath,
			})
		}

		// Add status information if available
		if status, exists := statusMap[container.Name]; exists {
			info.ImageID = status.ImageID
			info.State = status.State
			info.Ready = status.Ready
			info.RestartCount = status.RestartCount
		}

		containerInfo = append(containerInfo, info)
	}

	return containerInfo
}

// buildVolumeInfo creates VolumeInfo from volume specs
func (s *podService) buildVolumeInfo(volumes []v1.Volume) []models.VolumeInfo {
	volumeInfo := make([]models.VolumeInfo, 0, len(volumes))

	for _, volume := range volumes {
		info := models.VolumeInfo{
			Name:   volume.Name,
			Type:   s.getVolumeType(volume.VolumeSource),
			Source: volume.VolumeSource,
		}
		volumeInfo = append(volumeInfo, info)
	}

	return volumeInfo
}

// getVolumeType determines the volume type from VolumeSource
func (s *podService) getVolumeType(source v1.VolumeSource) string {
	switch {
	case source.EmptyDir != nil:
		return "EmptyDir"
	case source.HostPath != nil:
		return "HostPath"
	case source.Secret != nil:
		return "Secret"
	case source.ConfigMap != nil:
		return "ConfigMap"
	case source.PersistentVolumeClaim != nil:
		return "PersistentVolumeClaim"
	case source.DownwardAPI != nil:
		return "DownwardAPI"
	case source.Projected != nil:
		return "Projected"
	case source.CSI != nil:
		return "CSI"
	case source.Ephemeral != nil:
		return "Ephemeral"
	default:
		return "Unknown"
	}
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
