package shared

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracingMiddleware(t *testing.T) {
	// Inisialisasi in-memory span recorder untuk pengujian tracer tanpa butuh network/Tempo
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracingMw := TracingMiddleware("test-service")

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID, _ := r.Context().Value(TraceIDKey).(string)
			if traceID == "" {
				t.Fatalf("expected trace_id in context, got empty")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
		RequestIDMiddleware,
		tracingMw,
	)

	req := httptest.NewRequest("GET", "/api/test-trace", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// 1. Verifikasi X-Trace-ID diset di response header
	traceIDHeader := rec.Header().Get("X-Trace-ID")
	if traceIDHeader == "" {
		t.Fatalf("expected X-Trace-ID header in response")
	}

	// 2. Verifikasi span tersimpan di recorder
	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 ended span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name() != "GET /api/test-trace" {
		t.Fatalf("expected span name 'GET /api/test-trace', got %s", span.Name())
	}

	if span.SpanContext().TraceID().String() != traceIDHeader {
		t.Fatalf("expected span TraceID %s to match header %s", span.SpanContext().TraceID().String(), traceIDHeader)
	}
}

func TestInitTracer_DisabledEndpoint(t *testing.T) {
	tp, shutdown, err := InitTracer(context.Background(), "test-service", "disabled")
	if err != nil {
		t.Fatalf("expected no error for disabled endpoint, got %v", err)
	}
	if tp == nil {
		t.Fatalf("expected non-nil TracerProvider")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("expected clean shutdown, got %v", err)
	}
}
