package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

const (
	SchedulingStatusScheduled = "Scheduled"
	SchedulingStatusPending   = "Pending"
	SchedulingStatusFailed    = "Failed"
)

type podService struct {
	k8sClient kubernetes.Interface
	logger    *slog.Logger
}

func NewPodService(k8sClient kubernetes.Interface, logger *slog.Logger) core.PodService {
	return &podService{
		k8sClient: k8sClient,
		logger:    logger,
	}
}

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
		Conditions:        pod.Status.Conditions,
	}

	if pod.Spec.NodeName != "" {
		scheduling.Status = SchedulingStatusScheduled
	} else if pod.Status.Phase == v1.PodPending {
		scheduling.Status = SchedulingStatusPending
	} else {
		scheduling.Status = SchedulingStatusFailed
	}

	events, err := s.getSchedulingEvents(ctx, namespace, name)
	if err != nil {
		s.logger.Warn("failed to get scheduling events",
			"namespace", namespace,
			"pod", name,
			"error", err.Error())
	} else {
		scheduling.Events = events
	}

	if scheduling.Status == SchedulingStatusScheduled && pod.Spec.NodeName != "" {
		node, err := s.k8sClient.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
		if err != nil {
			s.logger.Warn("failed to get node for scheduling analysis",
				"node", pod.Spec.NodeName,
				"error", err.Error())
		} else {
			scheduling.SchedulingDecisions = s.analyzeSchedulingDecision(pod, node)
		}
	}

	if scheduling.Status == SchedulingStatusPending {
		unschedulableNodes, err := s.analyzeUnschedulableNodes(ctx, pod)
		if err != nil {
			s.logger.Warn("failed to analyze unschedulable nodes",
				"namespace", namespace,
				"pod", name,
				"error", err.Error())
		} else {
			scheduling.UnschedulableNodes = unschedulableNodes

			scheduling.FailureSummary = s.aggregateFailureCategories(unschedulableNodes, scheduling.Events)

			categorySet := make(map[models.SchedulingFailureCategory]bool)
			for _, summary := range scheduling.FailureSummary {
				categorySet[summary.Category] = true
			}
			scheduling.FailureCategories = make([]models.SchedulingFailureCategory, 0, len(categorySet))
			for cat := range categorySet {
				scheduling.FailureCategories = append(scheduling.FailureCategories, cat)
			}
		}
	}

	s.logger.Debug("successfully retrieved enhanced pod scheduling info",
		"namespace", namespace,
		"pod", name,
		"status", scheduling.Status,
		"failureCategories", scheduling.FailureCategories)
	return scheduling, nil
}

func (s *podService) GetPodResources(ctx context.Context, namespace, name string) (*models.PodResources, error) {
	s.logger.Debug("getting pod resources", "namespace", namespace, "pod", name)

	pod, err := s.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	containers := make([]models.ContainerResources, 0, len(pod.Spec.Containers)+len(pod.Spec.InitContainers))

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		containers = append(containers, models.ContainerResources{
			Name:     container.Name,
			Requests: container.Resources.Requests,
			Limits:   container.Resources.Limits,
		})
	}

	for i := range pod.Spec.InitContainers {
		container := &pod.Spec.InitContainers[i]
		containers = append(containers, models.ContainerResources{
			Name:     container.Name + " (init)",
			Requests: container.Resources.Requests,
			Limits:   container.Resources.Limits,
		})
	}

	totalCPURequest := resource.NewQuantity(0, resource.DecimalSI)
	totalCPULimit := resource.NewQuantity(0, resource.DecimalSI)
	totalMemoryRequest := resource.NewQuantity(0, resource.BinarySI)
	totalMemoryLimit := resource.NewQuantity(0, resource.BinarySI)

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
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

func (s *podService) GetPodDescription(ctx context.Context, namespace, name string) (*models.PodDescription, error) {
	s.logger.Debug("getting pod description", "namespace", namespace, "pod", name)

	pod, err := s.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	events, err := s.getPodEvents(ctx, namespace, name)
	if err != nil {
		s.logger.Warn("failed to get pod events",
			"namespace", namespace,
			"pod", name,
			"error", err.Error())
		events = []models.EventInfo{}
	}

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

	for _, podIP := range pod.Status.PodIPs {
		description.PodIPs = append(description.PodIPs, podIP.IP)
	}

	description.Containers = s.buildContainerInfo(pod.Spec.Containers, pod.Status.ContainerStatuses)

	if len(pod.Spec.InitContainers) > 0 {
		description.InitContainers = s.buildContainerInfo(pod.Spec.InitContainers, pod.Status.InitContainerStatuses)
	}

	description.Volumes = s.buildVolumeInfo(pod.Spec.Volumes)

	s.logger.Debug("successfully built pod description",
		"namespace", namespace,
		"pod", name,
		"containers", len(description.Containers),
		"volumes", len(description.Volumes),
		"events", len(description.Events))

	return description, nil
}

