package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration in seconds.",
			Buckets: []float64{
				0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
			},
		},
		[]string{"method", "path", "status"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_response_size_bytes",
			Help: "HTTP response size in bytes.",
			Buckets: []float64{
				100, 500, 1_000, 5_000, 10_000, 50_000, 100_000, 500_000, 1_000_000, 5_000_000,
			},
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)
)

func metricsHandler() http.Handler {
	return promhttp.Handler()
}

func recordHTTPMetrics(r *http.Request, method string, status int, written int64, duration time.Duration) {
	path := routePattern(r)
	// /metrics: scrape noise. /subm-updates: long-lived SSE; duration is connection lifetime.
	if r.URL.Path == "/metrics" || path == "/subm-updates" {
		return
	}

	statusLabel := strconv.Itoa(status)
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": statusLabel,
	}

	httpRequestDuration.With(labels).Observe(duration.Seconds())
	httpResponseSize.With(labels).Observe(float64(written))
	httpRequestsTotal.With(labels).Inc()
}

func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if p := rc.RoutePattern(); p != "" {
			return p
		}
	}
	return "unmatched"
}
