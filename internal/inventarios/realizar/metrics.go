package realizar

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics agrupa los instrumentos OTel del dominio registrar valores en conteo
type Metrics struct {
	// inventario.realizar.registrar_valor.duration: histograma de duración en ms
	RegistrarDuration metric.Float64Histogram

	// inventario.realizar.registrar_valor.total: contador de registros por resultado
	RegistrarTotal metric.Int64Counter

	// inventario.realizar.items_completados: gauge de items completados
	ItemsCompletados metric.Int64UpDownCounter
}

// NewMetrics inicializa los instrumentos OTel del dominio realizar
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("loopi-api/inventarios/realizar")

	registrarDuration, err := meter.Float64Histogram(
		"inventario.realizar.registrar_valor.duration",
		metric.WithDescription("Duración de registro de valor en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	registrarTotal, err := meter.Int64Counter(
		"inventario.realizar.registrar_valor.total",
		metric.WithDescription("Conteo de registros de valor por resultado"),
	)
	if err != nil {
		return nil, err
	}

	itemsCompletados, err := meter.Int64UpDownCounter(
		"inventario.realizar.items_completados",
		metric.WithDescription("Cantidad de items completados en conteo"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		RegistrarDuration: registrarDuration,
		RegistrarTotal:    registrarTotal,
		ItemsCompletados:  itemsCompletados,
	}, nil
}

// noopMetrics devuelve métricas no-op cuando OTel no está configurado
func noopMetrics() *Metrics {
	meter := otel.GetMeterProvider().Meter("loopi-api/inventarios/realizar")

	registrarDuration, _ := meter.Float64Histogram("inventario.realizar.registrar_valor.duration")
	registrarTotal, _ := meter.Int64Counter("inventario.realizar.registrar_valor.total")
	itemsCompletados, _ := meter.Int64UpDownCounter("inventario.realizar.items_completados")

	return &Metrics{
		RegistrarDuration: registrarDuration,
		RegistrarTotal:    registrarTotal,
		ItemsCompletados:  itemsCompletados,
	}
}

// RecordRegistrar registra duración y resultado de un registro de valor
func (m *Metrics) RecordRegistrar(ctx context.Context, tiendaID *int64, duracionMs float64, resultado string) {
	attrs := metric.WithAttributes(
		attribute.String("resultado", resultado),
	)
	if tiendaID != nil {
		attrs = metric.WithAttributes(
			attribute.String("resultado", resultado),
			attribute.Int64("tienda_id", *tiendaID),
		)
	}

	m.RegistrarDuration.Record(ctx, duracionMs, attrs)
	m.RegistrarTotal.Add(ctx, 1, attrs)
}

// RecordItemsCompletados registra cambio en cantidad de items completados
func (m *Metrics) RecordItemsCompletados(ctx context.Context, inventarioID int64, delta int64) {
	attrs := metric.WithAttributes(
		attribute.Int64("inventario_id", inventarioID),
	)
	m.ItemsCompletados.Add(ctx, delta, attrs)
}
