package models

import (
	v1 "k8s.io/api/core/v1"
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
