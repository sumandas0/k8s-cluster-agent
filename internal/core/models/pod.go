package models

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodScheduling contains scheduling-specific information for a pod
type PodScheduling struct {
	NodeName          string            `json:"nodeName,omitempty"`
	SchedulerName     string            `json:"schedulerName"`
	Affinity          *v1.Affinity      `json:"affinity,omitempty"`
	Tolerations       []v1.Toleration   `json:"tolerations,omitempty"`
	NodeSelector      map[string]string `json:"nodeSelector,omitempty"`
	Priority          *int32            `json:"priority,omitempty"`
	PriorityClassName string            `json:"priorityClassName,omitempty"`
}

// PodResources contains aggregated resource information for a pod
type PodResources struct {
	Containers []ContainerResources `json:"containers"`
	Total      ResourceSummary      `json:"total"`
}

// ContainerResources contains resource requirements for a single container
type ContainerResources struct {
	Name     string          `json:"name"`
	Requests v1.ResourceList `json:"requests"`
	Limits   v1.ResourceList `json:"limits"`
}

// ResourceSummary contains human-readable resource summary
type ResourceSummary struct {
	CPURequest    string `json:"cpuRequest"`
	CPULimit      string `json:"cpuLimit"`
	MemoryRequest string `json:"memoryRequest"`
	MemoryLimit   string `json:"memoryLimit"`
}

// PodDescription contains comprehensive pod information similar to kubectl describe pod
type PodDescription struct {
	// Basic Information
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	// Status Information
	Status PodStatusInfo `json:"status"`

	// Scheduling Information
	Node      string       `json:"node,omitempty"`
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Container Information
	Containers     []ContainerInfo `json:"containers"`
	InitContainers []ContainerInfo `json:"initContainers,omitempty"`

	// Volumes
	Volumes []VolumeInfo `json:"volumes,omitempty"`

	// Network
	PodIP  string   `json:"podIP,omitempty"`
	PodIPs []string `json:"podIPs,omitempty"`

	// QoS and Priority
	QOSClass          string `json:"qosClass,omitempty"`
	Priority          *int32 `json:"priority,omitempty"`
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Tolerations and Node Selection
	Tolerations  []v1.Toleration   `json:"tolerations,omitempty"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Events (most recent)
	Events []EventInfo `json:"events,omitempty"`

	// Conditions
	Conditions []v1.PodCondition `json:"conditions,omitempty"`
}

// PodStatusInfo contains detailed status information
type PodStatusInfo struct {
	Phase             string `json:"phase"`
	Reason            string `json:"reason,omitempty"`
	Message           string `json:"message,omitempty"`
	HostIP            string `json:"hostIP,omitempty"`
	PodIP             string `json:"podIP,omitempty"`
	NominatedNodeName string `json:"nominatedNodeName,omitempty"`
}

// ContainerInfo contains detailed container information
type ContainerInfo struct {
	Name         string                  `json:"name"`
	Image        string                  `json:"image"`
	ImageID      string                  `json:"imageID,omitempty"`
	State        v1.ContainerState       `json:"state"`
	Ready        bool                    `json:"ready"`
	RestartCount int32                   `json:"restartCount"`
	Resources    v1.ResourceRequirements `json:"resources,omitempty"`
	Environment  []v1.EnvVar             `json:"environment,omitempty"`
	Mounts       []VolumeMountInfo       `json:"mounts,omitempty"`
}

// VolumeInfo contains volume information
type VolumeInfo struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Source v1.VolumeSource `json:"source"`
}

// VolumeMountInfo contains volume mount information
type VolumeMountInfo struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
	SubPath   string `json:"subPath,omitempty"`
}

// EventInfo contains simplified event information
type EventInfo struct {
	Type           string      `json:"type"`
	Reason         string      `json:"reason"`
	Message        string      `json:"message"`
	FirstTimestamp metav1.Time `json:"firstTimestamp"`
	LastTimestamp  metav1.Time `json:"lastTimestamp"`
	Count          int32       `json:"count"`
	Source         string      `json:"source"`
}
