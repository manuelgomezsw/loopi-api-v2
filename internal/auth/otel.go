package auth

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// metricAttrs construye metric.MeasurementOption con un atributo clave-valor.
func metricAttrs(key, value string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String(key, value))
}

// Nombres de atributos OTel según RF-AUTH-06.
const (
	AttrAuthResult = "auth.result"
	AttrUserRole   = "user.role"
	AttrHTTPRoute  = "http.route"

	ResultSuccess            = "success"
	ResultInvalidCredentials = "invalid_credentials"
	ResultAccountInactive    = "account_inactive"
	ResultAccountLocked      = "account_locked"
)
