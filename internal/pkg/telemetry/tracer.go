// Package telemetry provides a single SetupTracer function that initialises
// the OpenTelemetry SDK and wires it to an OTLP gRPC exporter.
//
// Call it once at the top of main(), defer the returned shutdown function,
// and every span created anywhere in the process will be exported automatically.
//
//	shutdown, err := telemetry.SetupTracer(ctx, "my-service")
//	if err != nil { ... }
//	defer shutdown(context.Background())
package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ShutdownFunc must be called before the process exits to flush any
// buffered spans and close the exporter connection cleanly.
type ShutdownFunc func(ctx context.Context) error

// SetupTracer initialises the global OpenTelemetry TracerProvider and
// TextMapPropagator for the given service name.
//
// The OTLP endpoint is read from the OTEL_EXPORTER_OTLP_ENDPOINT env var
// (default: "localhost:4317"). This matches the standard OTel env convention
// so no code change is needed between local and production environments.
func SetupTracer(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	endpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	// Strip the "http://" prefix if present — the gRPC dialer expects host:port.
	endpoint = stripScheme(endpoint)

	// ── 1. Create the OTLP gRPC exporter ────────────────────────────────────
	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: failed to dial OTel Collector at %s: %w", endpoint, err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("telemetry: failed to create OTLP trace exporter: %w", err)
	}

	// ── 2. Build the resource (identifies this service in Tempo / Grafana) ───
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(getEnv("OTEL_RESOURCE_ATTRIBUTES_ENV", "local")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: failed to build resource: %w", err)
	}

	// ── 3. Create the TracerProvider with a batching span processor ──────────
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		// Sample every request in local dev. In production use:
		//   sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1))
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// ── 4. Register as the global provider ───────────────────────────────────
	// This is what otelgrpc and otelhttp read internally — no need to pass
	// the provider around manually.
	otel.SetTracerProvider(tp)

	// ── 5. Register the W3C TraceContext + Baggage propagators ───────────────
	// This is the piece that makes trace_id flow across process boundaries.
	// otelgrpc injects/extracts these headers automatically once registered.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C traceparent / tracestate headers
		propagation.Baggage{},      // W3C baggage header
	))

	shutdown := func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("telemetry: error shutting down TracerProvider: %w", err)
		}
		return conn.Close()
	}

	return shutdown, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// stripScheme removes "http://" or "https://" prefixes so the raw host:port
// string can be used directly with grpc.NewClient.
func stripScheme(endpoint string) string {
	for _, prefix := range []string{"http://", "https://"} {
		if len(endpoint) > len(prefix) && endpoint[:len(prefix)] == prefix {
			return endpoint[len(prefix):]
		}
	}
	return endpoint
}
