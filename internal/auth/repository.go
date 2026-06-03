package auth

import (
	"database/sql"
	"time"
)

// Repository define TODAS las operaciones de persistencia para autenticación.
// Es la única capa que conoce SQL — el service no debe tener ninguna sentencia SQL.
type Repository interface {
	// --- tokens_revocados ---

	// InsertTokenRevocado registra un JWT revocado al momento del logout.
	InsertTokenRevocado(jti string, expiraEn time.Time) error

	// ExisteTokenRevocado consulta si un jti ya fue revocado (paso 3 de validación JWT).
	ExisteTokenRevocado(jti string) (bool, error)

	// LimpiarTokensExpirados elimina registros cuyo expira_en < NOW().
	LimpiarTokensExpirados() (int64, error)

	// --- empleados (autenticación desde 003-gestion-empleados) ---

	// BuscarUsuarioPorNombre recupera los campos de autenticación desde `empleados`.
	// Devuelve sql.ErrNoRows si no existe.
	BuscarUsuarioPorNombre(nombre string) (*UsuarioAuth, error)

	// IncrementarIntentosFallidos suma 1 al contador en `empleados`.
	IncrementarIntentosFallidos(usuarioID int, nuevoContador int) error

	// BloquearUsuario establece bloqueado_hasta y resetea intentos_fallidos en `empleados`.
	BloquearUsuario(usuarioID int, hasta time.Time) error

	// ResetearIntentosLogin limpia intentos_fallidos y bloqueado_hasta en `empleados`.
	ResetearIntentosLogin(usuarioID int) error
}

// mysqlRepository implementa Repository usando Cloud SQL (MySQL).
type mysqlRepository struct {
	db *sql.DB
}

// NewRepository crea un nuevo repositorio de autenticación.
func NewRepository(db *sql.DB) Repository {
	return &mysqlRepository{db: db}
}

// --- tokens_revocados ---

// InsertTokenRevocado inserta un registro de token revocado con timestamps de auditoría.
func (r *mysqlRepository) InsertTokenRevocado(jti string, expiraEn time.Time) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(
		`INSERT INTO tokens_revocados (jti, expira_en, creado_en, actualizado_en)
         VALUES (?, ?, ?, ?)`,
		jti, expiraEn, now, now,
	)
	return err
}

// ExisteTokenRevocado consulta si el jti existe en tokens_revocados.
// Directo a Cloud SQL — sin caché en memoria (ver plan.md EC-003).
func (r *mysqlRepository) ExisteTokenRevocado(jti string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(1) FROM tokens_revocados WHERE jti = ?`, jti,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// LimpiarTokensExpirados elimina tokens con expira_en < NOW().
func (r *mysqlRepository) LimpiarTokensExpirados() (int64, error) {
	result, err := r.db.Exec(
		`DELETE FROM tokens_revocados WHERE expira_en < NOW()`,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- empleados (autenticación desde migración 003) ---

// BuscarUsuarioPorNombre recupera los campos de autenticación desde `empleados`.
// Busca por el campo `usuario` (nombre único de login).
func (r *mysqlRepository) BuscarUsuarioPorNombre(nombre string) (*UsuarioAuth, error) {
	u := &UsuarioAuth{}
	var tiendaID sql.NullInt64
	var bloqueadoHasta sql.NullTime
	var activo, requiereCambio int
	err := r.db.QueryRow(
		`SELECT id, contrasena_hash, rol, tienda_id, activo, bloqueado_hasta,
		        intentos_fallidos, requiere_cambio_contrasena
		 FROM empleados WHERE usuario = ?`,
		nombre,
	).Scan(
		&u.ID, &u.ContrasenaHash, &u.Rol, &tiendaID,
		&activo, &bloqueadoHasta, &u.IntentosFallidos, &requiereCambio,
	)
	if err != nil {
		return nil, err
	}
	u.Activo = activo == 1
	u.RequiereCambioContrasena = requiereCambio == 1
	if tiendaID.Valid {
		v := int(tiendaID.Int64)
		u.TiendaID = &v
	}
	if bloqueadoHasta.Valid {
		u.BloqueadoHasta = &bloqueadoHasta.Time
	}
	return u, nil
}

// IncrementarIntentosFallidos actualiza el contador de intentos en `empleados`.
func (r *mysqlRepository) IncrementarIntentosFallidos(usuarioID int, nuevoContador int) error {
	_, err := r.db.Exec(
		`UPDATE empleados SET intentos_fallidos = ? WHERE id = ?`,
		nuevoContador, usuarioID,
	)
	return err
}

// BloquearUsuario establece bloqueado_hasta y resetea intentos_fallidos en `empleados`.
func (r *mysqlRepository) BloquearUsuario(usuarioID int, hasta time.Time) error {
	_, err := r.db.Exec(
		`UPDATE empleados SET intentos_fallidos = 0, bloqueado_hasta = ? WHERE id = ?`,
		hasta, usuarioID,
	)
	return err
}

// ResetearIntentosLogin limpia el contador y desbloquea al usuario en `empleados`.
func (r *mysqlRepository) ResetearIntentosLogin(usuarioID int) error {
	_, err := r.db.Exec(
		`UPDATE empleados SET intentos_fallidos = 0, bloqueado_hasta = NULL WHERE id = ?`,
		usuarioID,
	)
	return err
}
