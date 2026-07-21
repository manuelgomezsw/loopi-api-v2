package completar

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics agrupa los instrumentos OTel del dominio completar conteo
type Metrics struct {
	// inventario.completar.confirmar.duration: histograma de duración en ms
	ConfirmarDuration metric.Float64Histogram

	// inventario.completar.confirmar.total: contador de confirmaciones por resultado
	ConfirmarTotal metric.Int64Counter

	// inventario.completar.items_ajustados: gauge de items ajustados
	ItemsAjustados metric.Int64UpDownCounter

	// inventario.completar.diferencia_promedio: gauge de diferencia promedio
	DiferenciaPromedio metric.Float64Gauge
}

// NewMetrics inicializa los instrumentos OTel del dominio completar
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("loopi-api/inventarios/completar")

	confirmarDuration, err := meter.Float64Histogram(
		"inventario.completar.confirmar.duration",
		metric.WithDescription("Duración de confirmación de conteo en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	confirmarTotal, err := meter.Int64Counter(
		"inventario.completar.confirmar.total",
		metric.WithDescription("Conteo de confirmaciones de conteo por resultado"),
	)
	if err != nil {
		return nil, err
	}

	itemsAjustados, err := meter.Int64UpDownCounter(
		"inventario.completar.items_ajustados",
		metric.WithDescription("Cantidad de items ajustados en stock"),
	)
	if err != nil {
		return nil, err
	}

	diferenciaPromedio, err := meter.Float64Gauge(
		"inventario.completar.diferencia_promedio",
		metric.WithDescription("Diferencia promedio entre valor esperado y real"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		ConfirmarDuration:  confirmarDuration,
		ConfirmarTotal:     confirmarTotal,
		ItemsAjustados:     itemsAjustados,
		DiferenciaPromedio: diferenciaPromedio,
	}, nil
}

// noopMetrics devuelve métricas no-op cuando OTel no está configurado
func noopMetrics() *Metrics {
	meter := otel.GetMeterProvider().Meter("loopi-api/inventarios/completar")

	confirmarDuration, _ := meter.Float64Histogram("inventario.completar.confirmar.duration")
	confirmarTotal, _ := meter.Int64Counter("inventario.completar.confirmar.total")
	itemsAjustados, _ := meter.Int64UpDownCounter("inventario.completar.items_ajustados")
	diferenciaPromedio, _ := meter.Float64Gauge("inventario.completar.diferencia_promedio")

	return &Metrics{
		ConfirmarDuration:  confirmarDuration,
		ConfirmarTotal:     confirmarTotal,
		ItemsAjustados:     itemsAjustados,
		DiferenciaPromedio: diferenciaPromedio,
	}
}

// RecordConfirmar registra duración y resultado de una confirmación
func (m *Metrics) RecordConfirmar(ctx context.Context, tiendaID *int64, duracionMs float64, resultado string) {
	attrs := metric.WithAttributes(
		attribute.String("resultado", resultado),
	)
	if tiendaID != nil {
		attrs = metric.WithAttributes(
			attribute.String("resultado", resultado),
			attribute.Int64("tienda_id", *tiendaID),
		)
	}

	m.ConfirmarDuration.Record(ctx, duracionMs, attrs)
	m.ConfirmarTotal.Add(ctx, 1, attrs)
}

// RecordItemsAjustados registra cantidad de items ajustados
func (m *Metrics) RecordItemsAjustados(ctx context.Context, count int64, tiendaID *int64) {
	attrs := metric.WithAttributes(
		attribute.Int64("count", count),
	)
	if tiendaID != nil {
		attrs = metric.WithAttributes(
			attribute.Int64("count", count),
			attribute.Int64("tienda_id", *tiendaID),
		)
	}
	m.ItemsAjustados.Add(ctx, count, attrs)
}

// RecordDiferenciaPromedio registra la diferencia promedio observada
func (m *Metrics) RecordDiferenciaPromedio(ctx context.Context, promedio float64, tiendaID *int64) {
	attrs := metric.WithAttributes()
	if tiendaID != nil {
		attrs = metric.WithAttributes(
			attribute.Int64("tienda_id", *tiendaID),
		)
	}
	m.DiferenciaPromedio.Record(ctx, promedio, attrs)
}
