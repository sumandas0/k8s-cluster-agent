package models

import "time"

type ClusterIssues struct {
	TotalPods         int                        `json:"totalPods"`
	HealthyPods       int                        `json:"healthyPods"`
	UnhealthyPods     int                        `json:"unhealthyPods"`
	IssueCategories   map[string]int             `json:"issueCategories"`
	IssuesByNamespace map[string]NamespaceIssues `json:"issuesByNamespace"`
	TopIssues         []IssueSummary             `json:"topIssues"`
	IssueVelocity     IssueVelocity              `json:"issueVelocity"`
	Patterns          []IssuePattern             `json:"patterns"`
	CriticalIssues    []ClusterPodIssue          `json:"criticalIssues"`
	CalculatedAt      time.Time                  `json:"calculatedAt"`
}

type NamespaceIssues struct {
	Namespace     string            `json:"namespace"`
	TotalPods     int               `json:"totalPods"`
	IssuesCount   int               `json:"issuesCount"`
	CriticalCount int               `json:"criticalCount"`
	WarningCount  int               `json:"warningCount"`
	TopIssues     []ClusterPodIssue `json:"topIssues,omitempty"`
}

type IssueSummary struct {
	Category     string   `json:"category"`
	Count        int      `json:"count"`
	Severity     string   `json:"severity"`
	Description  string   `json:"description"`
	AffectedPods []string `json:"affectedPods"`
}

type IssueVelocity struct {
	NewIssuesLastHour int     `json:"newIssuesLastHour"`
	NewIssuesLast24h  int     `json:"newIssuesLast24h"`
	ResolvedLastHour  int     `json:"resolvedLastHour"`
	ResolvedLast24h   int     `json:"resolvedLast24h"`
	TrendDirection    string  `json:"trendDirection"`
	VelocityPerHour   float64 `json:"velocityPerHour"`
}

type IssuePattern struct {
	Type         string            `json:"type"`
	Description  string            `json:"description"`
	Count        int               `json:"count"`
	Namespaces   []string          `json:"namespaces"`
	CommonLabels map[string]string `json:"commonLabels,omitempty"`
	FirstSeen    time.Time         `json:"firstSeen"`
	LastSeen     time.Time         `json:"lastSeen"`
}

type ClusterPodIssue struct {
	PodName       string    `json:"podName"`
	Namespace     string    `json:"namespace"`
	Category      string    `json:"category"`
	Severity      string    `json:"severity"`
	Reason        string    `json:"reason"`
	Message       string    `json:"message"`
	Count         int       `json:"count"`
	FirstSeen     time.Time `json:"firstSeen"`
	LastSeen      time.Time `json:"lastSeen"`
	IsRecurring   bool      `json:"isRecurring"`
	NodeName      string    `json:"nodeName,omitempty"`
	ContainerName string    `json:"containerName,omitempty"`
}

const (
	IssueCategoryCrashLoop     = "CrashLoopBackOff"
	IssueCategoryImagePull     = "ImagePullError"
	IssueCategoryPending       = "PendingScheduling"
	IssueCategoryOOMKilled     = "OOMKilled"
	IssueCategoryEvicted       = "Evicted"
	IssueCategoryFailed        = "Failed"
	IssueCategoryUnhealthy     = "Unhealthy"
	IssueCategoryInitError     = "InitContainerError"
	IssueCategoryVolumeMount   = "VolumeMountError"
	IssueCategoryConfigError   = "ConfigurationError"
	IssueCategoryNetworkError  = "NetworkError"
	IssueCategoryResourceQuota = "ResourceQuotaExceeded"

	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"

	TrendImproving = "improving"
	TrendStable    = "stable"
	TrendDegrading = "degrading"
)

