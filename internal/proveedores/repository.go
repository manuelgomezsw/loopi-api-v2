package proveedores

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	// ErrProveedorNoEncontrado se retorna cuando el proveedor no existe en la BD.
	ErrProveedorNoEncontrado = errors.New("proveedor_no_encontrado")
	// ErrNITDuplicado se retorna cuando el NIT ya existe.
	ErrNITDuplicado = errors.New("nit_duplicado")
)

// Repository define todas las operaciones de persistencia para proveedores.
type Repository interface {
	Crear(req *CrearProveedorRequest) (*Proveedor, error)
	ExisteNIT(nit string, excludeID *uint64) (bool, error)
	ObtenerPorID(id uint64) (*Proveedor, error)
	ObtenerPorIDConItems(id uint64) (*ProveedorDetalleResponse, error)
	Listar(filtros *FiltrosListado) (*ListarProveedoresResponse, error)
	Actualizar(id uint64, req *EditarProveedorRequest) (*Proveedor, error)
	CambiarEstado(id uint64, activo bool) error
	ContarItemsAsignados(id uint64) (int, error)
}

type mysqlRepository struct {
	db *sql.DB
}

// NewRepository crea un nuevo repositorio de proveedores.
func NewRepository(db *sql.DB) Repository {
	return &mysqlRepository{db: db}
}

const selectCols = `id, razon_social, nit, nombre_contacto, telefono_contacto, email_contacto, activo, creado_en, actualizado_en`

func scanProveedor(row interface {
	Scan(...any) error
}) (*Proveedor, error) {
	var p Proveedor
	var activo int
	err := row.Scan(
		&p.ID, &p.RazonSocial, &p.NIT, &p.NombreContacto, &p.TelefonoContacto,
		&p.EmailContacto, &activo, &p.CreadoEn, &p.ActualizadoEn,
	)
	if err != nil {
		return nil, err
	}
	p.Activo = activo == 1
	return &p, nil
}

// Crear inserta un proveedor nuevo (activo=1 por defecto) y retorna la entidad persistida.
func (r *mysqlRepository) Crear(req *CrearProveedorRequest) (*Proveedor, error) {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO proveedores
		  (razon_social, nit, nombre_contacto, telefono_contacto, email_contacto, activo, creado_en, actualizado_en)
		 VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
		req.RazonSocial, req.NIT, req.NombreContacto, req.TelefonoContacto, req.EmailContacto, now, now,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de proveedor creado: %w", err)
	}
	return r.ObtenerPorID(uint64(id))
}

// ExisteNIT comprueba si ya existe otro proveedor con el NIT dado.
// excludeID, si no es nil, excluye ese id de la búsqueda (uso en edición).
func (r *mysqlRepository) ExisteNIT(nit string, excludeID *uint64) (bool, error) {
	var count int
	var err error
	if excludeID != nil {
		err = r.db.QueryRow(
			`SELECT COUNT(1) FROM proveedores WHERE nit = ? AND id != ?`, nit, *excludeID,
		).Scan(&count)
	} else {
		err = r.db.QueryRow(
			`SELECT COUNT(1) FROM proveedores WHERE nit = ?`, nit,
		).Scan(&count)
	}
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ObtenerPorID recupera un proveedor por su PK.
func (r *mysqlRepository) ObtenerPorID(id uint64) (*Proveedor, error) {
	row := r.db.QueryRow(
		`SELECT `+selectCols+` FROM proveedores WHERE id = ?`, id,
	)
	p, err := scanProveedor(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProveedorNoEncontrado
	}
	return p, err
}

// ObtenerPorIDConItems recupera un proveedor + conteo de items que lo tienen asignado.
func (r *mysqlRepository) ObtenerPorIDConItems(id uint64) (*ProveedorDetalleResponse, error) {
	p, err := r.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	count, err := r.ContarItemsAsignados(id)
	if err != nil {
		return nil, err
	}
	return &ProveedorDetalleResponse{Proveedor: *p, ItemsAsignados: count}, nil
}

// Listar retorna proveedores paginados con filtros opcionales de estado y búsqueda.
func (r *mysqlRepository) Listar(filtros *FiltrosListado) (*ListarProveedoresResponse, error) {
	var where []string
	var args []any

	if filtros.Activo != nil {
		activo := 0
		if *filtros.Activo {
			activo = 1
		}
		where = append(where, "activo = ?")
		args = append(args, activo)
	}
	if filtros.Busqueda != "" {
		where = append(where, "(razon_social LIKE ? OR nit LIKE ?)")
		patron := "%" + filtros.Busqueda + "%"
		args = append(args, patron, patron)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.db.QueryRow(
		fmt.Sprintf("SELECT COUNT(1) FROM proveedores %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, err
	}

	offset := (filtros.Page - 1) * filtros.Limit
	queryArgs := append(args, filtros.Limit, offset)
	rows, err := r.db.Query(
		fmt.Sprintf(
			`SELECT `+selectCols+` FROM proveedores %s ORDER BY razon_social ASC LIMIT ? OFFSET ?`,
			whereClause,
		),
		queryArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proveedores []Proveedor
	for rows.Next() {
		p, err := scanProveedor(rows)
		if err != nil {
			return nil, err
		}
		proveedores = append(proveedores, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if proveedores == nil {
		proveedores = []Proveedor{}
	}
	return &ListarProveedoresResponse{
		Proveedores: proveedores,
		Total:       total,
		Page:        filtros.Page,
		Limit:       filtros.Limit,
	}, nil
}

// Actualizar actualiza solo los campos enviados (no-nil) de un proveedor existente.
func (r *mysqlRepository) Actualizar(id uint64, req *EditarProveedorRequest) (*Proveedor, error) {
	now := time.Now()
	var sets []string
	var args []any

	if req.RazonSocial != nil {
		sets = append(sets, "razon_social = ?")
		args = append(args, *req.RazonSocial)
	}
	if req.NIT != nil {
		sets = append(sets, "nit = ?")
		args = append(args, *req.NIT)
	}
	if req.NombreContacto != nil {
		sets = append(sets, "nombre_contacto = ?")
		args = append(args, *req.NombreContacto)
	}
	if req.TelefonoContacto != nil {
		sets = append(sets, "telefono_contacto = ?")
		args = append(args, *req.TelefonoContacto)
	}
	if req.EmailContacto != nil {
		sets = append(sets, "email_contacto = ?")
		args = append(args, *req.EmailContacto)
	}
	sets = append(sets, "actualizado_en = ?")
	args = append(args, now)
	args = append(args, id)

	_, err := r.db.Exec(
		fmt.Sprintf("UPDATE proveedores SET %s WHERE id = ?", strings.Join(sets, ", ")),
		args...,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	return r.ObtenerPorID(id)
}

// CambiarEstado actualiza el flag activo del proveedor indicado.
func (r *mysqlRepository) CambiarEstado(id uint64, activo bool) error {
	val := 0
	if activo {
		val = 1
	}
	_, err := r.db.Exec(
		`UPDATE proveedores SET activo = ?, actualizado_en = ? WHERE id = ?`,
		val, time.Now(), id,
	)
	return err
}

// ContarItemsAsignados retorna cuántos items activos tienen este proveedor asignado.
// Retorna 0 graciosamente si la tabla items aún no existe (007 no implementado).
func (r *mysqlRepository) ContarItemsAsignados(id uint64) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(1) FROM items WHERE proveedor_id = ? AND activo = 1`, id,
	).Scan(&count)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func mapMySQLErr(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrNITDuplicado
	}
	return err
}
