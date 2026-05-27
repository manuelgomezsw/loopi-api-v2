package auth

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const (
	// Nombre del instrumentador OTel para el paquete auth.
	otelScope = "loopi-api/auth"
)

// Metrics agrupa los instrumentos OTel del dominio de autenticación.
// Sigue el patrón RF-AUTH-06: métricas funcionales, no opcionales.
type Metrics struct {
	// auth.login.duration: histograma de duración del proceso de login en ms.
	LoginDuration metric.Float64Histogram

	// auth.login.result: contador por resultado (success, invalid_credentials, account_inactive, account_locked).
	LoginResult metric.Int64Counter

	// auth.blacklist.check.duration: histograma de latencia de consulta a tokens_revocados.
	BlacklistCheckDuration metric.Float64Histogram
}

// NewMetrics inicializa los instrumentos OTel del dominio auth.
// El MeterProvider se configura globalmente antes de llamar a NewMetrics.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(otelScope)

	loginDuration, err := meter.Float64Histogram(
		"auth.login.duration",
		metric.WithDescription("Duración del proceso de login en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	loginResult, err := meter.Int64Counter(
		"auth.login.result",
		metric.WithDescription("Conteo de resultados de login por etiqueta result"),
	)
	if err != nil {
		return nil, err
	}

	blacklistDuration, err := meter.Float64Histogram(
		"auth.blacklist.check.duration",
		metric.WithDescription("Latencia de la consulta a tokens_revocados en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		LoginDuration:          loginDuration,
		LoginResult:            loginResult,
		BlacklistCheckDuration: blacklistDuration,
	}, nil
}

// noopMetrics devuelve métricas no-op cuando OTel no está configurado.
// Evita nil checks en el handler.
func noopMetrics() *Metrics {
	meter := otel.GetMeterProvider().Meter(otelScope)

	loginDuration, _ := meter.Float64Histogram("auth.login.duration")
	loginResult, _ := meter.Int64Counter("auth.login.result")
	blacklistDuration, _ := meter.Float64Histogram("auth.blacklist.check.duration")

	return &Metrics{
		LoginDuration:          loginDuration,
		LoginResult:            loginResult,
		BlacklistCheckDuration: blacklistDuration,
	}
}

// RecordLogin registra duración y resultado de un intento de login.
func (m *Metrics) RecordLogin(ctx context.Context, duracionMs float64, resultado string) {
	attrs := metricAttrs("result", resultado)
	m.LoginDuration.Record(ctx, duracionMs, attrs)
	m.LoginResult.Add(ctx, 1, attrs)
}

// RecordBlacklistCheck registra la latencia de la consulta a tokens_revocados.
func (m *Metrics) RecordBlacklistCheck(ctx context.Context, duracionMs float64) {
	m.BlacklistCheckDuration.Record(ctx, duracionMs)
}
