package iniciar

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	otelScope = "loopi-api/inventarios/iniciar"
)

// Metrics agrupa los instrumentos OTel del dominio iniciar conteo.
type Metrics struct {
	// inventario.iniciar.crear.duration: histograma de duración de creación en ms.
	CrearDuration metric.Float64Histogram

	// inventario.iniciar.crear.total: contador de creaciones por resultado.
	CrearTotal metric.Int64Counter

	// inventario.iniciar.crear.items_count: gauge de cantidad de items en el conteo.
	ItemsCount metric.Int64Gauge

	// inventario.iniciar.determinar_tipo.duration: histograma de duración de determinación de tipo en ms.
	DeterminarTipoDuration metric.Float64Histogram

	// inventario.iniciar.cargar_items.duration: histograma de duración de carga de items en ms.
	CargarItemsDuration metric.Float64Histogram
}

// NewMetrics inicializa los instrumentos OTel del dominio iniciar.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(otelScope)

	crearDuration, err := meter.Float64Histogram(
		"inventario.iniciar.crear.duration",
		metric.WithDescription("Duración de la creación de inventario en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	crearTotal, err := meter.Int64Counter(
		"inventario.iniciar.crear.total",
		metric.WithDescription("Conteo de creaciones de inventario por resultado"),
	)
	if err != nil {
		return nil, err
	}

	itemsCount, err := meter.Int64Gauge(
		"inventario.iniciar.crear.items_count",
		metric.WithDescription("Cantidad de items en el conteo de inventario"),
	)
	if err != nil {
		return nil, err
	}

	determinarTipoDuration, err := meter.Float64Histogram(
		"inventario.iniciar.determinar_tipo.duration",
		metric.WithDescription("Duración de la determinación del tipo de inventario en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	cargarItemsDuration, err := meter.Float64Histogram(
		"inventario.iniciar.cargar_items.duration",
		metric.WithDescription("Duración de la carga de items en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		CrearDuration:          crearDuration,
		CrearTotal:             crearTotal,
		ItemsCount:             itemsCount,
		DeterminarTipoDuration: determinarTipoDuration,
		CargarItemsDuration:    cargarItemsDuration,
	}, nil
}

// noopMetrics devuelve métricas no-op cuando OTel no está configurado.
func noopMetrics() *Metrics {
	meter := otel.GetMeterProvider().Meter(otelScope)

	crearDuration, _ := meter.Float64Histogram("inventario.iniciar.crear.duration")
	crearTotal, _ := meter.Int64Counter("inventario.iniciar.crear.total")
	itemsCount, _ := meter.Int64Gauge("inventario.iniciar.crear.items_count")
	determinarTipoDuration, _ := meter.Float64Histogram("inventario.iniciar.determinar_tipo.duration")
	cargarItemsDuration, _ := meter.Float64Histogram("inventario.iniciar.cargar_items.duration")

	return &Metrics{
		CrearDuration:          crearDuration,
		CrearTotal:             crearTotal,
		ItemsCount:             itemsCount,
		DeterminarTipoDuration: determinarTipoDuration,
		CargarItemsDuration:    cargarItemsDuration,
	}
}

// RecordCrearInventario registra duración, resultado e items_count de una creación.
func (m *Metrics) RecordCrearInventario(ctx context.Context, duracionMs float64, resultado string, tiendaID int64, itemsCount int64, tipoDeterminado string) {
	tiendaIDAttr := metric.WithAttributes(
		attribute.Int64(AttrTiendaID, tiendaID),
	)
	m.CrearDuration.Record(ctx, duracionMs, metricAttrs(AttrInventarioResult, resultado), tiendaIDAttr)
	m.CrearTotal.Add(ctx, 1, metricAttrs(AttrInventarioResult, resultado), tiendaIDAttr)
	m.ItemsCount.Record(ctx, itemsCount, tiendaIDAttr)
}

// RecordDeterminarTipo registra duración de determinación de tipo.
func (m *Metrics) RecordDeterminarTipo(ctx context.Context, duracionMs float64) {
	m.DeterminarTipoDuration.Record(ctx, duracionMs)
}

// RecordCargarItems registra duración de carga de items.
func (m *Metrics) RecordCargarItems(ctx context.Context, duracionMs float64) {
	m.CargarItemsDuration.Record(ctx, duracionMs)
}