func (s *podService) getPodEvents(ctx context.Context, namespace, podName string) ([]models.EventInfo, error) {
	fieldSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.kind", "Pod"),
		fields.OneTermEqualSelector("involvedObject.name", podName),
		fields.OneTermEqualSelector("involvedObject.namespace", namespace),
	).String()

	eventList, err := s.k8sClient.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
		Limit:         20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get events for pod %s/%s: %w", namespace, podName, err)
	}

	sort.Slice(eventList.Items, func(i, j int) bool {
		return eventList.Items[i].LastTimestamp.After(eventList.Items[j].LastTimestamp.Time)
	})

	events := make([]models.EventInfo, 0, len(eventList.Items))
	for i := range eventList.Items {
		event := &eventList.Items[i]
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

func (s *podService) buildContainerInfo(containers []v1.Container, statuses []v1.ContainerStatus) []models.ContainerInfo {
	containerInfo := make([]models.ContainerInfo, 0, len(containers))

	statusMap := make(map[string]v1.ContainerStatus)
	for i := range statuses {
		statusMap[statuses[i].Name] = statuses[i]
	}

	for i := range containers {
		container := &containers[i]
		info := models.ContainerInfo{
			Name:        container.Name,
			Image:       container.Image,
			Resources:   container.Resources,
			Environment: container.Env,
		}

		for _, mount := range container.VolumeMounts {
			info.Mounts = append(info.Mounts, models.VolumeMountInfo{
				Name:      mount.Name,
				MountPath: mount.MountPath,
				ReadOnly:  mount.ReadOnly,
				SubPath:   mount.SubPath,
			})
		}

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

func (s *podService) buildVolumeInfo(volumes []v1.Volume) []models.VolumeInfo {
	volumeInfo := make([]models.VolumeInfo, 0, len(volumes))

	for i := range volumes {
		volume := &volumes[i]
		info := models.VolumeInfo{
			Name:   volume.Name,
			Type:   s.getVolumeType(&volume.VolumeSource),
			Source: volume.VolumeSource,
		}
		volumeInfo = append(volumeInfo, info)
	}

	return volumeInfo
}

const (
	VolumeTypeEmptyDir    = "EmptyDir"
	VolumeTypeHostPath    = "HostPath"
	VolumeTypeSecret      = "Secret"
	VolumeTypeConfigMap   = "ConfigMap"
	VolumeTypePVC         = "PersistentVolumeClaim"
	VolumeTypeDownwardAPI = "DownwardAPI"
	VolumeTypeProjected   = "Projected"
	VolumeTypeCSI         = "CSI"
	VolumeTypeEphemeral   = "Ephemeral"
	VolumeTypeUnknown     = "Unknown"
)

func (s *podService) getVolumeType(source *v1.VolumeSource) string {
	switch {
	case source.EmptyDir != nil:
		return VolumeTypeEmptyDir
	case source.HostPath != nil:
		return VolumeTypeHostPath
	case source.Secret != nil:
		return VolumeTypeSecret
	case source.ConfigMap != nil:
		return VolumeTypeConfigMap
	case source.PersistentVolumeClaim != nil:
		return VolumeTypePVC
	case source.DownwardAPI != nil:
		return VolumeTypeDownwardAPI
	case source.Projected != nil:
		return VolumeTypeProjected
	case source.CSI != nil:
		return VolumeTypeCSI
	case source.Ephemeral != nil:
		return VolumeTypeEphemeral
	default:
		return VolumeTypeUnknown
	}
}

func safeAddQuantity(total *resource.Quantity, add resource.Quantity) error {
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	total.Add(add)
	return nil
}

func (s *podService) evaluateNodeAffinity(pod *v1.Pod, node *v1.Node) (bool, []string) {
	reasons := []string{}

	if len(pod.Spec.NodeSelector) > 0 {
		for key, value := range pod.Spec.NodeSelector {
			if nodeValue, exists := node.Labels[key]; !exists || nodeValue != value {
				reasons = append(reasons, fmt.Sprintf("node selector %s=%s not matched", key, value))
				return false, reasons
			}
		}
		reasons = append(reasons, "all node selectors matched")
	}

	if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil {
		if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			matched := false
			for _, term := range pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
				if s.matchNodeSelectorTerm(node, term) {
					matched = true
					reasons = append(reasons, "required node affinity matched")
					break
				}
			}
			if !matched {
				reasons = append(reasons, "required node affinity not matched")
				return false, reasons
			}
		}

		if len(pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
			reasons = append(reasons, "has preferred node affinity (soft constraint)")
		}
	}

	return true, reasons
}

func (s *podService) matchNodeSelectorTerm(node *v1.Node, term v1.NodeSelectorTerm) bool {
	for _, expr := range term.MatchExpressions {
		if !s.matchNodeSelectorRequirement(node, expr) {
			return false
		}
	}
	for _, field := range term.MatchFields {
		if !s.matchNodeFieldSelector(node, field) {
			return false
		}
	}
	return true
}

func (s *podService) matchNodeSelectorRequirement(node *v1.Node, req v1.NodeSelectorRequirement) bool {
	nodeValue, exists := node.Labels[req.Key]

	switch req.Operator {
	case v1.NodeSelectorOpIn:
		if !exists {
			return false
		}
		for _, value := range req.Values {
			if nodeValue == value {
				return true
			}
		}
		return false
	case v1.NodeSelectorOpNotIn:
		if !exists {
			return true
		}
		for _, value := range req.Values {
			if nodeValue == value {
				return false
			}
		}
		return true
	case v1.NodeSelectorOpExists:
		return exists
	case v1.NodeSelectorOpDoesNotExist:
		return !exists
	case v1.NodeSelectorOpGt, v1.NodeSelectorOpLt:
		return true
	}
	return false
}

func (s *podService) matchNodeFieldSelector(node *v1.Node, field v1.NodeSelectorRequirement) bool {
	var fieldValue string
	switch field.Key {
	case "metadata.name":
		fieldValue = node.Name
	default:
		return false
	}

	switch field.Operator {
	case v1.NodeSelectorOpIn:
		for _, value := range field.Values {
			if fieldValue == value {
				return true
			}
		}
		return false
	case v1.NodeSelectorOpNotIn:
		for _, value := range field.Values {
			if fieldValue == value {
				return false
			}
		}
		return true
	}
	return false
}

func (s *podService) evaluateTaintsAndTolerations(pod *v1.Pod, node *v1.Node) (bool, []models.TaintInfo, []string) {
	untoleratedTaints := []models.TaintInfo{}
	toleratedTaints := []string{}

	for _, taint := range node.Spec.Taints {
		tolerated := false
		for _, toleration := range pod.Spec.Tolerations {
			if s.tolerationMatchesTaint(toleration, taint) {
				tolerated = true
				toleratedTaints = append(toleratedTaints, fmt.Sprintf("%s=%s:%s", taint.Key, taint.Value, taint.Effect))
				break
			}
		}
		if !tolerated && (taint.Effect == v1.TaintEffectNoSchedule || taint.Effect == v1.TaintEffectNoExecute) {
			untoleratedTaints = append(untoleratedTaints, models.TaintInfo{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: string(taint.Effect),
			})
		}
	}

	return len(untoleratedTaints) == 0, untoleratedTaints, toleratedTaints
}

func (s *podService) tolerationMatchesTaint(toleration v1.Toleration, taint v1.Taint) bool {
	if toleration.Key != "" && toleration.Key != taint.Key {
		return false
	}

	if toleration.Effect != "" && toleration.Effect != taint.Effect {
		return false
	}

	switch toleration.Operator {
	case v1.TolerationOpEqual, "":
		return toleration.Value == taint.Value
	case v1.TolerationOpExists:
		return true
	}

	return false
}

func (s *podService) evaluateResourceFit(pod *v1.Pod, node *v1.Node) (models.ResourceFitDetails, []string) {
	insufficientResources := []string{}

	podCPURequest := resource.NewQuantity(0, resource.DecimalSI)
	podMemoryRequest := resource.NewQuantity(0, resource.BinarySI)

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if cpuReq, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
			podCPURequest.Add(cpuReq)
		}
		if memReq, ok := container.Resources.Requests[v1.ResourceMemory]; ok {
			podMemoryRequest.Add(memReq)
		}
	}

	nodeCPUAllocatable := node.Status.Allocatable[v1.ResourceCPU]
	nodeMemoryAllocatable := node.Status.Allocatable[v1.ResourceMemory]

	details := models.ResourceFitDetails{
		NodeCapacity:    node.Status.Capacity,
		NodeAllocatable: node.Status.Allocatable,
		NodeRequested:   v1.ResourceList{},
		PodRequests: v1.ResourceList{
			v1.ResourceCPU:    *podCPURequest,
			v1.ResourceMemory: *podMemoryRequest,
		},
		Fits: true,
	}

	if podCPURequest.Cmp(nodeCPUAllocatable) > 0 {
		details.Fits = false
		insufficientResources = append(insufficientResources, fmt.Sprintf("insufficient CPU (requested: %s, allocatable: %s)",
			podCPURequest.String(), nodeCPUAllocatable.String()))
	}

	if podMemoryRequest.Cmp(nodeMemoryAllocatable) > 0 {
		details.Fits = false
		insufficientResources = append(insufficientResources, fmt.Sprintf("insufficient memory (requested: %s, allocatable: %s)",
			podMemoryRequest.String(), nodeMemoryAllocatable.String()))
	}

	return details, insufficientResources
}

