package core

import "errors"

var (
	ErrPodNotFound = errors.New("pod not found")

	ErrNodeNotFound = errors.New("node not found")

	ErrMetricsNotAvailable = errors.New("metrics server not available")
)
