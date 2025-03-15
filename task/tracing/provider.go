package tracing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracerProvider is a wrapper around the OpenTelemetry TracerProvider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
}

// NewTracerProvider creates a new TracerProvider with Jaeger exporter
func NewTracerProvider(serviceName string, jaegerEndpoint string, timeout time.Duration) (*TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create a connection to the Jaeger collector using the recommended approach
	// Note: Since we're using a recent version of gRPC, we'll continue using DialContext
	// with WithBlock() which is still supported in 1.x versions
	conn, err := grpc.DialContext(ctx, jaegerEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Block until connection is established or timeout occurs
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Create the OTLP exporter
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create a resource describing the service
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create the trace provider with the exporter
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set the global trace provider
	otel.SetTracerProvider(provider)

	return &TracerProvider{
		provider: provider,
	}, nil
}

// Shutdown stops the trace provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := tp.provider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown provider: %w", err)
	}
	return nil
}

// Tracer returns a named tracer from the provider
func (tp *TracerProvider) Tracer(name string) trace.Tracer {
	return tp.provider.Tracer(name)
}

// DefaultJaegerEndpoint returns the default Jaeger endpoint
func DefaultJaegerEndpoint() string {
	return "localhost:4317" // Default OTLP gRPC endpoint for Jaeger
}

// InitTracing initializes the global tracer provider with default settings
func InitTracing(serviceName string) (*TracerProvider, error) {
	slog.Info("Initializing Jaeger tracing", "service", serviceName, "endpoint", DefaultJaegerEndpoint())

	// Use a 10-second timeout for connection establishment
	tp, err := NewTracerProvider(serviceName, DefaultJaegerEndpoint(), 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	slog.Info("Jaeger tracing initialized successfully", "service", serviceName)
	return tp, nil
}
