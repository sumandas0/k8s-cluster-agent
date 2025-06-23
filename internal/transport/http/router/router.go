package router

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/handlers"
	customMiddleware "github.com/sumandas0/k8s-cluster-agent/internal/transport/http/middleware"
	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/openapi"
)

func NewRouter(services *core.Services, logger *slog.Logger) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.RecoveryMiddleware(logger))
	r.Use(customMiddleware.LoggingMiddleware(logger))
	r.Use(customMiddleware.TimeoutMiddleware(5 * time.Second))

	podHandlers := handlers.NewPodHandlers(services.Pod, logger)
	nodeHandlers := handlers.NewNodeHandlers(services.Node, logger)
	namespaceHandlers := handlers.NewNamespaceHandlers(services.Namespace, logger)
	healthScoreHandler := handlers.NewHealthScoreHandler(services.HealthScore, logger)
	clusterIssuesHandler := handlers.NewClusterIssuesHandler(services.ClusterIssues, logger)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/pods/{namespace}/{podName}", func(r chi.Router) {
			r.Get("/describe", podHandlers.GetPodDescribe)
			r.Get("/scheduling", podHandlers.GetPodScheduling)
			r.Get("/resources", podHandlers.GetPodResources)
			r.Get("/failure-events", podHandlers.GetPodFailureEvents)
			r.Get("/scheduling/explain", podHandlers.GetPodSchedulingExplanation)
			r.Get("/health-score", healthScoreHandler.GetPodHealthScore)
		})

		r.Get("/nodes/{nodeName}/utilization", nodeHandlers.GetNodeUtilization)

		r.Get("/namespace/{namespace}/error", namespaceHandlers.GetNamespaceErrors)

		r.Get("/cluster/pod-issues", clusterIssuesHandler.GetClusterIssues)
	})

	r.Get("/healthz", handlers.HandleHealth)
	r.Get("/readyz", handlers.HandleReadiness)

	r.Mount("/swagger", openapi.SwaggerHandler())

	return r
}