func (s *podService) evaluatePodAntiAffinity(ctx context.Context, pod *v1.Pod, node *v1.Node) (bool, []string) {
	conflicts := []string{}

	if pod.Spec.Affinity == nil || pod.Spec.Affinity.PodAntiAffinity == nil {
		return true, conflicts
	}

	podList, err := s.k8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
	})
	if err != nil {
		s.logger.Warn("failed to list pods on node for anti-affinity check",
			"node", node.Name,
			"error", err.Error())
		return true, conflicts
	}

	for _, term := range pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
		for j := range podList.Items {
			if s.podMatchesAntiAffinityTerm(&podList.Items[j], term) {
				conflicts = append(conflicts, fmt.Sprintf("anti-affinity conflict with pod %s/%s",
					podList.Items[j].Namespace, podList.Items[j].Name))
			}
		}
	}

	return len(conflicts) == 0, conflicts
}

func (s *podService) podMatchesAntiAffinityTerm(pod *v1.Pod, term v1.PodAffinityTerm) bool {
	if term.NamespaceSelector != nil {
	}

	namespaceMatch := false
	if len(term.Namespaces) == 0 {
		namespaceMatch = true
	} else {
		for _, ns := range term.Namespaces {
			if pod.Namespace == ns {
				namespaceMatch = true
				break
			}
		}
	}

	if !namespaceMatch {
		return false
	}

	if term.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(term.LabelSelector)
		if err != nil {
			return false
		}
		return selector.Matches(labels.Set(pod.Labels))
	}

	return false
}

