package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
)

// --- Mocks ---

// mockService implementa auth.Service para tests.
type mockService struct {
	authenticateFunc   func(usuario, contrasena string) (*auth.AuthResult, error)
	revocarTokenFunc   func(claims *auth.Claims) error
}

func (m *mockService) Authenticate(usuario, contrasena string) (*auth.AuthResult, error) {
	return m.authenticateFunc(usuario, contrasena)
}

func (m *mockService) RevocarToken(claims *auth.Claims) error {
	return m.revocarTokenFunc(claims)
}

// mockRepo implementa auth.Repository para tests.
type mockRepo struct {
	insertFunc                  func(jti string, expiraEn time.Time) error
	existeFunc                  func(jti string) (bool, error)
	limpiarFunc                 func() (int64, error)
	buscarFunc                  func(nombre string) (*auth.UsuarioAuth, error)
	incrementarIntentosFallidos func(usuarioID int, nuevoContador int) error
	bloquearFunc                func(usuarioID int, hasta time.Time) error
	resetearFunc                func(usuarioID int) error
}

func (m *mockRepo) InsertTokenRevocado(jti string, expiraEn time.Time) error {
	return m.insertFunc(jti, expiraEn)
}

func (m *mockRepo) ExisteTokenRevocado(jti string) (bool, error) {
	return m.existeFunc(jti)
}

func (m *mockRepo) LimpiarTokensExpirados() (int64, error) {
	return m.limpiarFunc()
}

func (m *mockRepo) BuscarUsuarioPorNombre(nombre string) (*auth.UsuarioAuth, error) {
	if m.buscarFunc != nil {
		return m.buscarFunc(nombre)
	}
	return nil, nil
}

func (m *mockRepo) IncrementarIntentosFallidos(usuarioID int, nuevoContador int) error {
	if m.incrementarIntentosFallidos != nil {
		return m.incrementarIntentosFallidos(usuarioID, nuevoContador)
	}
	return nil
}

func (m *mockRepo) BloquearUsuario(usuarioID int, hasta time.Time) error {
	if m.bloquearFunc != nil {
		return m.bloquearFunc(usuarioID, hasta)
	}
	return nil
}

func (m *mockRepo) ResetearIntentosLogin(usuarioID int) error {
	if m.resetearFunc != nil {
		return m.resetearFunc(usuarioID)
	}
	return nil
}

// --- Tests: Login ---

func TestLogin_CuerpoInvalido_Devuelve400(t *testing.T) {
	svc := &mockService{}
	repo := &mockRepo{}
	h := auth.NewHandler(svc, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestLogin_CredencialesInvalidas_Devuelve401(t *testing.T) {
	svc := &mockService{
		authenticateFunc: func(_, _ string) (*auth.AuthResult, error) {
			return nil, auth.ErrCredencialesInvalidas
		},
	}
	repo := &mockRepo{}
	h := auth.NewHandler(svc, repo)

	body := `{"usuario":"test","contrasena":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "credenciales_invalidas" {
		t.Errorf("error inesperado: %q", resp["error"])
	}
}

func TestLogin_CuentaBloqueada_Devuelve423(t *testing.T) {
	svc := &mockService{
		authenticateFunc: func(_, _ string) (*auth.AuthResult, error) {
			return nil, auth.ErrCuentaBloqueada
		},
	}
	repo := &mockRepo{}
	h := auth.NewHandler(svc, repo)

	body := `{"usuario":"bloqueado","contrasena":"any"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusLocked {
		t.Errorf("esperado 423, obtenido %d", rec.Code)
	}
}

func TestLogin_Exitoso_Devuelve200ConCookies(t *testing.T) {
	tiendaID := 3
	svc := &mockService{
		authenticateFunc: func(_, _ string) (*auth.AuthResult, error) {
			return &auth.AuthResult{
				Token:     "jwt.token.here",
				JTI:       "some-jti",
				ExpiresAt: time.Now().Add(24 * time.Hour),
				Rol:       "lider_tienda",
				TiendaID:  &tiendaID,
			}, nil
		},
	}
	repo := &mockRepo{}
	h := auth.NewHandler(svc, repo)

	body := `{"usuario":"user","contrasena":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}

	// Verificar cookies en la respuesta.
	cookies := rec.Result().Cookies()
	cookieMap := make(map[string]*http.Cookie)
	for _, c := range cookies {
		cookieMap[c.Name] = c
	}

	jwtCookie, ok := cookieMap["jwt"]
	if !ok {
		t.Fatal("cookie jwt no encontrada")
	}
	if !jwtCookie.HttpOnly {
		t.Error("cookie jwt debe ser httpOnly")
	}
	if !jwtCookie.Secure {
		t.Error("cookie jwt debe ser Secure")
	}

	xsrfCookie, ok := cookieMap["XSRF-TOKEN"]
	if !ok {
		t.Fatal("cookie XSRF-TOKEN no encontrada")
	}
	if xsrfCookie.HttpOnly {
		t.Error("cookie XSRF-TOKEN NO debe ser httpOnly (Angular necesita leerla)")
	}

	// Verificar body.
	var resp map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["rol"] != "lider_tienda" {
		t.Errorf("rol esperado 'lider_tienda', obtenido %q", resp["rol"])
	}
}

// --- Tests: Logout ---

func TestLogout_SinHeaderCSRF_Devuelve403(t *testing.T) {
	svc := &mockService{}
	repo := &mockRepo{}
	h := auth.NewHandler(svc, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	// Sin header X-XSRF-TOKEN.
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
}

func TestLogout_CSRFValido_Devuelve204(t *testing.T) {
	svc := &mockService{
		revocarTokenFunc: func(_ *auth.Claims) error {
			return nil
		},
	}
	repo := &mockRepo{}
	h := auth.NewHandler(svc, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("X-XSRF-TOKEN", "token-csrf-123")
	req.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: "token-csrf-123"})

	// Inyectar claims en el contexto (simula que el middleware JWT ya validó).
	ctx := auth.ContextWithClaims(req.Context(), &auth.Claims{})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("esperado 204, obtenido %d", rec.Code)
	}
}

// --- Tests: Me ---

func TestMe_SinClaims_Devuelve401(t *testing.T) {
	h := auth.NewHandler(&mockService{}, &mockRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestMe_ConClaims_Devuelve200(t *testing.T) {
	tiendaID := 5
	claims := &auth.Claims{Rol: "barista", TiendaID: &tiendaID}
	claims.Subject = "42"

	h := auth.NewHandler(&mockService{}, &mockRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	ctx := auth.ContextWithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}
