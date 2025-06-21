package factory

import (
	"log/slog"

	"github.com/sumandas0/k8s-cluster-agent/internal/config"
	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/services"
	"github.com/sumandas0/k8s-cluster-agent/internal/kubernetes"
)

func NewServices(clients *kubernetes.Clients, cfg *config.Config, logger *slog.Logger) *core.Services {
	return &core.Services{
		Pod:       services.NewPodService(clients.Kubernetes, logger),
		Node:      services.NewNodeService(clients.Kubernetes, clients.Metrics, logger),
		Namespace: services.NewNamespaceService(clients.Kubernetes, cfg, logger),
	}
}
