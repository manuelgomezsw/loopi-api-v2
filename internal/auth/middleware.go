package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey es el tipo para las claves del contexto de request (evita colisiones).
type ContextKey string

const (
	// ContextKeyClaims es la clave bajo la que se almacenan los claims JWT validados.
	ContextKeyClaims ContextKey = "jwt_claims"
)

// Claims contiene los claims del JWT de loopi-api.
type Claims struct {
	jwt.RegisteredClaims
	JTI                      string `json:"jti"`
	Rol                      string `json:"rol"`
	TiendaID                 *int   `json:"tienda_id"`
	RequiereCambioContrasena bool   `json:"requiere_cambio_contrasena"`
}

// JWTMiddleware valida el token JWT en la cookie httpOnly en 3 pasos:
//  1. Verifica la firma HS256
//  2. Verifica que el claim `exp` no esté vencido
//  3. Verifica que el `jti` no esté en `tokens_revocados` (política fail-closed)
//
// Si cualquier paso falla → 401 (sin detalles de diagnóstico al cliente).
// Si Cloud SQL no está disponible en el paso 3 → 503 (fail-closed).
func JWTMiddleware(secret string, repo Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extraer JWT de la cookie httpOnly.
			cookie, err := r.Cookie("jwt")
			if err != nil {
				http.Error(w, "no autorizado", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimSpace(cookie.Value)
			if tokenStr == "" {
				http.Error(w, "no autorizado", http.StatusUnauthorized)
				return
			}

			// Paso 1 + 2: verificar firma HS256 y exp.
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("método de firma inesperado: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			}, jwt.WithExpirationRequired())

			if err != nil || !token.Valid {
				http.Error(w, "no autorizado", http.StatusUnauthorized)
				return
			}

			// Paso 3: verificar blacklist en Cloud SQL (fail-closed).
			revocado, err := repo.ExisteTokenRevocado(claims.JTI)
			if err != nil {
				// Cloud SQL no disponible → 503 (fail-closed).
				http.Error(w, "servicio no disponible", http.StatusServiceUnavailable)
				return
			}
			if revocado {
				http.Error(w, "no autorizado", http.StatusUnauthorized)
				return
			}

			// Paso 4: bloquear si el empleado debe cambiar su contraseña (RF-EMP-04.6).
			// El único endpoint permitido es POST .../contrasena/cambiar.
			if claims.RequiereCambioContrasena && !esCambioContrasena(r) {
				http.Error(w, `{"error":"cambio_contrasena_requerido","mensaje":"Debes cambiar tu contraseña antes de continuar."}`, http.StatusForbidden)
				return
			}

			// Inyectar claims en el contexto para handlers posteriores (RBAC).
			ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext extrae los claims JWT del contexto del request.
// Devuelve nil si no hay claims (request no pasó por el middleware).
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(ContextKeyClaims).(*Claims)
	return claims
}

// esCambioContrasena reporta si el request es el endpoint de cambio de contraseña.
// Este endpoint es el único permitido cuando requiere_cambio_contrasena = true (RF-EMP-04.6).
func esCambioContrasena(r *http.Request) bool {
	return r.Method == http.MethodPost && len(r.URL.Path) > 0 &&
		hasPathSuffix(r.URL.Path, "/contrasena/cambiar")
}

func hasPathSuffix(path, suffix string) bool {
	if len(path) < len(suffix) {
		return false
	}
	return path[len(path)-len(suffix):] == suffix
}

// ContextWithClaims inyecta claims en un contexto.
// Usado por el middleware y también por tests para simular autenticación.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ContextKeyClaims, claims)
}
