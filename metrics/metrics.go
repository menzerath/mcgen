// Package metrics provides Prometheus metrics for mcgen.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace          = "mcgen"
	subsystemGenerator = "generator"
)

// all our metrics
var (
	AchievementGenerationRuntime = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystemGenerator,
		Name:      "runtime",
		Help:      "How long it took to generate an achievement image in seconds.",
		Buckets:   prometheus.ExponentialBuckets(0.0001, 1.4, 30),
	})
)

// ExposeMetrics starts a http server to serve prometheus metrics
func ExposeMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9100", nil)
}
