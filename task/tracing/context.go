package tracing

import (
	"context"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ContextPropagator handles propagation of trace context between services
type ContextPropagator struct {
	propagator propagation.TextMapPropagator
}

// NewContextPropagator creates a new context propagator
func NewContextPropagator() *ContextPropagator {
	return &ContextPropagator{
		propagator: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	}
}

// Extract extracts trace context from carrier into context
func (p *ContextPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return p.propagator.Extract(ctx, carrier)
}

// Inject injects trace context from context into carrier
func (p *ContextPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	p.propagator.Inject(ctx, carrier)
}

// HeaderCarrier implements TextMapCarrier for HTTP headers
type HeaderCarrier map[string]string

// Get returns the value for the given key
func (c HeaderCarrier) Get(key string) string {
	return c[key]
}

// Set sets the value for the given key
func (c HeaderCarrier) Set(key, value string) {
	c[key] = value
}

// Keys returns all keys in the carrier
func (c HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// GetTraceID extracts the trace ID from context if present
func GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.TraceID().String()
}

// GetSpanID extracts the span ID from context if present
func GetSpanID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.SpanID().String()
}

// GetBaggageItem gets a baggage item from context
func GetBaggageItem(ctx context.Context, key string) string {
	bags := baggage.FromContext(ctx)
	if member := bags.Member(key); member.Key() != "" {
		return member.Value()
	}
	return ""
}
