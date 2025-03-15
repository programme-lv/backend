#!/bin/bash

# Check if Jaeger is already running
if docker ps | grep -q jaeger; then
  echo "Jaeger is already running"
  exit 0
fi

# Run Jaeger
echo "Starting Jaeger..."
docker run --rm --name jaeger \
  -d \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 5778:5778 \
  -p 9411:9411 \
  jaegertracing/jaeger:2.4.0

echo "Jaeger is now running"
echo "UI available at: http://localhost:16686"
echo "OTLP gRPC endpoint: localhost:4317"
echo "OTLP HTTP endpoint: localhost:4318"
echo ""
echo "To enable tracing in the application, set:"
echo "ENABLE_JAEGER_TRACING=true"
echo ""
echo "IMPORTANT: When tracing is enabled, the application will not start"
echo "           if it cannot connect to Jaeger."
echo ""
echo "To customize the connection timeout (default is 10s), set:"
echo "JAEGER_CONNECTION_TIMEOUT=15s"
echo ""
echo "To stop Jaeger, run: docker stop jaeger" 