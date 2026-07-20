package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
)

const testSecret = "secreto-de-test-32-caracteres-ok!!"

// generarToken firma un JWT con los parámetros dados usando el secreto de test.
func generarToken(t *testing.T, jti string, rol string, exp time.Time, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"jti": jti,
		"sub": "1", // sub debe ser string (RFC 7519 §4.1.2)
		"rol": rol,
		"iat": time.Now().UTC().Unix(),
		"exp": exp.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("error firmando token: %v", err)
	}
	return signed
}

// handlerSentinel es un handler que siempre responde 200 y sirve para verificar
// que el middleware dejó pasar la request.
var handlerSentinel = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// aplicarMiddleware envuelve el handlerSentinel con JWTMiddleware y ejecuta la request.
func aplicarMiddleware(t *testing.T, req *http.Request, repo auth.Repository) *httptest.ResponseRecorder {
	t.Helper()
	handler := auth.JWTMiddleware(testSecret, repo)(handlerSentinel)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// --- Tests ---

func TestMiddleware_SinCookie_Devuelve401(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := aplicarMiddleware(t, req, &mockRepo{
		existeFunc: func(_ string) (bool, error) { return false, nil },
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestMiddleware_CookieVacia_Devuelve401(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: ""})
	rec := aplicarMiddleware(t, req, &mockRepo{
		existeFunc: func(_ string) (bool, error) { return false, nil },
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestMiddleware_FirmaInvalida_Devuelve401(t *testing.T) {
	tokenFirmadoConOtroSecreto := generarToken(t, "jti-1", "admin",
		time.Now().UTC().Add(1*time.Hour), "otro-secreto-diferente")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: tokenFirmadoConOtroSecreto})
	rec := aplicarMiddleware(t, req, &mockRepo{
		existeFunc: func(_ string) (bool, error) { return false, nil },
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestMiddleware_TokenExpirado_Devuelve401(t *testing.T) {
	tokenExpirado := generarToken(t, "jti-exp", "admin",
		time.Now().UTC().Add(-1*time.Hour), testSecret) // expiró hace 1 hora

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: tokenExpirado})
	rec := aplicarMiddleware(t, req, &mockRepo{
		existeFunc: func(_ string) (bool, error) { return false, nil },
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestMiddleware_AlgoritmoIncorrecto_Devuelve401(t *testing.T) {
	// Token firmado con RS256 (algoritmo no aceptado por el middleware).
	// Simulamos modificando el header.alg manualmente vía un token none-alg.
	// En jwt/v5 ya no hay "none" alg sin opción explícita; usamos un token inválido.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.e30."})
	rec := aplicarMiddleware(t, req, &mockRepo{
		existeFunc: func(_ string) (bool, error) { return false, nil },
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestMiddleware_TokenRevocado_Devuelve401(t *testing.T) {
	signed := generarToken(t, "jti-revocado", "admin",
		time.Now().UTC().Add(1*time.Hour), testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: signed})
	rec := aplicarMiddleware(t, req, &mockRepo{
		existeFunc: func(_ string) (bool, error) { return true, nil }, // token en blacklist
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestMiddleware_BDNoDisponible_Devuelve503(t *testing.T) {
	signed := generarToken(t, "jti-bd-error", "admin",
		time.Now().UTC().Add(1*time.Hour), testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: signed})

	// Simular BD no disponible con cualquier error no-nil.
	rec := aplicarMiddleware(t, req, &mockRepo{
		existeFunc: func(_ string) (bool, error) { return false, http.ErrNoCookie },
	})
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("esperado 503, obtenido %d (fail-closed cuando BD no disponible)", rec.Code)
	}
}

func TestMiddleware_TokenValido_InyectaClaimsYPasaAlHandler(t *testing.T) {
	jtiEsperado := "jti-valido"
	signed := generarToken(t, jtiEsperado, "lider_tienda",
		time.Now().UTC().Add(1*time.Hour), testSecret)

	var claimsCapturados *auth.Claims
	handlerCapturador := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claimsCapturados = auth.ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: signed})
	rec := httptest.NewRecorder()

	handler := auth.JWTMiddleware(testSecret, &mockRepo{
		existeFunc: func(_ string) (bool, error) { return false, nil },
	})(handlerCapturador)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
	if claimsCapturados == nil {
		t.Fatal("claims no inyectados en el contexto")
	}
	if claimsCapturados.JTI != jtiEsperado {
		t.Errorf("JTI esperado %q, obtenido %q", jtiEsperado, claimsCapturados.JTI)
	}
	if claimsCapturados.Rol != "lider_tienda" {
		t.Errorf("rol esperado 'lider_tienda', obtenido %q", claimsCapturados.Rol)
	}
}