func (s *podService) getSchedulingEvents(ctx context.Context, namespace, podName string) ([]models.SchedulingEvent, error) {
	fieldSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.kind", "Pod"),
		fields.OneTermEqualSelector("involvedObject.name", podName),
		fields.OneTermEqualSelector("involvedObject.namespace", namespace),
	).String()

	eventList, err := s.k8sClient.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
		Limit:         50,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get events for pod %s/%s: %w", namespace, podName, err)
	}

	schedulingEvents := []models.SchedulingEvent{}
	for i := range eventList.Items {
		event := &eventList.Items[i]
		if event.Reason == "FailedScheduling" || event.Reason == "Scheduled" ||
			event.Reason == "Preempted" || event.Reason == "NotTriggerScaleUp" ||
			event.Source.Component == "default-scheduler" {
			schedulingEvents = append(schedulingEvents, models.SchedulingEvent{
				Type:      event.Type,
				Reason:    event.Reason,
				Message:   event.Message,
				Timestamp: event.LastTimestamp,
				Count:     event.Count,
			})
		}
	}

	sort.Slice(schedulingEvents, func(i, j int) bool {
		return schedulingEvents[i].Timestamp.After(schedulingEvents[j].Timestamp.Time)
	})

	return schedulingEvents, nil
}

func (s *podService) analyzeSchedulingDecision(pod *v1.Pod, node *v1.Node) *models.SchedulingDecisions {
	decision := &models.SchedulingDecisions{
		SelectedNode: node.Name,
		Reasons:      []string{},
	}

	affinityMatched, affinityReasons := s.evaluateNodeAffinity(pod, node)
	if affinityMatched {
		decision.Reasons = append(decision.Reasons, affinityReasons...)
		decision.MatchedAffinity = affinityReasons
	}

	taintsOk, _, toleratedTaints := s.evaluateTaintsAndTolerations(pod, node)
	if taintsOk {
		if len(toleratedTaints) > 0 {
			decision.Reasons = append(decision.Reasons, fmt.Sprintf("tolerates node taints: %v", toleratedTaints))
			decision.ToleratedTaints = toleratedTaints
		} else {
			decision.Reasons = append(decision.Reasons, "node has no taints")
		}
	}

	if len(pod.Spec.NodeSelector) > 0 {
		decision.MatchedNodeSelector = make(map[string]string)
		for key, value := range pod.Spec.NodeSelector {
			if nodeValue, exists := node.Labels[key]; exists && nodeValue == value {
				decision.MatchedNodeSelector[key] = value
			}
		}
		if len(decision.MatchedNodeSelector) > 0 {
			decision.Reasons = append(decision.Reasons, "node selector labels matched")
		}
	}

	resourcesFit, _ := s.evaluateResourceFit(pod, node)
	decision.ResourcesFit = resourcesFit
	if resourcesFit.Fits {
		decision.Reasons = append(decision.Reasons, "node has sufficient resources")
	}

	if len(decision.Reasons) == 0 {
		decision.Reasons = append(decision.Reasons, "no specific scheduling constraints, selected by scheduler algorithm")
	}

	return decision
}

