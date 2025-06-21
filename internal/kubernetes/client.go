package kubernetes

import (
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Clients holds the Kubernetes clientsets
type Clients struct {
	Kubernetes kubernetes.Interface
	Metrics    metricsclientset.Interface
}

// NewClients creates new Kubernetes clients using in-cluster configuration
func NewClients(timeout time.Duration) (*Clients, error) {
	// Load in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
	}

	// Set timeout
	config.Timeout = timeout

	// Create standard Kubernetes clientset
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Create metrics clientset
	// Don't fail if metrics client creation fails - metrics are optional
	metricsClient, err := metricsclientset.NewForConfig(config)
	if err != nil {
		// Log warning but don't fail - metrics are optional
		slog.Warn("failed to create metrics client", "error", err)
		metricsClient = nil
	}

	return &Clients{
		Kubernetes: k8sClient,
		Metrics:    metricsClient,
	}, nil
}
