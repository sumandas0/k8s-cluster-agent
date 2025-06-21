package models

import (
	"time"
)

// NodeUtilization contains current resource utilization for a node
type NodeUtilization struct {
	NodeName         string    `json:"nodeName"`
	CPUUsage         string    `json:"cpuUsage"`
	CPUCapacity      string    `json:"cpuCapacity"`
	CPUPercentage    float64   `json:"cpuPercentage"`
	MemoryUsage      string    `json:"memoryUsage"`
	MemoryCapacity   string    `json:"memoryCapacity"`
	MemoryPercentage float64   `json:"memoryPercentage"`
	Timestamp        time.Time `json:"timestamp"`
}