func (s *podService) analyzeUnschedulableNodes(ctx context.Context, pod *v1.Pod) ([]models.UnschedulableNode, error) {
	nodeList, err := s.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	hasVolumes := s.checkPodVolumes(pod)

	unschedulableNodes := make([]models.UnschedulableNode, 0, len(nodeList.Items))

	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		unschedulable := models.UnschedulableNode{
			NodeName: node.Name,
			Reasons:  []string{},
		}

		nodeReady := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
				nodeReady = true
				break
			}
		}
		if !nodeReady {
			unschedulable.Reasons = append(unschedulable.Reasons, "node is not ready")
		}

		if node.Spec.Unschedulable {
			unschedulable.Reasons = append(unschedulable.Reasons, "node is marked as unschedulable")
		}

		affinityMatched, affinityReasons := s.evaluateNodeAffinity(pod, node)
		if !affinityMatched {
			unschedulable.Reasons = append(unschedulable.Reasons, affinityReasons...)
			unschedulable.UnmatchedAffinity = affinityReasons
		}

		taintsOk, untoleratedTaints, _ := s.evaluateTaintsAndTolerations(pod, node)
		if !taintsOk {
			unschedulable.Reasons = append(unschedulable.Reasons,
				fmt.Sprintf("node has untolerated taints: %d", len(untoleratedTaints)))
			unschedulable.UntoleratedTaints = untoleratedTaints
		}

		if len(pod.Spec.NodeSelector) > 0 {
			unschedulable.UnmatchedSelectors = make(map[string]string)
			for key, value := range pod.Spec.NodeSelector {
				if nodeValue, exists := node.Labels[key]; !exists || nodeValue != value {
					unschedulable.UnmatchedSelectors[key] = value
				}
			}
			if len(unschedulable.UnmatchedSelectors) > 0 {
				unschedulable.Reasons = append(unschedulable.Reasons, "node selector not matched")
			}
		}

		resourcesFit, insufficientResources := s.evaluateResourceFit(pod, node)
		if !resourcesFit.Fits {
			unschedulable.Reasons = append(unschedulable.Reasons, "insufficient resources")
			unschedulable.InsufficientResources = insufficientResources
		}

		antiAffinityOk, conflicts := s.evaluatePodAntiAffinity(ctx, pod, node)
		if !antiAffinityOk {
			unschedulable.Reasons = append(unschedulable.Reasons, "pod anti-affinity conflict")
			unschedulable.PodAffinityConflicts = conflicts
		}

		if hasVolumes {
			volumeOk, volumeIssues := s.analyzeVolumeConstraints(ctx, pod, node)
			if !volumeOk {
				unschedulable.Reasons = append(unschedulable.Reasons, volumeIssues...)
			}
		}

		if len(unschedulable.Reasons) > 0 {
			unschedulableNodes = append(unschedulableNodes, unschedulable)
		}
	}

	return unschedulableNodes, nil
}

func (s *podService) categorizeSchedulingFailure(reasons []string, events []models.SchedulingEvent) []models.SchedulingFailureCategory {
	categories := make(map[models.SchedulingFailureCategory]bool)

	for _, reason := range reasons {
		reasonLower := strings.ToLower(reason)

		if strings.Contains(reasonLower, "insufficient cpu") {
			categories[models.FailureCategoryResourceCPU] = true
		}
		if strings.Contains(reasonLower, "insufficient memory") {
			categories[models.FailureCategoryResourceMemory] = true
		}
		if strings.Contains(reasonLower, "insufficient storage") || strings.Contains(reasonLower, "insufficient ephemeral-storage") {
			categories[models.FailureCategoryResourceStorage] = true
		}

		if strings.Contains(reasonLower, "node is not ready") || strings.Contains(reasonLower, "node not ready") {
			categories[models.FailureCategoryNodeNotReady] = true
		}

		if strings.Contains(reasonLower, "node affinity") || strings.Contains(reasonLower, "node selector") {
			categories[models.FailureCategoryNodeAffinity] = true
		}

		if strings.Contains(reasonLower, "taint") || strings.Contains(reasonLower, "toleration") {
			categories[models.FailureCategoryTaints] = true
		}

		if strings.Contains(reasonLower, "pod affinity") || strings.Contains(reasonLower, "anti-affinity") {
			categories[models.FailureCategoryPodAffinity] = true
		}
	}

	volumeCategories := s.parseEventsForVolumeIssues(events)
	for cat := range volumeCategories {
		categories[cat] = true
	}

	result := make([]models.SchedulingFailureCategory, 0, len(categories))
	for cat := range categories {
		result = append(result, cat)
	}

	if len(result) == 0 && len(reasons) > 0 {
		result = append(result, models.FailureCategoryMiscellaneous)
	}

	return result
}

