package conversion_test

import (
	"errors"
	"testing"

	"github.com/manuelgomezsw/loopi-api-v2/internal/conversion"
)

func TestConvertirKgAGramos(t *testing.T) {
	resultado, err := conversion.Convertir(2.0, 1000.0, "peso", 1.0, "peso")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado != 2000.0 {
		t.Errorf("esperaba 2000.0000, obtuvo %v", resultado)
	}
}

func TestConvertirLitrosAMililitros(t *testing.T) {
	resultado, err := conversion.Convertir(1.5, 1000.0, "volumen", 1.0, "volumen")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado != 1500.0 {
		t.Errorf("esperaba 1500.0000, obtuvo %v", resultado)
	}
}

func TestConvertirDocenasAUnidades(t *testing.T) {
	resultado, err := conversion.Convertir(2.0, 12.0, "unidad", 1.0, "unidad")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado != 24.0 {
		t.Errorf("esperaba 24.0000, obtuvo %v", resultado)
	}
}

func TestConvertirTiposIncompatibles(t *testing.T) {
	_, err := conversion.Convertir(1.0, 1000.0, "peso", 1.0, "volumen")
	if !errors.Is(err, conversion.ErrTipoIncompatible) {
		t.Errorf("esperaba ErrTipoIncompatible, obtuvo: %v", err)
	}
}

func TestConvertirFactorCero(t *testing.T) {
	_, err := conversion.Convertir(1.0, 0.0, "peso", 1.0, "peso")
	if !errors.Is(err, conversion.ErrFactorInvalido) {
		t.Errorf("esperaba ErrFactorInvalido, obtuvo: %v", err)
	}
}

func TestConvertirFactorHaciaCero(t *testing.T) {
	_, err := conversion.Convertir(1.0, 1000.0, "peso", 0.0, "peso")
	if !errors.Is(err, conversion.ErrFactorInvalido) {
		t.Errorf("esperaba ErrFactorInvalido para factorHacia=0, obtuvo: %v", err)
	}
}

func TestConvertirMismaUnidad(t *testing.T) {
	resultado, err := conversion.Convertir(5.0, 1.0, "unidad", 1.0, "unidad")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado != 5.0 {
		t.Errorf("esperaba 5.0000, obtuvo %v", resultado)
	}
}

func TestEsCompatible(t *testing.T) {
	if !conversion.EsCompatible("peso", "peso") {
		t.Error("peso y peso deben ser compatibles")
	}
	if conversion.EsCompatible("peso", "volumen") {
		t.Error("peso y volumen NO deben ser compatibles")
	}
}
