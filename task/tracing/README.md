# Jaeger Tracing for Task Module

This package adds Jaeger v2 tracing support to the task module. It provides:

1. A tracer provider setup for Jaeger
2. A service wrapper that adds tracing to all task service methods
3. HTTP middleware for tracing HTTP requests
4. Context propagation utilities

## Usage

To enable Jaeger tracing, set the environment variable:

```
ENABLE_JAEGER_TRACING=true
```

### Connection Behavior

When Jaeger tracing is enabled, the application will **block on startup** while attempting to connect to the Jaeger collector. This ensures that tracing is properly initialized before the application starts handling requests.

To prevent indefinite hanging if Jaeger is not available, a timeout is applied to the connection attempt. The default timeout is 10 seconds, but you can customize it by setting:

```
JAEGER_CONNECTION_TIMEOUT=15s
```

**Important**: If the connection to Jaeger cannot be established within the timeout period, the application will log an error and **exit**. The application will not start without a successful connection to Jaeger when tracing is enabled.

### Implementation Note

The connection to Jaeger uses `grpc.DialContext` with `WithBlock()` to provide the blocking behavior with timeout. While `DialContext` is marked as deprecated in favor of `NewClient` in the gRPC documentation, it's still fully supported throughout the 1.x versions of gRPC and is the approach recommended in the OpenTelemetry Go SDK examples.

## Running Jaeger

You can run Jaeger using Docker:

```bash
docker run --rm --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 5778:5778 \
  -p 9411:9411 \
  jaegertracing/jaeger:2.4.0
```

Or use the provided script:

```bash
./scripts/run-jaeger.sh
```

## Accessing the Jaeger UI

Once Jaeger is running, you can access the UI at:

```
http://localhost:16686
```

## Implementation Details

### Tracer Provider

The tracer provider is initialized with the service name "task-service" and connects to the Jaeger collector at `localhost:4317` (default OTLP gRPC endpoint). The connection is established with a blocking call that has a configurable timeout.

### Service Wrapper

The service wrapper adds tracing to all methods of the `TaskSrvcClient` interface. Each method creates a span with relevant attributes and propagates the context.

### HTTP Middleware

The HTTP middleware adds tracing to HTTP handlers. It extracts trace context from incoming requests, creates spans for each request, and adds relevant HTTP attributes.

### Context Propagation

The context propagation utilities help extract and inject trace context between services, particularly useful for distributed tracing across service boundaries. 