func (s *podService) parseEventsForVolumeIssues(events []models.SchedulingEvent) map[models.SchedulingFailureCategory]bool {
	categories := make(map[models.SchedulingFailureCategory]bool)

	for _, event := range events {
		msgLower := strings.ToLower(event.Message)

		if strings.Contains(msgLower, "multi-attach error") ||
			strings.Contains(msgLower, "volume is already exclusively attached") ||
			strings.Contains(msgLower, "volume is already used by") {
			categories[models.FailureCategoryVolumeMultiAttach] = true
		}

		if strings.Contains(msgLower, "volume node affinity conflict") ||
			strings.Contains(msgLower, "nodeaffinity") ||
			(strings.Contains(msgLower, "volume") && strings.Contains(msgLower, "node affinity")) {
			categories[models.FailureCategoryVolumeNodeAffinity] = true
		}

		if strings.Contains(msgLower, "failedattachvolume") ||
			strings.Contains(msgLower, "failed to attach volume") ||
			strings.Contains(msgLower, "unable to attach") ||
			strings.Contains(msgLower, "attachvolume.attach failed") {
			categories[models.FailureCategoryVolumeAttachment] = true
		}

		if event.Reason == "FailedScheduling" && strings.Contains(msgLower, "volume") {
			if len(categories) == 0 {
				categories[models.FailureCategoryVolumeAttachment] = true
			}
		}
	}

	return categories
}

func (s *podService) aggregateFailureCategories(unschedulableNodes []models.UnschedulableNode, events []models.SchedulingEvent) []models.FailureCategorySummary {
	categoryMap := make(map[models.SchedulingFailureCategory]*models.FailureCategorySummary)

	for _, node := range unschedulableNodes {
		nodeCategories := s.categorizeSchedulingFailure(node.Reasons, events)

		for _, cat := range nodeCategories {
			if summary, exists := categoryMap[cat]; exists {
				summary.Count++
				summary.Nodes = append(summary.Nodes, node.NodeName)
			} else {
				categoryMap[cat] = &models.FailureCategorySummary{
					Category:    cat,
					Count:       1,
					Description: getCategoryDescription(cat),
					Nodes:       []string{node.NodeName},
				}
			}
		}
	}

	summaries := make([]models.FailureCategorySummary, 0, len(categoryMap))
	for _, summary := range categoryMap {
		summaries = append(summaries, *summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Count > summaries[j].Count
	})

	return summaries
}

func getCategoryDescription(category models.SchedulingFailureCategory) string {
	descriptions := map[models.SchedulingFailureCategory]string{
		models.FailureCategoryResourceCPU:        "Insufficient CPU resources available on nodes",
		models.FailureCategoryResourceMemory:     "Insufficient memory resources available on nodes",
		models.FailureCategoryResourceStorage:    "Insufficient storage resources available on nodes",
		models.FailureCategoryVolumeAttachment:   "Failed to attach persistent volume to node",
		models.FailureCategoryVolumeMultiAttach:  "Volume already attached to another node (ReadWriteOnce)",
		models.FailureCategoryVolumeNodeAffinity: "Volume zone/region doesn't match node placement",
		models.FailureCategoryNodeAffinity:       "Node selector or affinity requirements not satisfied",
		models.FailureCategoryTaints:             "Node taints not tolerated by pod",
		models.FailureCategoryPodAffinity:        "Pod affinity or anti-affinity constraints not satisfied",
		models.FailureCategoryNodeNotReady:       "Node is not in ready state",
		models.FailureCategoryMiscellaneous:      "Other scheduling constraints not satisfied",
	}

	if desc, ok := descriptions[category]; ok {
		return desc
	}
	return "Unknown scheduling failure"
}

func (s *podService) checkPodVolumes(pod *v1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			return true
		}
	}
	return false
}

func (s *podService) analyzeVolumeConstraints(ctx context.Context, pod *v1.Pod, node *v1.Node) (bool, []string) {
	volumeIssues := []string{}

	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim == nil {
			continue
		}

		pvc, err := s.k8sClient.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(
			ctx, volume.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
		if err != nil {
			s.logger.Warn("failed to get PVC for volume analysis",
				"pvc", volume.PersistentVolumeClaim.ClaimName,
				"namespace", pod.Namespace,
				"error", err.Error())
			continue
		}

		if pvc.Status.Phase != v1.ClaimBound {
			volumeIssues = append(volumeIssues, fmt.Sprintf("PVC %s is not bound (status: %s)",
				pvc.Name, pvc.Status.Phase))
			continue
		}

		if pvc.Spec.VolumeName != "" {
			pv, err := s.k8sClient.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{})
			if err != nil {
				s.logger.Warn("failed to get PV for volume analysis",
					"pv", pvc.Spec.VolumeName,
					"error", err.Error())
				continue
			}

			if pv.Spec.NodeAffinity != nil && pv.Spec.NodeAffinity.Required != nil {
				matches := false
				for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
					if s.matchNodeSelectorTerm(node, term) {
						matches = true
						break
					}
				}
				if !matches {
					volumeIssues = append(volumeIssues,
						fmt.Sprintf("PV %s has node affinity that doesn't match this node", pv.Name))
				}
			}

			if len(pvc.Status.AccessModes) > 0 {
				for _, mode := range pvc.Status.AccessModes {
					if mode == v1.ReadWriteOnce {
						volumeIssues = append(volumeIssues,
							fmt.Sprintf("PVC %s has ReadWriteOnce access mode (potential multi-attach issue)", pvc.Name))
					}
				}
			}
		}
	}

	return len(volumeIssues) == 0, volumeIssues
}

