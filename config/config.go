package config

import (
	"os"
	"strconv"
)

// Config contiene la configuración de la aplicación cargada desde variables de entorno.
type Config struct {
	// JWTSecret es la clave secreta para firmar y verificar tokens JWT (HS256).
	// Cargado desde JWT_SECRET. Nunca debe estar hardcodeado.
	JWTSecret string

	// JWTExpiryHours es el tiempo de vida del token en horas.
	// Cargado desde JWT_EXPIRY_HOURS. Default: 24.
	JWTExpiryHours int
}

// Load carga la configuración desde variables de entorno.
// Si JWT_SECRET está vacío, devuelve un error para forzar configuración explícita.
func Load() (*Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errMissingJWTSecret
	}

	expiryHours := 24
	if raw := os.Getenv("JWT_EXPIRY_HOURS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return nil, errInvalidJWTExpiryHours
		}
		expiryHours = parsed
	}

	return &Config{
		JWTSecret:      secret,
		JWTExpiryHours: expiryHours,
	}, nil
}
