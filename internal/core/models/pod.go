package models

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SchedulingFailureCategory represents the category of scheduling failure
type SchedulingFailureCategory string

const (
	// Resource-related failures
	FailureCategoryResourceCPU     SchedulingFailureCategory = "InsufficientCPU"
	FailureCategoryResourceMemory  SchedulingFailureCategory = "InsufficientMemory"
	FailureCategoryResourceStorage SchedulingFailureCategory = "InsufficientStorage"

	// Volume-related failures
	FailureCategoryVolumeAttachment   SchedulingFailureCategory = "VolumeAttachmentError"
	FailureCategoryVolumeMultiAttach  SchedulingFailureCategory = "VolumeMultiAttachError"
	FailureCategoryVolumeNodeAffinity SchedulingFailureCategory = "VolumeNodeAffinityConflict"

	// Scheduling constraint failures
	FailureCategoryNodeAffinity SchedulingFailureCategory = "NodeAffinityNotMatch"
	FailureCategoryTaints       SchedulingFailureCategory = "TaintTolerationMismatch"
	FailureCategoryPodAffinity  SchedulingFailureCategory = "PodAffinityConflict"

	// Node status failures
	FailureCategoryNodeNotReady SchedulingFailureCategory = "NodeNotReady"

	// Other failures
	FailureCategoryMiscellaneous SchedulingFailureCategory = "Miscellaneous"
)

// FailureCategorySummary provides a summary of failure categories
type FailureCategorySummary struct {
	Category    SchedulingFailureCategory `json:"category"`
	Count       int                       `json:"count"`
	Description string                    `json:"description"`
	Nodes       []string                  `json:"nodes,omitempty"`
}

// PodScheduling contains scheduling-specific information for a pod
type PodScheduling struct {
	NodeName          string            `json:"nodeName,omitempty"`
	SchedulerName     string            `json:"schedulerName"`
	Affinity          *v1.Affinity      `json:"affinity,omitempty"`
	Tolerations       []v1.Toleration   `json:"tolerations,omitempty"`
	NodeSelector      map[string]string `json:"nodeSelector,omitempty"`
	Priority          *int32            `json:"priority,omitempty"`
	PriorityClassName string            `json:"priorityClassName,omitempty"`

	// Enhanced scheduling information
	Status              string                      `json:"status"` // "Scheduled", "Pending", "Failed"
	SchedulingDecisions *SchedulingDecisions        `json:"schedulingDecisions,omitempty"`
	UnschedulableNodes  []UnschedulableNode         `json:"unschedulableNodes,omitempty"`
	Events              []SchedulingEvent           `json:"events,omitempty"`
	Conditions          []v1.PodCondition           `json:"conditions,omitempty"`
	FailureCategories   []SchedulingFailureCategory `json:"failureCategories,omitempty"`
	FailureSummary      []FailureCategorySummary    `json:"failureSummary,omitempty"`
}

// SchedulingDecisions explains why a pod was scheduled on a specific node
type SchedulingDecisions struct {
	SelectedNode        string             `json:"selectedNode"`
	Reasons             []string           `json:"reasons"`
	NodeScore           int32              `json:"nodeScore,omitempty"`
	MatchedAffinity     []string           `json:"matchedAffinity,omitempty"`
	ToleratedTaints     []string           `json:"toleratedTaints,omitempty"`
	MatchedNodeSelector map[string]string  `json:"matchedNodeSelector,omitempty"`
	ResourcesFit        ResourceFitDetails `json:"resourcesFit"`
}

// UnschedulableNode contains information about why a pod cannot be scheduled on a specific node
type UnschedulableNode struct {
	NodeName              string            `json:"nodeName"`
	Reasons               []string          `json:"reasons"`
	UntoleratedTaints     []TaintInfo       `json:"untoleratedTaints,omitempty"`
	UnmatchedAffinity     []string          `json:"unmatchedAffinity,omitempty"`
	UnmatchedSelectors    map[string]string `json:"unmatchedSelectors,omitempty"`
	InsufficientResources []string          `json:"insufficientResources,omitempty"`
	PodAffinityConflicts  []string          `json:"podAffinityConflicts,omitempty"`
}

// TaintInfo contains simplified taint information
type TaintInfo struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

// ResourceFitDetails contains information about resource availability
type ResourceFitDetails struct {
	NodeCapacity    v1.ResourceList `json:"nodeCapacity"`
	NodeAllocatable v1.ResourceList `json:"nodeAllocatable"`
	NodeRequested   v1.ResourceList `json:"nodeRequested"`
	PodRequests     v1.ResourceList `json:"podRequests"`
	Fits            bool            `json:"fits"`
}

// SchedulingEvent contains scheduling-related event information
type SchedulingEvent struct {
	Type      string      `json:"type"`
	Reason    string      `json:"reason"`
	Message   string      `json:"message"`
	Timestamp metav1.Time `json:"timestamp"`
	Count     int32       `json:"count"`
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

// FailureEventCategory represents the category of failure event
type FailureEventCategory string

const (
	// Scheduling failures
	FailureEventCategoryScheduling FailureEventCategory = "Scheduling"
	// Image pull failures
	FailureEventCategoryImagePull FailureEventCategory = "ImagePull"
	// Container crash failures
	FailureEventCategoryCrash FailureEventCategory = "ContainerCrash"
	// Volume attachment failures
	FailureEventCategoryVolume FailureEventCategory = "Volume"
	// Resource limit failures
	FailureEventCategoryResource FailureEventCategory = "Resource"
	// Liveness/Readiness probe failures
	FailureEventCategoryProbe FailureEventCategory = "Probe"
	// Network/Service failures
	FailureEventCategoryNetwork FailureEventCategory = "Network"
	// Other uncategorized failures
	FailureEventCategoryOther FailureEventCategory = "Other"
)

// FailureEvent represents a failure event with analysis
type FailureEvent struct {
	EventInfo
	Category        FailureEventCategory `json:"category"`
	Severity        string               `json:"severity"` // "critical", "warning", "info"
	IsRecurring     bool                 `json:"isRecurring"`
	RecurrenceRate  string               `json:"recurrenceRate,omitempty"` // e.g., "5 times in last hour"
	TimeSinceFirst  string               `json:"timeSinceFirst,omitempty"` // e.g., "2h30m"
	PossibleCauses  []string             `json:"possibleCauses,omitempty"`
	SuggestedAction string               `json:"suggestedAction,omitempty"`
}

// PodFailureEvents contains failure events analysis for a pod
type PodFailureEvents struct {
	PodName         string                       `json:"podName"`
	Namespace       string                       `json:"namespace"`
	TotalEvents     int                          `json:"totalEvents"`
	FailureEvents   []FailureEvent               `json:"failureEvents"`
	EventCategories map[FailureEventCategory]int `json:"eventCategories"`
	CriticalEvents  int                          `json:"criticalEvents"`
	WarningEvents   int                          `json:"warningEvents"`
	MostRecentIssue *FailureEvent                `json:"mostRecentIssue,omitempty"`
	OngoingIssues   []string                     `json:"ongoingIssues,omitempty"`
	PodPhase        string                       `json:"podPhase"`
	PodStatus       string                       `json:"podStatus"`
}