func (s *podService) GetPodFailureEvents(ctx context.Context, namespace, name string) (*models.PodFailureEvents, error) {
	s.logger.Debug("getting pod failure events", "namespace", namespace, "pod", name)

	pod, err := s.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	events, err := s.getPodEvents(ctx, namespace, name)
	if err != nil {
		s.logger.Warn("failed to get pod events for failure analysis",
			"namespace", namespace,
			"pod", name,
			"error", err.Error())
		events = []models.EventInfo{}
	}

	failureEvents := s.analyzeFailureEvents(events, pod)

	result := &models.PodFailureEvents{
		PodName:         name,
		Namespace:       namespace,
		TotalEvents:     len(events),
		FailureEvents:   failureEvents,
		EventCategories: make(map[models.FailureEventCategory]int),
		PodPhase:        string(pod.Status.Phase),
		PodStatus:       pod.Status.Reason,
	}

	for i := range failureEvents {
		event := &failureEvents[i]
		result.EventCategories[event.Category]++

		switch event.Severity {
		case "critical":
			result.CriticalEvents++
		case "warning":
			result.WarningEvents++
		}

		if result.MostRecentIssue == nil || event.LastTimestamp.After(result.MostRecentIssue.LastTimestamp.Time) {
			result.MostRecentIssue = event
		}
	}

	result.OngoingIssues = s.identifyOngoingIssues(failureEvents)

	s.logger.Debug("successfully analyzed pod failure events",
		"namespace", namespace,
		"pod", name,
		"total_events", result.TotalEvents,
		"failure_events", len(result.FailureEvents),
		"critical_events", result.CriticalEvents,
		"warning_events", result.WarningEvents)

	return result, nil
}

