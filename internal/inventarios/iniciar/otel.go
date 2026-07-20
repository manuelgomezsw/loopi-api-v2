package iniciar

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// metricAttrs construye metric.MeasurementOption con un atributo clave-valor.
func metricAttrs(key, value string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String(key, value))
}

// Nombres de atributos OTel según spec de observabilidad.
const (
	AttrInventarioResult = "resultado"
	AttrTiendaID         = "tienda_id"
	AttrTipoDeterminado  = "tipo_determinado"

	ResultSuccess           = "success"
	ResultValidationError   = "validation_error"
	ResultNotFound          = "not_found"
	ResultConflict          = "conflict"
	ResultServerError       = "server_error"
)
