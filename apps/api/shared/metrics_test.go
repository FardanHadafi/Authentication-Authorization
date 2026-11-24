package shared

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetricsMiddleware_RecordsMetrics(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/test-metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("metrics test passed"))
	})
	mux.Handle("GET /metrics", promhttp.Handler())

	handler := Chain(mux, MetricsMiddleware)

	// 1. Eksekusi request ke endpoint uji
	req := httptest.NewRequest("GET", "/api/test-metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// 2. Scraping endpoint /metrics untuk memastikan metrik tercatat
	metricsReq := httptest.NewRequest("GET", "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)

	if metricsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from /metrics, got %d", metricsRec.Code)
	}

	body, err := io.ReadAll(metricsRec.Body)
	if err != nil {
		t.Fatalf("failed to read /metrics body: %v", err)
	}

	bodyStr := string(body)

	// Pastikan counter http_requests_total mencatat /api/test-metrics
	if !strings.Contains(bodyStr, "http_requests_total") {
		t.Fatalf("expected http_requests_total in metrics output")
	}
	if !strings.Contains(bodyStr, "path=\"/api/test-metrics\"") && !strings.Contains(bodyStr, "path=\"GET /api/test-metrics\"") {
		t.Fatalf("expected test-metrics path in http_requests_total, got output:\n%s", bodyStr)
	}

	// Pastikan histogram http_request_duration_seconds tercatat
	if !strings.Contains(bodyStr, "http_request_duration_seconds") {
		t.Fatalf("expected http_request_duration_seconds in metrics output")
	}

	// Pastikan gauge http_active_requests tersedia
	if !strings.Contains(bodyStr, "http_active_requests") {
		t.Fatalf("expected http_active_requests in metrics output")
	}
}
