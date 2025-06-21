package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sumandas0/k8s-cluster-agent/internal/config"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/factory"
	"github.com/sumandas0/k8s-cluster-agent/internal/kubernetes"
	"github.com/sumandas0/k8s-cluster-agent/internal/logging"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/router"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logging.NewLogger(cfg)
	logger.Info("starting k8s-cluster-agent")

	// Initialize Kubernetes clients
	k8sClients, err := kubernetes.NewClients(cfg.K8sTimeout)
	if err != nil {
		logger.Error("failed to initialize Kubernetes clients", "error", err)
		os.Exit(1)
	}

	// Create services
	services := factory.NewServices(k8sClients, cfg, logger)

	// Create router
	r := router.NewRouter(services, logger)

	// Create HTTP server
	httpServer := server.New(cfg, r, logger)

	// Start server in a goroutine
	go func() {
		if err := httpServer.Start(); err != nil {
			logger.Error("failed to start HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("received shutdown signal")

	// Create shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Shutdown server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown server gracefully", "error", err)
		os.Exit(1)
	}

	logger.Info("server shutdown complete")
}
