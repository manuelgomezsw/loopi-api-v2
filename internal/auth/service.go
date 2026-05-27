package auth

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/manuelgomezsw/loopi-api-v2/config"
)

// dummyHash normaliza el tiempo de respuesta cuando el usuario no existe,
// evitando un timing attack que revelaría si un nombre de usuario es válido.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

// Errores exportados del dominio de autenticación.
var (
	ErrCredencialesInvalidas = errors.New("credenciales inválidas")
	ErrCuentaBloqueada       = errors.New("cuenta bloqueada")
)

// UsuarioAuth agrupa los campos de `usuarios` relevantes para autenticación.
// Exportado para que el repositorio y los tests puedan usarlo.
type UsuarioAuth struct {
	ID               int
	ContrasenaHash   string
	Rol              string
	TiendaID         *int
	Activo           bool
	BloqueadoHasta   *time.Time
	IntentosFallidos int
}

// AuthResult contiene los datos del login exitoso que el handler necesita.
type AuthResult struct {
	Token     string
	JTI       string
	ExpiresAt time.Time
	Rol       string
	TiendaID  *int
}

// Service define las operaciones de negocio de autenticación.
type Service interface {
	Authenticate(usuario, contrasena string) (*AuthResult, error)
	RevocarToken(claims *Claims) error
}

type service struct {
	cfg  *config.Config
	repo Repository // única dependencia — cero SQL directo aquí
}

// NewService crea el servicio de autenticación.
// El service solo recibe el repositorio; no sabe que existe sql.DB.
func NewService(cfg *config.Config, repo Repository) Service {
	return &service{cfg: cfg, repo: repo}
}

// Authenticate valida credenciales y emite un JWT en caso de éxito.
//
// Lógica de negocio (RF-AUTH-01) — sin ninguna sentencia SQL:
//  1. Buscar usuario → repositorio
//  2. Verificar activo
//  3. Verificar bloqueo temporal
//  4. Comparar contraseña con bcrypt
//  5. En fallo → incrementar contador → repositorio
//  6. En éxito → resetear contador → repositorio, emitir JWT
func (s *service) Authenticate(usuario, contrasena string) (*AuthResult, error) {
	u, err := s.repo.BuscarUsuarioPorNombre(usuario)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Normalizar tiempo: hacer bcrypt aunque el usuario no exista.
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(contrasena))
			return nil, ErrCredencialesInvalidas
		}
		return nil, err
	}

	if !u.Activo {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(contrasena))
		return nil, ErrCredencialesInvalidas
	}

	if u.BloqueadoHasta != nil && u.BloqueadoHasta.After(time.Now().UTC()) {
		return nil, ErrCuentaBloqueada
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.ContrasenaHash), []byte(contrasena)); err != nil {
		s.registrarFallo(u) // lógica de negocio → delega escritura al repo
		return nil, ErrCredencialesInvalidas
	}

	// Login exitoso: resetear contador (lógica de negocio → delega al repo).
	_ = s.repo.ResetearIntentosLogin(u.ID)

	return s.emitirToken(u)
}

// RevocarToken inserta el jti en tokens_revocados para invalidar el JWT inmediatamente.
//
// Guarda la expiración original para que el job de limpieza pueda purgar tokens
// revocados una vez que hayan expirado de forma natural.
func (s *service) RevocarToken(claims *Claims) error {
	if claims.RegisteredClaims.ExpiresAt == nil {
		return errors.New("revocación: el token no contiene claim exp")
	}
	return s.repo.InsertTokenRevocado(claims.JTI, claims.RegisteredClaims.ExpiresAt.Time)
}

// registrarFallo aplica la regla de negocio de bloqueo:
// incrementa el contador; si llega a 5 bloquea la cuenta 5 minutos.
// Toda escritura va al repositorio — aquí solo está la decisión.
func (s *service) registrarFallo(u *UsuarioAuth) {
	nuevoContador := u.IntentosFallidos + 1
	if nuevoContador >= 5 {
		hasta := time.Now().UTC().Add(5 * time.Minute)
		_ = s.repo.BloquearUsuario(u.ID, hasta)
	} else {
		_ = s.repo.IncrementarIntentosFallidos(u.ID, nuevoContador)
	}
}

// emitirToken genera el JWT firmado HS256 — lógica pura, sin acceso a BD.
func (s *service) emitirToken(u *UsuarioAuth) (*AuthResult, error) {
	jti := uuid.New().String()
	now := time.Now().UTC()
	exp := now.Add(time.Duration(s.cfg.JWTExpiryHours) * time.Hour)

	claims := jwt.MapClaims{
		"jti":       jti,
		"sub":       strconv.Itoa(u.ID), // sub debe ser string (RFC 7519 §4.1.2)
		"rol":       u.Rol,
		"tienda_id": u.TiendaID,
		"iat":       now.Unix(),
		"exp":       exp.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token:     signed,
		JTI:       jti,
		ExpiresAt: exp,
		Rol:       u.Rol,
		TiendaID:  u.TiendaID,
	}, nil
}
