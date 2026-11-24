package shared

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal menghitung total request HTTP berdasarkan method, path, dan status code
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by method, path, and response status code.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDurationSeconds mencatat durasi latensi request HTTP dalam detik
	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response latency for HTTP requests in seconds.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// HTTPActiveRequests memantau jumlah request aktif (in-flight) secara real-time
	HTTPActiveRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of active in-flight HTTP requests.",
		},
	)

	// HTTPResponseSizeBytes mencatat ukuran byte payload response HTTP
	HTTPResponseSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Histogram of response payload sizes in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6), // 100B, 1KB, 10KB, 100KB, 1MB, 10MB
		},
		[]string{"method", "path"},
	)
)

// MetricsMiddleware merekam metrik Prometheus untuk setiap incoming HTTP request
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Abaikan rekaman metrik untuk endpoint /metrics itu sendiri
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		HTTPActiveRequests.Inc()
		defer HTTPActiveRequests.Dec()

		start := time.Now()
		wrapped := newResponseWriter(w)

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// Gunakan registered pattern jika ada agar label tidak meledak (high cardinality)
		path := r.URL.Path
		if r.Pattern != "" {
			path = r.Pattern
		}

		statusStr := strconv.Itoa(wrapped.statusCode)

		HTTPRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		HTTPRequestDurationSeconds.WithLabelValues(r.Method, path).Observe(duration.Seconds())
		HTTPResponseSizeBytes.WithLabelValues(r.Method, path).Observe(float64(wrapped.bytesWritten))
	})
}
