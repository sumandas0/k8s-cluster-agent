package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/sumandas0/k8s-cluster-agent/internal/transport/http/responses"
)

// TimeoutMiddleware creates a middleware that enforces request timeout
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal when the handler is done
			done := make(chan struct{})

			go func() {
				// Call the next handler with the timeout context
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Handler completed successfully
				return
			case <-ctx.Done():
				// Timeout occurred
				if ctx.Err() == context.DeadlineExceeded {
					responses.WriteTimeout(w, "Request timeout")
				}
			}
		})
	}
}
