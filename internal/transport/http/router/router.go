package router

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/handlers"
	customMiddleware "github.com/sumandas0/k8s-cluster-agent/internal/transport/http/middleware"
)

// NewRouter creates and configures the Chi router
func NewRouter(services *core.Services, logger *slog.Logger) chi.Router {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.RecoveryMiddleware(logger))
	r.Use(customMiddleware.LoggingMiddleware(logger))
	r.Use(customMiddleware.TimeoutMiddleware(500 * time.Millisecond))

	// Create handlers
	podHandlers := handlers.NewPodHandlers(services.Pod, logger)
	nodeHandlers := handlers.NewNodeHandlers(services.Node, logger)
	namespaceHandlers := handlers.NewNamespaceHandlers(services.Namespace, logger)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Pod endpoints
		r.Route("/pods/{namespace}/{podName}", func(r chi.Router) {
			r.Get("/describe", podHandlers.GetPodDescribe)
			r.Get("/scheduling", podHandlers.GetPodScheduling)
			r.Get("/resources", podHandlers.GetPodResources)
			r.Get("/failure-events", podHandlers.GetPodFailureEvents)
		})

		// Node endpoints
		r.Get("/nodes/{nodeName}/utilization", nodeHandlers.GetNodeUtilization)

		// Namespace endpoints
		r.Get("/namespace/{namespace}/error", namespaceHandlers.GetNamespaceErrors)
	})

	// Health checks
	r.Get("/healthz", handlers.HandleHealth)
	r.Get("/readyz", handlers.HandleReadiness)

	return r
}
