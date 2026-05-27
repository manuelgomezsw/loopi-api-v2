package auth

import (
	"database/sql"
	"time"
)

// Repository define las operaciones de persistencia para autenticación.
type Repository interface {
	// InsertTokenRevocado registra un JWT revocado al momento del logout.
	InsertTokenRevocado(jti string, expiraEn time.Time) error

	// ExisteTokenRevocado consulta si un jti ya fue revocado (paso 3 de validación JWT).
	ExisteTokenRevocado(jti string) (bool, error)

	// LimpiarTokensExpirados elimina registros de tokens_revocados cuyo expira_en < NOW().
	// Devuelve el número de filas eliminadas.
	LimpiarTokensExpirados() (int64, error)
}

// mysqlRepository implementa Repository usando Cloud SQL (MySQL).
type mysqlRepository struct {
	db *sql.DB
}

// NewRepository crea un nuevo repositorio de autenticación.
func NewRepository(db *sql.DB) Repository {
	return &mysqlRepository{db: db}
}

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

// ExisteTokenRevocado consulta si el jti existe en la tabla de tokens revocados.
// La consulta es directa a Cloud SQL — sin caché en memoria por diseño (ver plan.md EC-003).
func (r *mysqlRepository) ExisteTokenRevocado(jti string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(1) FROM tokens_revocados WHERE jti = ?`,
		jti,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// LimpiarTokensExpirados elimina tokens_revocados con expira_en < NOW().
// Devuelve el número de filas eliminadas. Invocado por Cloud Scheduler diariamente.
func (r *mysqlRepository) LimpiarTokensExpirados() (int64, error) {
	result, err := r.db.Exec(
		`DELETE FROM tokens_revocados WHERE expira_en < NOW()`,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
