package trace

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	forceFlushTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
	samplingRate      = 100.0
)

func NewTracerProvider(
	ctx context.Context,
	name, version string,
) (trace.TracerProvider, func()) {
	if _, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); !ok {
		slog.Warn("trace: OTEL_EXPORTER_OTLP_ENDPOINT is not set. use noop exporter")
		return noop.NewTracerProvider(), func() {}
	}
	slog.Info("trace: create otlp http exporter", slog.String("endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")))

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingRate))
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		slog.Error("trace: failed to create otlp http exporter. use noop exporter instead", slog.String("error", err.Error()))
		return noop.NewTracerProvider(), func() {}
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(name),
			semconv.ServiceVersionKey.String(version),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), forceFlushTimeout)
		defer cancel()
		if err := tp.ForceFlush(ctx); err != nil {
			slog.Error("trace: failed to force flush spans", slog.String("error", err.Error()))
		}
		sctx, scancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer scancel()
		if err := tp.Shutdown(sctx); err != nil {
			slog.Error("trace: failed to shutdown tracer provider", slog.String("error", err.Error()))
		}
	}
	return tp, cleanup
}
