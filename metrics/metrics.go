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
		Buckets: []float64{
			0.0001, // 100µs
			0.0002,
			0.0005,
			0.001, // 1ms
			0.002,
			0.005,
			0.01, // 10ms
			0.02,
			0.05,
			0.1, // 100 ms
			0.2,
			0.5,
			1.0, // 1s
			2.0,
			5.0,
		},
	})
)

// ExposeMetrics starts a http server to serve prometheus metrics
func ExposeMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9100", nil)
}
