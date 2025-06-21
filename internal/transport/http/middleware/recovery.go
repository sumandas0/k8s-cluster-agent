package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

// RecoveryMiddleware creates a middleware that recovers from panics
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic with stack trace
					logger.Error("panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
						"request_id", middleware.GetReqID(r.Context()),
						"stack", string(debug.Stack()),
					)

					// Return internal server error
					responses.WriteInternalError(w, "Internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
