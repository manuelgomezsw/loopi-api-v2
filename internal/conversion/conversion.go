package conversion

import (
	"errors"
	"math"
)

var (
	// ErrTipoIncompatible se retorna al intentar convertir entre tipos de medida distintos.
	ErrTipoIncompatible = errors.New("no se puede convertir entre tipos de medida distintos")
	// ErrFactorInvalido se retorna cuando algún factor de conversión es <= 0.
	ErrFactorInvalido = errors.New("el factor de conversión debe ser mayor que cero")
)

// Convertir convierte cantidad desde la unidad con factorDesde (tipo tipoDesde) hacia la
// unidad con factorHacia (tipo tipoHacia). Ambos tipos deben ser iguales.
// El resultado se redondea a 4 decimales para coincidir con DECIMAL(12,4) en BD.
func Convertir(cantidad, factorDesde float64, tipoDesde string, factorHacia float64, tipoHacia string) (float64, error) {
	if tipoDesde != tipoHacia {
		return 0, ErrTipoIncompatible
	}
	if factorDesde <= 0 || factorHacia <= 0 {
		return 0, ErrFactorInvalido
	}
	resultado := cantidad * (factorDesde / factorHacia)
	return math.Round(resultado*10000) / 10000, nil
}

// EsCompatible retorna true si ambos tipos de medida son iguales (conversión permitida).
func EsCompatible(tipo1, tipo2 string) bool {
	return tipo1 == tipo2
}
