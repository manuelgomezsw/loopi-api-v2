package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/manuelgomezsw/loopi-api-v2/config"
)

// dummyHash se usa para normalizar el tiempo de respuesta cuando el usuario no existe
// (evita timing attack que revelaría si un usuario existe o no).
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

// Errores exportados del dominio de autenticación.
var (
	// ErrCredencialesInvalidas se devuelve cuando el usuario/contraseña no coinciden,
	// el usuario no existe, o la cuenta está inactiva.
	ErrCredencialesInvalidas = errors.New("credenciales inválidas")

	// ErrCuentaBloqueada se devuelve cuando bloqueado_hasta > NOW().
	ErrCuentaBloqueada = errors.New("cuenta bloqueada")
)

// usuarioAuth agrupa los campos de `usuarios` relevantes para autenticación.
type usuarioAuth struct {
	ID               int
	ContrasenaHash   string
	Rol              string
	TiendaID         *int
	Activo           bool
	BloqueadoHasta   *time.Time
	IntentosFallidos int
}

// AuthResult contiene los datos del login exitoso que el handler necesita para
// construir las cookies y el body de respuesta.
type AuthResult struct {
	Token          string
	JTI            string
	ExpiresAt      time.Time
	Rol            string
	TiendaID       *int
}

// Service define las operaciones de negocio de autenticación.
type Service interface {
	// Authenticate valida credenciales y devuelve un AuthResult en caso de éxito.
	Authenticate(usuario, contrasena string) (*AuthResult, error)

	// RevocarToken inserta el jti en tokens_revocados y devuelve la duración del token
	// para que el handler pueda expirar las cookies.
	RevocarToken(claims *Claims) error
}

type service struct {
	cfg  *config.Config
	repo Repository
	db   dbQuerier
}

// dbQuerier abstrae las consultas a la tabla usuarios que necesita el servicio.
// Permite testear sin una BD real.
type dbQuerier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// NewServiceWithDB crea el servicio de autenticación con acceso a la BD.
func NewServiceWithDB(cfg *config.Config, repo Repository, db dbQuerier) Service {
	return &service{cfg: cfg, repo: repo, db: db}
}

// Authenticate valida usuario y contraseña contra la tabla `usuarios`.
//
// Flujo (RF-AUTH-01):
//  1. Buscar usuario por nombre.
//  2. Si no existe → bcrypt dummy + devolver ErrCredencialesInvalidas.
//  3. Si no activo → bcrypt dummy + devolver ErrCredencialesInvalidas.
//  4. Si bloqueado_hasta > NOW() → devolver ErrCuentaBloqueada.
//  5. Comparar contraseña con bcrypt.
//  6. Si incorrecto → incrementar intentos_fallidos; si llega a 5, bloquear 5 min.
//  7. Si correcto → emitir JWT + resetear intentos.
func (s *service) Authenticate(usuario, contrasena string) (*AuthResult, error) {
	u, err := s.buscarUsuario(usuario)
	if err != nil {
		// Usuario no existe: ejecutar bcrypt dummy para normalizar tiempo.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(contrasena))
		return nil, ErrCredencialesInvalidas
	}

	if !u.Activo {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(contrasena))
		return nil, ErrCredencialesInvalidas
	}

	now := time.Now().UTC()
	if u.BloqueadoHasta != nil && u.BloqueadoHasta.After(now) {
		return nil, ErrCuentaBloqueada
	}

	// Verificar contraseña.
	if err := bcrypt.CompareHashAndPassword([]byte(u.ContrasenaHash), []byte(contrasena)); err != nil {
		s.registrarFalloLogin(u)
		return nil, ErrCredencialesInvalidas
	}

	// Login exitoso: resetear contador.
	_, _ = s.db.Exec(
		`UPDATE usuarios SET intentos_fallidos = 0, bloqueado_hasta = NULL WHERE id = ?`,
		u.ID,
	)

	return s.emitirToken(u)
}

// RevocarToken inserta el jti del JWT activo en tokens_revocados.
func (s *service) RevocarToken(claims *Claims) error {
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return err
	}
	return s.repo.InsertTokenRevocado(claims.JTI, exp.Time)
}

// buscarUsuario consulta la tabla usuarios por nombre de usuario.
func (s *service) buscarUsuario(nombre string) (*usuarioAuth, error) {
	u := &usuarioAuth{}
	err := s.db.QueryRow(
		`SELECT id, contrasena_hash, rol, tienda_id, activo, bloqueado_hasta, intentos_fallidos
         FROM usuarios WHERE nombre = ?`,
		nombre,
	).Scan(&u.ID, &u.ContrasenaHash, &u.Rol, &u.TiendaID, &u.Activo, &u.BloqueadoHasta, &u.IntentosFallidos)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// registrarFalloLogin incrementa intentos_fallidos; bloquea si llega a 5.
func (s *service) registrarFalloLogin(u *usuarioAuth) {
	u.IntentosFallidos++
	if u.IntentosFallidos >= 5 {
		bloqueadoHasta := time.Now().UTC().Add(5 * time.Minute)
		_, _ = s.db.Exec(
			`UPDATE usuarios SET intentos_fallidos = 0, bloqueado_hasta = ? WHERE id = ?`,
			bloqueadoHasta, u.ID,
		)
	} else {
		_, _ = s.db.Exec(
			`UPDATE usuarios SET intentos_fallidos = ? WHERE id = ?`,
			u.IntentosFallidos, u.ID,
		)
	}
}

// emitirToken genera el JWT firmado con HS256 y los claims del usuario.
func (s *service) emitirToken(u *usuarioAuth) (*AuthResult, error) {
	jti := uuid.New().String()
	now := time.Now().UTC()
	exp := now.Add(time.Duration(s.cfg.JWTExpiryHours) * time.Hour)

	claims := jwt.MapClaims{
		"jti":       jti,
		"sub":       u.ID,
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
