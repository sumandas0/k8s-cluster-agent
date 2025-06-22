package models

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SchedulingFailureCategory string

const (
	FailureCategoryResourceCPU     SchedulingFailureCategory = "InsufficientCPU"
	FailureCategoryResourceMemory  SchedulingFailureCategory = "InsufficientMemory"
	FailureCategoryResourceStorage SchedulingFailureCategory = "InsufficientStorage"

	FailureCategoryVolumeAttachment   SchedulingFailureCategory = "VolumeAttachmentError"
	FailureCategoryVolumeMultiAttach  SchedulingFailureCategory = "VolumeMultiAttachError"
	FailureCategoryVolumeNodeAffinity SchedulingFailureCategory = "VolumeNodeAffinityConflict"

	FailureCategoryNodeAffinity SchedulingFailureCategory = "NodeAffinityNotMatch"
	FailureCategoryTaints       SchedulingFailureCategory = "TaintTolerationMismatch"
	FailureCategoryPodAffinity  SchedulingFailureCategory = "PodAffinityConflict"

	FailureCategoryNodeNotReady SchedulingFailureCategory = "NodeNotReady"

	FailureCategoryMiscellaneous SchedulingFailureCategory = "Miscellaneous"
)

type FailureCategorySummary struct {
	Category    SchedulingFailureCategory `json:"category"`
	Count       int                       `json:"count"`
	Description string                    `json:"description"`
	Nodes       []string                  `json:"nodes,omitempty"`
}

type PodScheduling struct {
	NodeName          string            `json:"nodeName,omitempty"`
	SchedulerName     string            `json:"schedulerName"`
	Affinity          *v1.Affinity      `json:"affinity,omitempty"`
	Tolerations       []v1.Toleration   `json:"tolerations,omitempty"`
	NodeSelector      map[string]string `json:"nodeSelector,omitempty"`
	Priority          *int32            `json:"priority,omitempty"`
	PriorityClassName string            `json:"priorityClassName,omitempty"`

	Status              string                      `json:"status"`
	SchedulingDecisions *SchedulingDecisions        `json:"schedulingDecisions,omitempty"`
	UnschedulableNodes  []UnschedulableNode         `json:"unschedulableNodes,omitempty"`
	Events              []SchedulingEvent           `json:"events,omitempty"`
	Conditions          []v1.PodCondition           `json:"conditions,omitempty"`
	FailureCategories   []SchedulingFailureCategory `json:"failureCategories,omitempty"`
	FailureSummary      []FailureCategorySummary    `json:"failureSummary,omitempty"`
}

type SchedulingDecisions struct {
	SelectedNode        string             `json:"selectedNode"`
	Reasons             []string           `json:"reasons"`
	NodeScore           int32              `json:"nodeScore,omitempty"`
	MatchedAffinity     []string           `json:"matchedAffinity,omitempty"`
	ToleratedTaints     []string           `json:"toleratedTaints,omitempty"`
	MatchedNodeSelector map[string]string  `json:"matchedNodeSelector,omitempty"`
	ResourcesFit        ResourceFitDetails `json:"resourcesFit"`
}

type UnschedulableNode struct {
	NodeName              string            `json:"nodeName"`
	Reasons               []string          `json:"reasons"`
	UntoleratedTaints     []TaintInfo       `json:"untoleratedTaints,omitempty"`
	UnmatchedAffinity     []string          `json:"unmatchedAffinity,omitempty"`
	UnmatchedSelectors    map[string]string `json:"unmatchedSelectors,omitempty"`
	InsufficientResources []string          `json:"insufficientResources,omitempty"`
	PodAffinityConflicts  []string          `json:"podAffinityConflicts,omitempty"`
}

type TaintInfo struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

type ResourceFitDetails struct {
	NodeCapacity    v1.ResourceList `json:"nodeCapacity"`
	NodeAllocatable v1.ResourceList `json:"nodeAllocatable"`
	NodeRequested   v1.ResourceList `json:"nodeRequested"`
	PodRequests     v1.ResourceList `json:"podRequests"`
	Fits            bool            `json:"fits"`
}

