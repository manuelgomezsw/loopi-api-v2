package tiendas

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ErrTiendaNoEncontrada se retorna cuando la tienda no existe en la BD.
var ErrTiendaNoEncontrada = errors.New("tienda no encontrada")

// ErrCodigoDuplicado se retorna cuando el código ya existe.
var ErrCodigoDuplicado = errors.New("codigo_duplicado")

// ErrNombreDuplicado se retorna cuando el nombre ya existe (case-insensitive).
var ErrNombreDuplicado = errors.New("nombre_duplicado")

// TiendaRepository define todas las operaciones de persistencia para tiendas.
type TiendaRepository interface {
	Crear(t Tienda) (Tienda, error)
	ObtenerPorID(id uint64) (Tienda, error)
	Listar(estado string, pagina, limite int) ([]Tienda, int, error)
	Actualizar(id uint64, req TiendaUpdateRequest, adminID uint64, ahora time.Time) (Tienda, error)
	CambiarActivo(id uint64, activo bool, adminID uint64, ahora time.Time) (Tienda, error)
}

// mysqlTiendaRepository implementa TiendaRepository sobre Cloud SQL (MySQL).
type mysqlTiendaRepository struct {
	db *sql.DB
}

// NewRepository crea un nuevo repositorio de tiendas.
func NewRepository(db *sql.DB) TiendaRepository {
	return &mysqlTiendaRepository{db: db}
}

// Crear inserta una tienda nueva y retorna la entidad persistida con su ID.
func (r *mysqlTiendaRepository) Crear(t Tienda) (Tienda, error) {
	result, err := r.db.Exec(
		`INSERT INTO tiendas
		  (codigo, nombre, direccion, ciudad, telefono, activo,
		   creado_por, creado_en, actualizado_por, actualizado_en)
		 VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?)`,
		t.Codigo, t.Nombre, t.Direccion, t.Ciudad, t.Telefono,
		t.CreadoPor, t.CreadoEn, t.ActualizadoPor, t.ActualizadoEn,
	)
	if err != nil {
		return Tienda{}, mapMySQLError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Tienda{}, fmt.Errorf("obtener id de tienda creada: %w", err)
	}
	return r.ObtenerPorID(uint64(id))
}

// ObtenerPorID recupera una tienda por su PK.
func (r *mysqlTiendaRepository) ObtenerPorID(id uint64) (Tienda, error) {
	var t Tienda
	var activo int
	err := r.db.QueryRow(
		`SELECT id, codigo, nombre, direccion, ciudad, telefono, activo,
		        creado_por, creado_en, actualizado_por, actualizado_en
		 FROM tiendas WHERE id = ?`, id,
	).Scan(
		&t.ID, &t.Codigo, &t.Nombre, &t.Direccion, &t.Ciudad, &t.Telefono, &activo,
		&t.CreadoPor, &t.CreadoEn, &t.ActualizadoPor, &t.ActualizadoEn,
	)
	if err == sql.ErrNoRows {
		return Tienda{}, ErrTiendaNoEncontrada
	}
	if err != nil {
		return Tienda{}, err
	}
	t.Activo = activo == 1
	return t, nil
}

// Listar retorna tiendas paginadas ordenadas por nombre ASC.
// estado: "todas" | "activas" | "inactivas"
func (r *mysqlTiendaRepository) Listar(estado string, pagina, limite int) ([]Tienda, int, error) {
	offset := (pagina - 1) * limite

	whereClause := ""
	var args []interface{}
	if estado == "activas" {
		whereClause = "WHERE activo = 1"
	} else if estado == "inactivas" {
		whereClause = "WHERE activo = 0"
	}

	var total int
	err := r.db.QueryRow(
		fmt.Sprintf("SELECT COUNT(1) FROM tiendas %s", whereClause),
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		fmt.Sprintf(
			`SELECT id, codigo, nombre, direccion, ciudad, telefono, activo,
			        creado_por, creado_en, actualizado_por, actualizado_en
			 FROM tiendas %s ORDER BY nombre ASC LIMIT ? OFFSET ?`,
			whereClause,
		),
		append(args, limite, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tiendas []Tienda
	for rows.Next() {
		var t Tienda
		var activo int
		if err := rows.Scan(
			&t.ID, &t.Codigo, &t.Nombre, &t.Direccion, &t.Ciudad, &t.Telefono, &activo,
			&t.CreadoPor, &t.CreadoEn, &t.ActualizadoPor, &t.ActualizadoEn,
		); err != nil {
			return nil, 0, err
		}
		t.Activo = activo == 1
		tiendas = append(tiendas, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if tiendas == nil {
		tiendas = []Tienda{}
	}
	return tiendas, total, nil
}

// Actualizar modifica nombre, dirección, ciudad y teléfono. Nunca actualiza codigo.
func (r *mysqlTiendaRepository) Actualizar(id uint64, req TiendaUpdateRequest, adminID uint64, ahora time.Time) (Tienda, error) {
	_, err := r.db.Exec(
		`UPDATE tiendas
		 SET nombre = ?, direccion = ?, ciudad = ?, telefono = ?,
		     actualizado_por = ?, actualizado_en = ?
		 WHERE id = ?`,
		req.Nombre, req.Direccion, req.Ciudad, req.Telefono,
		adminID, ahora, id,
	)
	if err != nil {
		return Tienda{}, mapMySQLError(err)
	}
	return r.ObtenerPorID(id)
}

// CambiarActivo actualiza el campo activo y los campos de auditoría.
func (r *mysqlTiendaRepository) CambiarActivo(id uint64, activo bool, adminID uint64, ahora time.Time) (Tienda, error) {
	activoInt := 0
	if activo {
		activoInt = 1
	}
	_, err := r.db.Exec(
		`UPDATE tiendas
		 SET activo = ?, actualizado_por = ?, actualizado_en = ?
		 WHERE id = ?`,
		activoInt, adminID, ahora, id,
	)
	if err != nil {
		return Tienda{}, err
	}
	return r.ObtenerPorID(id)
}

// mapMySQLError convierte errores MySQL 1062 en errores de negocio tipados.
func mapMySQLError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		if containsConstraint(mysqlErr.Message, "uq_tiendas_nombre") {
			return ErrNombreDuplicado
		}
		return ErrCodigoDuplicado
	}
	return err
}

func containsConstraint(msg, constraint string) bool {
	for i := 0; i <= len(msg)-len(constraint); i++ {
		if msg[i:i+len(constraint)] == constraint {
			return true
		}
	}
	return false
}
