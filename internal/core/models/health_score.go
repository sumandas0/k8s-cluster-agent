package models

import "time"

type PodHealthScore struct {
	PodName      string                     `json:"podName"`
	Namespace    string                     `json:"namespace"`
	OverallScore int                        `json:"overallScore"`
	Status       string                     `json:"status"`
	Components   map[string]HealthComponent `json:"components"`
	CalculatedAt time.Time                  `json:"calculatedAt"`
	Details      HealthDetails              `json:"details"`
}

type HealthComponent struct {
	Name        string  `json:"name"`
	Score       int     `json:"score"`
	Weight      float64 `json:"weight"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
}

type HealthDetails struct {
	RestartCount      int32             `json:"restartCount"`
	RestartFrequency  string            `json:"restartFrequency,omitempty"`
	Uptime            string            `json:"uptime"`
	LastRestartTime   *time.Time        `json:"lastRestartTime,omitempty"`
	LastRestartReason string            `json:"lastRestartReason,omitempty"`
	ContainerStatuses []ContainerHealth `json:"containerStatuses"`
	RecentEvents      []EventSummary    `json:"recentEvents"`
	PodConditions     []ConditionStatus `json:"podConditions"`
}

type ContainerHealth struct {
	Name         string `json:"name"`
	State        string `json:"state"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	ExitCode     *int32 `json:"exitCode,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type EventSummary struct {
	Type     string    `json:"type"`
	Reason   string    `json:"reason"`
	Message  string    `json:"message"`
	Count    int32     `json:"count"`
	LastSeen time.Time `json:"lastSeen"`
}

type ConditionStatus struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

func (h *PodHealthScore) GetHealthStatus() string {
	switch {
	case h.OverallScore >= 90:
		return "Healthy"
	case h.OverallScore >= 70:
		return "Good"
	case h.OverallScore >= 50:
		return "Warning"
	case h.OverallScore >= 30:
		return "Degraded"
	default:
		return "Critical"
	}
}
