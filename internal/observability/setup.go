package observability

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// Setup inicializa TracerProvider y MeterProvider globales con exporters OTLP/HTTP.
// Si OTEL_EXPORTER_OTLP_ENDPOINT está vacía, opera en modo no-op sin errores.
// El caller debe diferir la función shutdown retornada con un timeout de 5 s.
func Setup(ctx context.Context) (shutdown func(context.Context) error, err error) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Printf("[otel] error al exportar: %v", err)
	}))

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName()),
			semconv.ServiceVersion(os.Getenv("APP_VERSION")),
			semconv.DeploymentEnvironmentName(deploymentEnv()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: resource: %w", err)
	}

	headers := otlpHeaders()

	traceExp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	metricExp, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(endpoint),
		otlpmetrichttp.WithHeaders(headers),
		// Datadog solo acepta temporalidad delta; OTel usa cumulative por defecto.
		otlpmetrichttp.WithTemporalitySelector(func(_ metric.InstrumentKind) metricdata.Temporality {
			return metricdata.DeltaTemporality
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: metric exporter: %w", err)
	}
	mp := metric.NewMeterProvider(
		metric.WithReader(
			metric.NewPeriodicReader(skipEmptyExporter{metricExp}, metric.WithInterval(15*time.Second)),
		),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	return func(ctx context.Context) error {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil
	}, nil
}

// skipEmptyExporter evita enviar payloads vacíos a Datadog, que los rechaza con 400.
type skipEmptyExporter struct{ metric.Exporter }

func (e skipEmptyExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	if rm == nil || len(rm.ScopeMetrics) == 0 {
		return nil
	}
	return e.Exporter.Export(ctx, rm)
}

// otlpHeaders parsea OTEL_EXPORTER_OTLP_HEADERS (formato: "k1=v1,k2=v2").
func otlpHeaders() map[string]string {
	raw := os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")
	if raw == "" {
		return nil
	}
	headers := make(map[string]string)
	for pair := range strings.SplitSeq(raw, ",") {
		k, v, ok := strings.Cut(pair, "=")
		if ok {
			headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return headers
}

func serviceName() string {
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		return v
	}
	return "loopi-api"
}

func deploymentEnv() string {
	switch os.Getenv("ENV") {
	case "prod":
		return "production"
	case "stage":
		return "staging"
	default:
		return "development"
	}
}