func (s *podService) analyzeFailureEvents(events []models.EventInfo, pod *v1.Pod) []models.FailureEvent {
	failureEvents := []models.FailureEvent{}
	now := time.Now()

	failurePatterns := map[string]struct {
		category        models.FailureEventCategory
		severity        string
		possibleCauses  []string
		suggestedAction string
	}{
		"FailedScheduling": {
			category:        models.FailureEventCategoryScheduling,
			severity:        "critical",
			possibleCauses:  []string{"Insufficient resources", "Node selector mismatch", "Affinity rules", "Taints not tolerated"},
			suggestedAction: "Check node resources and scheduling constraints",
		},
		"BackOff": {
			category:        models.FailureEventCategoryCrash,
			severity:        "critical",
			possibleCauses:  []string{"Application crash", "Missing dependencies", "Configuration error"},
			suggestedAction: "Check container logs for crash details",
		},
		"CrashLoopBackOff": {
			category:        models.FailureEventCategoryCrash,
			severity:        "critical",
			possibleCauses:  []string{"Repeated application crashes", "Startup failure", "Missing configuration"},
			suggestedAction: "Examine container logs and fix application startup issues",
		},
		"ImagePullBackOff": {
			category:        models.FailureEventCategoryImagePull,
			severity:        "critical",
			possibleCauses:  []string{"Image not found", "Registry authentication failure", "Network issues"},
			suggestedAction: "Verify image name and registry credentials",
		},
		"ErrImagePull": {
			category:        models.FailureEventCategoryImagePull,
			severity:        "critical",
			possibleCauses:  []string{"Invalid image name", "Registry unreachable", "No pull secrets"},
			suggestedAction: "Check image availability and pull secrets",
		},
		"FailedAttachVolume": {
			category:        models.FailureEventCategoryVolume,
			severity:        "critical",
			possibleCauses:  []string{"Volume already attached", "Volume not found", "Zone mismatch"},
			suggestedAction: "Check volume status and node availability zones",
		},
		"FailedMount": {
			category:        models.FailureEventCategoryVolume,
			severity:        "critical",
			possibleCauses:  []string{"Volume not ready", "Mount permissions", "Filesystem issues"},
			suggestedAction: "Verify volume is properly provisioned and accessible",
		},
		"Unhealthy": {
			category:        models.FailureEventCategoryProbe,
			severity:        "warning",
			possibleCauses:  []string{"Liveness probe failure", "Readiness probe failure", "Application not responding"},
			suggestedAction: "Review probe configuration and application health endpoints",
		},
		"OOMKilled": {
			category:        models.FailureEventCategoryResource,
			severity:        "critical",
			possibleCauses:  []string{"Memory limit exceeded", "Memory leak", "Insufficient memory allocation"},
			suggestedAction: "Increase memory limits or optimize application memory usage",
		},
		"Evicted": {
			category:        models.FailureEventCategoryResource,
			severity:        "warning",
			possibleCauses:  []string{"Node pressure", "Resource limits", "Priority preemption"},
			suggestedAction: "Check node resources and pod priority settings",
		},
		"NetworkNotReady": {
			category:        models.FailureEventCategoryNetwork,
			severity:        "warning",
			possibleCauses:  []string{"CNI plugin issues", "Network policy blocking", "Service mesh problems"},
			suggestedAction: "Check network plugin status and network policies",
		},
	}

	for _, event := range events {
		if event.Type == "Normal" && event.Count < 5 {
			continue
		}

		var failureEvent *models.FailureEvent
		for pattern, config := range failurePatterns {
			if strings.Contains(event.Reason, pattern) {
				failureEvent = &models.FailureEvent{
					EventInfo:       event,
					Category:        config.category,
					Severity:        config.severity,
					PossibleCauses:  config.possibleCauses,
					SuggestedAction: config.suggestedAction,
				}
				break
			}
		}

		if failureEvent == nil && event.Type == "Warning" {
			failureEvent = &models.FailureEvent{
				EventInfo:       event,
				Category:        models.FailureEventCategoryOther,
				Severity:        "warning",
				PossibleCauses:  []string{"Check event message for details"},
				SuggestedAction: "Investigate based on event message",
			}
		}

		if failureEvent == nil {
			continue
		}

		if event.Count > 3 {
			failureEvent.IsRecurring = true
			duration := event.LastTimestamp.Sub(event.FirstTimestamp.Time)
			if duration > 0 {
				rate := float64(event.Count) / duration.Hours()
				if rate > 1 {
					failureEvent.RecurrenceRate = fmt.Sprintf("%.1f times per hour", rate)
				} else {
					failureEvent.RecurrenceRate = fmt.Sprintf("%d times in %.1f hours", event.Count, duration.Hours())
				}
			}
		}

		timeSinceFirst := now.Sub(event.FirstTimestamp.Time)
		if timeSinceFirst > 0 {
			failureEvent.TimeSinceFirst = s.formatDuration(timeSinceFirst)
		}

		s.enhanceFailureEventContext(failureEvent, pod)

		failureEvents = append(failureEvents, *failureEvent)
	}

	sort.Slice(failureEvents, func(i, j int) bool {
		if failureEvents[i].Severity != failureEvents[j].Severity {
			return s.severityWeight(failureEvents[i].Severity) > s.severityWeight(failureEvents[j].Severity)
		}
		return failureEvents[i].LastTimestamp.After(failureEvents[j].LastTimestamp.Time)
	})

	return failureEvents
}

func (s *podService) enhanceFailureEventContext(event *models.FailureEvent, pod *v1.Pod) {
	switch event.Category {
	case models.FailureEventCategoryCrash:
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
				event.PossibleCauses = append(event.PossibleCauses,
					fmt.Sprintf("Container %s exited with code %d", status.Name, status.State.Terminated.ExitCode))
			}
			if status.RestartCount > 0 {
				event.PossibleCauses = append(event.PossibleCauses,
					fmt.Sprintf("Container %s has restarted %d times", status.Name, status.RestartCount))
			}
		}
	case models.FailureEventCategoryResource:
		if pod.Status.QOSClass == v1.PodQOSBurstable || pod.Status.QOSClass == v1.PodQOSBestEffort {
			event.PossibleCauses = append(event.PossibleCauses,
				fmt.Sprintf("Pod QoS class is %s - consider setting guaranteed QoS", pod.Status.QOSClass))
		}
	}
}

func (s *podService) identifyOngoingIssues(events []models.FailureEvent) []string {
	ongoing := []string{}
	threshold := time.Now().Add(-5 * time.Minute)

	for _, event := range events {
		if event.LastTimestamp.After(threshold) && event.Severity == "critical" {
			issue := fmt.Sprintf("%s: %s", event.Reason, event.Message)
			if len(issue) > 100 {
				issue = issue[:97] + "..."
			}
			ongoing = append(ongoing, issue)
		}
	}

	return ongoing
}

func (s *podService) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}

func (s *podService) severityWeight(severity string) int {
	switch severity {
	case "critical":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}
