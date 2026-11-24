package shared

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	TraceIDKey contextKey = "trace_id"
	SpanIDKey  contextKey = "span_id"
)

// InitTracer menginisialisasi OpenTelemetry TracerProvider dengan OTLP gRPC exporter ke Grafana Tempo
func InitTracer(ctx context.Context, serviceName, tempoEndpoint string) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	// Set global W3C TraceContext + Baggage propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", "development"),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Jika tempoEndpoint kosong atau "disabled", gunakan provider tanpa exporter (graceful fallback)
	if tempoEndpoint == "" || tempoEndpoint == "disabled" {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)
		otel.SetTracerProvider(tp)
		return tp, tp.Shutdown, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(tempoEndpoint),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return tp, tp.Shutdown, nil
}

// TracingMiddleware membuat span OpenTelemetry untuk setiap request HTTP yang masuk
func TracingMiddleware(serviceName string) Middleware {
	tracer := otel.GetTracerProvider().Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Abaikan tracing untuk scraping /metrics
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract context W3C TraceContext dari incoming header jika ada
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			path := r.URL.Path
			if r.Pattern != "" {
				path = r.Pattern
			}

			spanName := fmt.Sprintf("%s %s", r.Method, path)
			ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()

			reqID := GetRequestID(ctx)
			if reqID == "" {
				reqID = GetRequestID(r.Context())
			}

			// Pasang atribut semantik OpenTelemetry
			attrs := []attribute.KeyValue{
				attribute.String("http.method", r.Method),
				attribute.String("http.route", path),
				attribute.String("http.target", r.URL.RequestURI()),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("client.ip", GetClientIP(r)),
			}
			if reqID != "" {
				attrs = append(attrs, attribute.String("app.request_id", reqID))
			}
			span.SetAttributes(attrs...)

			// Injeksi trace_id dan span_id ke header response dan context request
			if spanCtx := span.SpanContext(); spanCtx.IsValid() {
				traceID := spanCtx.TraceID().String()
				spanID := spanCtx.SpanID().String()

				w.Header().Set("X-Trace-ID", traceID)

				ctx = context.WithValue(ctx, TraceIDKey, traceID)
				ctx = context.WithValue(ctx, SpanIDKey, spanID)
			}

			wrapped := newResponseWriter(w)

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Rekam status kode ke span
			span.SetAttributes(attribute.Int("http.status_code", wrapped.statusCode))
			if wrapped.statusCode >= 500 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d error", wrapped.statusCode))
			} else {
				span.SetStatus(codes.Ok, "OK")
			}
		})
	}
}
