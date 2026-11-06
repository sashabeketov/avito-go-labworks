package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/sashabeketov/avito-go-labworks/internal/config"
)

type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	Propagator     propagation.TextMapPropagator
}

func New(ctx context.Context, cfg config.TelemetryConfig) (*Telemetry, error) {
	traceEndpoint, err := url.JoinPath(cfg.Endpoint, "v1", "traces")
	if err != nil {
		return nil, fmt.Errorf("build trace endpoint: %w", err)
	}
	metricEndpoint, err := url.JoinPath(cfg.Endpoint, "v1", "metrics")
	if err != nil {
		return nil, fmt.Errorf("build metric endpoint: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", buildVersion()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(traceEndpoint))
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}
	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(metricEndpoint))
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		return nil, fmt.Errorf("create metric exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSampler(sampler(cfg.Sampler)),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))),
	)
	return &Telemetry{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		Propagator:     propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	}, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.MeterProvider.Shutdown(ctx), t.TracerProvider.Shutdown(ctx))
}

func (t *Telemetry) Meter(name string) metric.Meter {
	return t.MeterProvider.Meter(name)
}

func (t *Telemetry) Tracer(name string) trace.Tracer {
	return t.TracerProvider.Tracer(name)
}

func sampler(value string) sdktrace.Sampler {
	switch value {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}
