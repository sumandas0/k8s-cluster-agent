package core

import "errors"

// Define domain-specific errors
var (
	// ErrPodNotFound is returned when a pod is not found
	ErrPodNotFound = errors.New("pod not found")

	// ErrNodeNotFound is returned when a node is not found
	ErrNodeNotFound = errors.New("node not found")

	// ErrMetricsNotAvailable is returned when metrics server is not available
	ErrMetricsNotAvailable = errors.New("metrics server not available")
)