type SchedulingEvent struct {
	Type      string      `json:"type"`
	Reason    string      `json:"reason"`
	Message   string      `json:"message"`
	Timestamp metav1.Time `json:"timestamp"`
	Count     int32       `json:"count"`
}

type PodResources struct {
	Containers []ContainerResources `json:"containers"`
	Total      ResourceSummary      `json:"total"`
}

type ContainerResources struct {
	Name     string          `json:"name"`
	Requests v1.ResourceList `json:"requests"`
	Limits   v1.ResourceList `json:"limits"`
}

type ResourceSummary struct {
	CPURequest    string `json:"cpuRequest"`
	CPULimit      string `json:"cpuLimit"`
	MemoryRequest string `json:"memoryRequest"`
	MemoryLimit   string `json:"memoryLimit"`
}

type PodDescription struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Status PodStatusInfo `json:"status"`

	Node      string       `json:"node,omitempty"`
	StartTime *metav1.Time `json:"startTime,omitempty"`

	Containers     []ContainerInfo `json:"containers"`
	InitContainers []ContainerInfo `json:"initContainers,omitempty"`

	Volumes []VolumeInfo `json:"volumes,omitempty"`

	PodIP  string   `json:"podIP,omitempty"`
	PodIPs []string `json:"podIPs,omitempty"`

	QOSClass          string `json:"qosClass,omitempty"`
	Priority          *int32 `json:"priority,omitempty"`
	PriorityClassName string `json:"priorityClassName,omitempty"`

	Tolerations  []v1.Toleration   `json:"tolerations,omitempty"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	Events []EventInfo `json:"events,omitempty"`

	Conditions []v1.PodCondition `json:"conditions,omitempty"`
}

type PodStatusInfo struct {
	Phase             string `json:"phase"`
	Reason            string `json:"reason,omitempty"`
	Message           string `json:"message,omitempty"`
	HostIP            string `json:"hostIP,omitempty"`
	PodIP             string `json:"podIP,omitempty"`
	NominatedNodeName string `json:"nominatedNodeName,omitempty"`
}

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

type VolumeInfo struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Source v1.VolumeSource `json:"source"`
}

type VolumeMountInfo struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
	SubPath   string `json:"subPath,omitempty"`
}

type EventInfo struct {
	Type           string      `json:"type"`
	Reason         string      `json:"reason"`
	Message        string      `json:"message"`
	FirstTimestamp metav1.Time `json:"firstTimestamp"`
	LastTimestamp  metav1.Time `json:"lastTimestamp"`
	Count          int32       `json:"count"`
	Source         string      `json:"source"`
}

type FailureEventCategory string

const (
	FailureEventCategoryScheduling FailureEventCategory = "Scheduling"
	FailureEventCategoryImagePull FailureEventCategory = "ImagePull"
	FailureEventCategoryCrash FailureEventCategory = "ContainerCrash"
	FailureEventCategoryVolume FailureEventCategory = "Volume"
	FailureEventCategoryResource FailureEventCategory = "Resource"
	FailureEventCategoryProbe FailureEventCategory = "Probe"
	FailureEventCategoryNetwork FailureEventCategory = "Network"
	FailureEventCategoryOther FailureEventCategory = "Other"
)

type FailureEvent struct {
	EventInfo
	Category        FailureEventCategory `json:"category"`
	Severity        string               `json:"severity"`
	IsRecurring     bool                 `json:"isRecurring"`
	RecurrenceRate  string               `json:"recurrenceRate,omitempty"`
	TimeSinceFirst  string               `json:"timeSinceFirst,omitempty"`
	PossibleCauses  []string             `json:"possibleCauses,omitempty"`
	SuggestedAction string               `json:"suggestedAction,omitempty"`
}

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

type SchedulingExplanation struct {
	PodName      string                      `json:"podName"`
	Namespace    string                      `json:"namespace"`
	Status       string                      `json:"status"`
	NodeName     string                      `json:"nodeName,omitempty"`
	NodeAnalysis []NodeSchedulingExplanation `json:"nodeAnalysis"`
	Summary      SchedulingSummary           `json:"summary"`
	Events       []SchedulingEvent           `json:"events,omitempty"`
}

type NodeSchedulingExplanation struct {
	NodeName          string                    `json:"nodeName"`
	Schedulable       bool                      `json:"schedulable"`
	Reasons           NodeSchedulingReasons     `json:"reasons"`
	Score             int32                     `json:"score,omitempty"`
	Recommendation    string                    `json:"recommendation,omitempty"`
}

type NodeSchedulingReasons struct {
	NodeReady    *NodeReadyExplanation    `json:"nodeReady,omitempty"`
	Resources    *ResourceExplanation     `json:"resources,omitempty"`
	Affinity     *AffinityExplanation     `json:"affinity,omitempty"`
	Taints       *TaintExplanation        `json:"taints,omitempty"`
	PodAffinity  *PodAffinityExplanation  `json:"podAffinity,omitempty"`
	Volume       *VolumeExplanation       `json:"volume,omitempty"`
}

type NodeReadyExplanation struct {
	Ready      bool     `json:"ready"`
	Conditions []string `json:"conditions,omitempty"`
}

type ResourceExplanation struct {
	Fits    bool                          `json:"fits"`
	Details map[string]ResourceDetail     `json:"details"`
	Summary string                        `json:"summary,omitempty"`
}

type ResourceDetail struct {
	PodRequests      string  `json:"podRequests"`
	NodeCapacity     string  `json:"nodeCapacity"`
	NodeAllocatable  string  `json:"nodeAllocatable"`
	NodeAllocated    string  `json:"nodeAllocated"`
	NodeAvailable    string  `json:"nodeAvailable"`
	Shortage         string  `json:"shortage,omitempty"`
	PercentUsed      float64 `json:"percentUsed"`
	Recommendation   string  `json:"recommendation,omitempty"`
}

type AffinityExplanation struct {
	NodeSelector      *SelectorExplanation   `json:"nodeSelector,omitempty"`
	NodeAffinity      *NodeAffinityDetail    `json:"nodeAffinity,omitempty"`
	Summary           string                 `json:"summary,omitempty"`
}

type SelectorExplanation struct {
	Matched       bool              `json:"matched"`
	Required      map[string]string `json:"required"`
	NodeLabels    map[string]string `json:"nodeLabels"`
	MissingLabels []string          `json:"missingLabels,omitempty"`
	Details       string            `json:"details,omitempty"`
}

type NodeAffinityDetail struct {
	RequiredMatched  bool     `json:"requiredMatched"`
	PreferredScore   int32    `json:"preferredScore,omitempty"`
	FailedTerms      []string `json:"failedTerms,omitempty"`
	Details          string   `json:"details,omitempty"`
}

type TaintExplanation struct {
	Tolerated         bool        `json:"tolerated"`
	NodeTaints        []TaintInfo `json:"nodeTaints"`
	PodTolerations    []string    `json:"podTolerations"`
	UntoleratedTaints []TaintInfo `json:"untoleratedTaints,omitempty"`
	Details           string      `json:"details,omitempty"`
}

type PodAffinityExplanation struct {
	Satisfied          bool     `json:"satisfied"`
	ConflictingPods    []string `json:"conflictingPods,omitempty"`
	RequiredNotMet     []string `json:"requiredNotMet,omitempty"`
	AntiAffinityFailed []string `json:"antiAffinityFailed,omitempty"`
	Details            string   `json:"details,omitempty"`
}

type VolumeExplanation struct {
	Satisfied       bool     `json:"satisfied"`
	Issues          []string `json:"issues,omitempty"`
	Details         string   `json:"details,omitempty"`
}

type SchedulingSummary struct {
	TotalNodes               int      `json:"totalNodes"`
	FilteredByNodeSelector   int      `json:"filteredByNodeSelector"`
	FilteredByNodeAffinity   int      `json:"filteredByNodeAffinity"`
	FilteredByTaints         int      `json:"filteredByTaints"`
	FilteredByResources      int      `json:"filteredByResources"`
	FilteredByPodAffinity    int      `json:"filteredByPodAffinity"`
	FilteredByVolume         int      `json:"filteredByVolume"`
	FilteredByNodeNotReady   int      `json:"filteredByNodeNotReady"`
	Recommendation           string   `json:"recommendation"`
	PossibleActions          []string `json:"possibleActions,omitempty"`
}
