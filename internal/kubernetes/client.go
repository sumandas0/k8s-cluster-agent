package kubernetes

import (
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Clients struct {
	Kubernetes kubernetes.Interface
	Metrics    metricsclientset.Interface
}

func NewClients(timeout time.Duration) (*Clients, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
	}

	config.Timeout = timeout

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	metricsClient, err := metricsclientset.NewForConfig(config)
	if err != nil {
		slog.Warn("failed to create metrics client", "error", err)
		metricsClient = nil
	}

	return &Clients{
		Kubernetes: k8sClient,
		Metrics:    metricsClient,
	}, nil
}
