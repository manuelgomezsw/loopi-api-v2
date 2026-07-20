package config_test

import (
	"testing"

	"github.com/manuelgomezsw/loopi-api-v2/config"
)

func TestLoad_SecretoVacio_RetornaError(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_EXPIRY_HOURS", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("esperaba error cuando JWT_SECRET está vacío")
	}
}

func TestLoad_ConfigMinima_Default24Horas(t *testing.T) {
	t.Setenv("JWT_SECRET", "mi-secreto-seguro")
	t.Setenv("JWT_EXPIRY_HOURS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if cfg.JWTSecret != "mi-secreto-seguro" {
		t.Errorf("JWTSecret incorrecto: %q", cfg.JWTSecret)
	}
	if cfg.JWTExpiryHours != 24 {
		t.Errorf("JWTExpiryHours esperado 24, obtenido %d", cfg.JWTExpiryHours)
	}
}

func TestLoad_HorasPersonalizadas_RetornaValorCorrecto(t *testing.T) {
	t.Setenv("JWT_SECRET", "secreto-test")
	t.Setenv("JWT_EXPIRY_HOURS", "48")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if cfg.JWTExpiryHours != 48 {
		t.Errorf("JWTExpiryHours esperado 48, obtenido %d", cfg.JWTExpiryHours)
	}
}

func TestLoad_HorasNoNumericas_RetornaError(t *testing.T) {
	t.Setenv("JWT_SECRET", "secreto-test")
	t.Setenv("JWT_EXPIRY_HOURS", "no-es-numero")

	_, err := config.Load()
	if err == nil {
		t.Fatal("esperaba error con JWT_EXPIRY_HOURS no numérico")
	}
}

func TestLoad_HorasCero_RetornaError(t *testing.T) {
	t.Setenv("JWT_SECRET", "secreto-test")
	t.Setenv("JWT_EXPIRY_HOURS", "0")

	_, err := config.Load()
	if err == nil {
		t.Fatal("esperaba error con JWT_EXPIRY_HOURS = 0")
	}
}

func TestLoad_HorasNegativas_RetornaError(t *testing.T) {
	t.Setenv("JWT_SECRET", "secreto-test")
	t.Setenv("JWT_EXPIRY_HOURS", "-5")

	_, err := config.Load()
	if err == nil {
		t.Fatal("esperaba error con JWT_EXPIRY_HOURS negativo")
	}
}
