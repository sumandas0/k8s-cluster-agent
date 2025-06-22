package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/sumandas0/k8s-cluster-agent/docs"
	"github.com/sumandas0/k8s-cluster-agent/internal/config"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/factory"
	"github.com/sumandas0/k8s-cluster-agent/internal/kubernetes"
	"github.com/sumandas0/k8s-cluster-agent/internal/logging"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/router"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/server"
)

// @title K8s Cluster Agent API
// @version 1.0
// @description A lightweight, read-only Kubernetes service that provides a RESTful API for querying pod and node information
// @description
// @description The K8s Cluster Agent runs inside a Kubernetes cluster and provides various endpoints for:
// @description - Pod information (description, scheduling, resources, health)
// @description - Node utilization metrics
// @description - Cluster-wide pod issues dashboard
// @description - Namespace error analysis

// @contact.name K8s Cluster Agent Team
// @contact.url https://github.com/sumandas0/k8s-cluster-agent

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @schemes http https

// @tag.name Pods
// @tag.description Pod-related operations for querying pod status, scheduling, resources, and health

// @tag.name Nodes
// @tag.description Node-related operations for querying node utilization and metrics

// @tag.name Cluster
// @tag.description Cluster-wide operations for analyzing pod issues across namespaces

// @tag.name Namespace
// @tag.description Namespace-specific operations for error analysis

// @tag.name Health
// @tag.description Health check endpoints for monitoring service availability

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	logger := logging.NewLogger(cfg)
	logger.Info("starting k8s-cluster-agent")

	k8sClients, err := kubernetes.NewClients(cfg.K8sTimeout)
	if err != nil {
		logger.Error("failed to initialize Kubernetes clients", "error", err)
		os.Exit(1)
	}

	services := factory.NewServices(k8sClients, cfg, logger)

	r := router.NewRouter(services, logger)

	httpServer := server.New(cfg, r, logger)

	go func() {
		if err := httpServer.Start(); err != nil {
			logger.Error("failed to start HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("received shutdown signal")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown server gracefully", "error", err)
		os.Exit(1)
	}

	logger.Info("server shutdown complete")
}
