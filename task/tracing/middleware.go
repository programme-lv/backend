package tracing

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware adds tracing to HTTP handlers
type TracingMiddleware struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

// NewTracingMiddleware creates a new tracing middleware
func NewTracingMiddleware(serviceName string) *TracingMiddleware {
	return &TracingMiddleware{
		tracer: otel.Tracer(serviceName),
		propagator: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	}
}

// Middleware returns an http.Handler middleware function
func (tm *TracingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract context from the incoming request
		ctx := tm.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Start a new span
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := tm.tracer.Start(ctx, spanName)
		defer span.End()

		// Set span attributes
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.host", r.Host),
			attribute.String("http.user_agent", r.UserAgent()),
			attribute.String("http.remote_addr", r.RemoteAddr),
		)

		// Create a wrapped response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200 OK
		}

		// Call the next handler with the traced context
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Record response status
		span.SetAttributes(attribute.Int("http.status_code", rw.statusCode))

		// If status code is 4xx or 5xx, mark span as error
		if rw.statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", rw.statusCode))
		}
	})
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before writing it
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures a 200 status code if WriteHeader hasn't been called yet
func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// Unwrap returns the original ResponseWriter
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
