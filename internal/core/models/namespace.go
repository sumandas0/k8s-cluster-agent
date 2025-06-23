package models

import (
	"time"
)

type PodIssueType string

const (
	PodIssueHighRestarts        PodIssueType = "HighRestarts"
	PodIssuePending             PodIssueType = "Pending"
	PodIssueFailed              PodIssueType = "Failed"
	PodIssueCrashLoop           PodIssueType = "CrashLoopBackOff"
	PodIssueImagePull           PodIssueType = "ImagePullError"
	PodIssueResourceConstraints PodIssueType = "ResourceConstraints"
	PodIssueUnschedulable       PodIssueType = "Unschedulable"
)

type PodIssue struct {
	Type        PodIssueType `json:"type"`
	Description string       `json:"description"`
	Severity    string       `json:"severity"`
	Details     string       `json:"details,omitempty"`
}

type ProblematicPod struct {
	Name         string        `json:"name"`
	Namespace    string        `json:"namespace"`
	OwnerKind    string        `json:"ownerKind"`
	OwnerName    string        `json:"ownerName"`
	Phase        string        `json:"phase"`
	Status       string        `json:"status,omitempty"`
	RestartCount int32         `json:"restartCount"`
	Age          time.Duration `json:"-"`
	AgeString    string        `json:"age"`
	Issues       []PodIssue    `json:"issues"`
	Events       []EventInfo   `json:"recentEvents,omitempty"`
}

type NamespaceErrorSummary struct {
	IssueType    PodIssueType `json:"issueType"`
	Count        int          `json:"count"`
	Description  string       `json:"description"`
	AffectedPods []string     `json:"affectedPods"`
}

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
