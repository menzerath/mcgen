package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	middleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// http metrics tracked per-route
var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http",
		Name:      "request_duration_seconds",
		Help:      "Duration of HTTP requests in seconds.",
	}, []string{"method", "path"})
)

// prometheusMiddleware records per-route HTTP request counts and durations.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		duration := time.Since(start)

		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		httpRequestsTotal.WithLabelValues(r.Method, routePattern, fmt.Sprintf("%d", ww.Status())).Inc()
		httpRequestDuration.WithLabelValues(r.Method, routePattern).Observe(duration.Seconds())
	})
}
