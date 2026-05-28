package auth_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/manuelgomezsw/loopi-api-v2/config"
	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

// --- helpers de test ---

func newTestConfig() *config.Config {
	return &config.Config{JWTSecret: "secreto-de-test-32-caracteres-ok!!", JWTExpiryHours: 1}
}

// repoSpy captura llamadas al repositorio para verificar comportamiento del service.
type repoSpy struct {
	mockRepo
	bloquearLlamado        bool
	incrementarLlamado     bool
	resetearLlamado        bool
	bloquearHastaCap       time.Time
	incrementarContadorCap int
}

func (r *repoSpy) BloquearUsuario(usuarioID int, hasta time.Time) error {
	r.bloquearLlamado = true
	r.bloquearHastaCap = hasta
	return nil
}

func (r *repoSpy) IncrementarIntentosFallidos(usuarioID int, nuevoContador int) error {
	r.incrementarLlamado = true
	r.incrementarContadorCap = nuevoContador
	return nil
}

func (r *repoSpy) ResetearIntentosLogin(usuarioID int) error {
	r.resetearLlamado = true
	return nil
}

// hashMin genera un bcrypt con MinCost para agilizar los tests.
func hashMin(t *testing.T, pass string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("error generando hash: %v", err)
	}
	return string(h)
}

// claimsConExp construye un auth.Claims con ExpirationTime válido para RevocarToken.
func claimsConExp(jti string, exp time.Time) *auth.Claims {
	c := &auth.Claims{}
	c.JTI = jti
	c.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(exp)
	return c
}

// --- Tests: Authenticate ---

