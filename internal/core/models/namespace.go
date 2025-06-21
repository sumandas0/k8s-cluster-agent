package models

import (
	"time"
)

// PodIssueType represents the type of issue a pod is experiencing
type PodIssueType string

const (
	// Pod is experiencing high restart counts
	PodIssueHighRestarts PodIssueType = "HighRestarts"
	// Pod is stuck in pending state
	PodIssuePending PodIssueType = "Pending"
	// Pod is in a failed state
	PodIssueFailed PodIssueType = "Failed"
	// Pod is in CrashLoopBackOff state
	PodIssueCrashLoop PodIssueType = "CrashLoopBackOff"
	// Pod has image pull issues
	PodIssueImagePull PodIssueType = "ImagePullError"
	// Pod has resource constraint issues
	PodIssueResourceConstraints PodIssueType = "ResourceConstraints"
	// Pod has unschedulable issues
	PodIssueUnschedulable PodIssueType = "Unschedulable"
)

// PodIssue represents a specific issue with a pod
type PodIssue struct {
	Type        PodIssueType `json:"type"`
	Description string       `json:"description"`
	Severity    string       `json:"severity"` // "critical", "warning", "info"
	Details     string       `json:"details,omitempty"`
}

// ProblematicPod represents a pod with issues
type ProblematicPod struct {
	Name         string        `json:"name"`
	Namespace    string        `json:"namespace"`
	OwnerKind    string        `json:"ownerKind"` // "Deployment", "StatefulSet"
	OwnerName    string        `json:"ownerName"`
	Phase        string        `json:"phase"`
	Status       string        `json:"status,omitempty"`
	RestartCount int32         `json:"restartCount"`
	Age          time.Duration `json:"-"`
	AgeString    string        `json:"age"`
	Issues       []PodIssue    `json:"issues"`
	Events       []EventInfo   `json:"recentEvents,omitempty"`
}

// NamespaceErrorSummary provides a summary of issues by type
type NamespaceErrorSummary struct {
	IssueType    PodIssueType `json:"issueType"`
	Count        int          `json:"count"`
	Description  string       `json:"description"`
	AffectedPods []string     `json:"affectedPods"`
}

// NamespaceErrorReport contains the full analysis of namespace errors
type NamespaceErrorReport struct {
	Namespace            string                  `json:"namespace"`
	AnalysisTime         time.Time               `json:"analysisTime"`
	TotalPodsAnalyzed    int                     `json:"totalPodsAnalyzed"`
	ProblematicPodsCount int                     `json:"problematicPodsCount"`
	HealthyPodsCount     int                     `json:"healthyPodsCount"`
	RestartThresholdUsed int                     `json:"restartThresholdUsed"`
	Summary              []NamespaceErrorSummary `json:"summary"`
	ProblematicPods      []ProblematicPod        `json:"problematicPods"`
	CriticalIssuesCount  int                     `json:"criticalIssuesCount"`
	WarningIssuesCount   int                     `json:"warningIssuesCount"`
}

