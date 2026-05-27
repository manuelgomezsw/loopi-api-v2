package config

import "errors"

var (
	errMissingJWTSecret      = errors.New("config: JWT_SECRET es obligatorio y no puede estar vacío")
	errInvalidJWTExpiryHours = errors.New("config: JWT_EXPIRY_HOURS debe ser un entero positivo")
)