func TestAuthenticate_UsuarioNoExiste_RetornaErrCredenciales(t *testing.T) {
	repo := &mockRepo{
		buscarFunc: func(_ string) (*auth.UsuarioAuth, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := auth.NewService(newTestConfig(), repo)

	_, err := svc.Authenticate("inexistente", "cualquier")
	if !errors.Is(err, auth.ErrCredencialesInvalidas) {
		t.Errorf("esperaba ErrCredencialesInvalidas, obtuvo: %v", err)
	}
}

func TestAuthenticate_UsuarioInactivo_RetornaErrCredenciales(t *testing.T) {
	repo := &mockRepo{
		buscarFunc: func(_ string) (*auth.UsuarioAuth, error) {
			hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
			return &auth.UsuarioAuth{
				ID:             1,
				ContrasenaHash: string(hash),
				Activo:         false,
			}, nil
		},
	}
	svc := auth.NewService(newTestConfig(), repo)

	_, err := svc.Authenticate("usuario", "pass")
	if !errors.Is(err, auth.ErrCredencialesInvalidas) {
		t.Errorf("esperaba ErrCredencialesInvalidas, obtuvo: %v", err)
	}
}

func TestAuthenticate_CuentaBloqueada_RetornaErrBloqueada(t *testing.T) {
	hasta := time.Now().UTC().Add(5 * time.Minute)
	repo := &mockRepo{
		buscarFunc: func(_ string) (*auth.UsuarioAuth, error) {
			hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
			return &auth.UsuarioAuth{
				ID:             1,
				ContrasenaHash: string(hash),
				Activo:         true,
				BloqueadoHasta: &hasta,
			}, nil
		},
	}
	svc := auth.NewService(newTestConfig(), repo)

	_, err := svc.Authenticate("usuario", "pass")
	if !errors.Is(err, auth.ErrCuentaBloqueada) {
		t.Errorf("esperaba ErrCuentaBloqueada, obtuvo: %v", err)
	}
}

func TestAuthenticate_ContrasenaIncorrecta_IncrementaContador(t *testing.T) {
	spy := &repoSpy{}
	spy.buscarFunc = func(_ string) (*auth.UsuarioAuth, error) {
		return &auth.UsuarioAuth{
			ID:               1,
			ContrasenaHash:   hashMin(t, "correcta"),
			Activo:           true,
			IntentosFallidos: 1,
		}, nil
	}
	svc := auth.NewService(newTestConfig(), spy)

	_, err := svc.Authenticate("usuario", "incorrecta")
	if !errors.Is(err, auth.ErrCredencialesInvalidas) {
		t.Errorf("esperaba ErrCredencialesInvalidas, obtuvo: %v", err)
	}
	if !spy.incrementarLlamado {
		t.Error("esperaba que se llamara IncrementarIntentosFallidos")
	}
	if spy.incrementarContadorCap != 2 {
		t.Errorf("esperaba contador 2, obtuvo %d", spy.incrementarContadorCap)
	}
}

func TestAuthenticate_QuintoIntentoFallido_BloquearCuenta(t *testing.T) {
	spy := &repoSpy{}
	spy.buscarFunc = func(_ string) (*auth.UsuarioAuth, error) {
		return &auth.UsuarioAuth{
			ID:               1,
			ContrasenaHash:   hashMin(t, "correcta"),
			Activo:           true,
			IntentosFallidos: 4, // el 5° intento activa el bloqueo
		}, nil
	}
	svc := auth.NewService(newTestConfig(), spy)

	_, err := svc.Authenticate("usuario", "incorrecta")
	if !errors.Is(err, auth.ErrCredencialesInvalidas) {
		t.Errorf("esperaba ErrCredencialesInvalidas, obtuvo: %v", err)
	}
	if !spy.bloquearLlamado {
		t.Error("esperaba que se llamara BloquearUsuario al llegar al 5° intento")
	}
	if !spy.bloquearHastaCap.After(time.Now().UTC()) {
		t.Error("bloqueado_hasta debe ser en el futuro")
	}
}

func TestAuthenticate_LoginExitoso_RetornaToken(t *testing.T) {
	tiendaID := 7
	spy := &repoSpy{}
	spy.buscarFunc = func(_ string) (*auth.UsuarioAuth, error) {
		return &auth.UsuarioAuth{
			ID:               1,
			ContrasenaHash:   hashMin(t, "correcta"),
			Rol:              "lider_tienda",
			TiendaID:         &tiendaID,
			Activo:           true,
			IntentosFallidos: 0,
		}, nil
	}
	svc := auth.NewService(newTestConfig(), spy)

	result, err := svc.Authenticate("usuario", "correcta")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if result.Token == "" {
		t.Error("el token JWT no debe estar vacío")
	}
	if result.JTI == "" {
		t.Error("el JTI no debe estar vacío")
	}
	if result.Rol != "lider_tienda" {
		t.Errorf("rol esperado 'lider_tienda', obtenido %q", result.Rol)
	}
	if result.TiendaID == nil || *result.TiendaID != 7 {
		t.Error("tienda_id incorrecto en el resultado")
	}
	if result.ExpiresAt.Before(time.Now().UTC()) {
		t.Error("ExpiresAt debe ser en el futuro")
	}
	if !spy.resetearLlamado {
		t.Error("esperaba que se llamara ResetearIntentosLogin tras login exitoso")
	}
}

func TestAuthenticate_ErrorBD_PropagaError(t *testing.T) {
	errBD := errors.New("bd no disponible")
	repo := &mockRepo{
		buscarFunc: func(_ string) (*auth.UsuarioAuth, error) {
			return nil, errBD
		},
	}
	svc := auth.NewService(newTestConfig(), repo)

	_, err := svc.Authenticate("usuario", "pass")
	if err == nil {
		t.Fatal("esperaba error de BD")
	}
	if errors.Is(err, auth.ErrCredencialesInvalidas) {
		t.Error("no debe enmascarar error de BD como ErrCredencialesInvalidas")
	}
}

// --- Tests: RevocarToken ---

func TestRevocarToken_Exitoso_LlamaInsert(t *testing.T) {
	insertado := false
	repo := &mockRepo{
		insertFunc: func(_ string, _ time.Time) error {
			insertado = true
			return nil
		},
	}
	svc := auth.NewService(newTestConfig(), repo)

	claims := claimsConExp("jti-ok", time.Now().UTC().Add(1*time.Hour))
	err := svc.RevocarToken(claims)
	if err != nil {
		t.Errorf("RevocarToken no debería fallar: %v", err)
	}
	if !insertado {
		t.Error("esperaba que se llamara InsertTokenRevocado")
	}
}

func TestRevocarToken_SinExpiracion_RetornaError(t *testing.T) {
	repo := &mockRepo{
		insertFunc: func(_ string, _ time.Time) error { return nil },
	}
	svc := auth.NewService(newTestConfig(), repo)

	// Claims sin ExpiresAt — jwt.RegisteredClaims.ExpiresAt == nil
	claims := &auth.Claims{}
	claims.JTI = "jti-sin-exp"
	// ExpiresAt queda nil intencionalmente

	err := svc.RevocarToken(claims)
	if err == nil {
		t.Error("esperaba error cuando el token no tiene claim exp")
	}
}

func TestRevocarToken_ErrorRepo_PropagaError(t *testing.T) {
	errRepo := errors.New("fallo de BD")
	repo := &mockRepo{
		insertFunc: func(_ string, _ time.Time) error {
			return errRepo
		},
	}
	svc := auth.NewService(newTestConfig(), repo)

	claims := claimsConExp("jti-err", time.Now().UTC().Add(1*time.Hour))
	err := svc.RevocarToken(claims)
	if !errors.Is(err, errRepo) {
		t.Errorf("esperaba errRepo, obtuvo: %v", err)
	}
}
