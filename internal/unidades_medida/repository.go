package unidades_medida

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	// ErrUnidadNoEncontrada se retorna cuando la unidad no existe en la BD.
	ErrUnidadNoEncontrada = errors.New("unidad_no_encontrada")
	// ErrCodigoDuplicado se retorna cuando el código ya existe.
	ErrCodigoDuplicado = errors.New("codigo_duplicado")
)

// UMRepository define todas las operaciones de persistencia para unidades de medida.
type UMRepository interface {
	Crear(req *CrearUMRequest) (*UnidadMedida, error)
	ExisteConCodigo(codigo string) (bool, error)
	ObtenerPorID(id uint64) (*UnidadMedida, error)
	ObtenerPorIDConItems(id uint64) (*DetalleUMResponse, error)
	Listar(params *ListarUMParams) (*ListarUMResponse, error)
	Editar(id uint64, req *EditarUMRequest) (*UnidadMedida, error)
	Inactivar(id uint64) error
	ContarItemsConUnidadCanonica(id uint64) (int, error)
	ContarUnidadesActivasPorTipo(tipo string, excludeID uint64) (int, error)
}

type mysqlUMRepository struct {
	db *sql.DB
}

// NewRepository crea un nuevo repositorio de unidades de medida.
func NewRepository(db *sql.DB) UMRepository {
	return &mysqlUMRepository{db: db}
}

const selectCols = `id, codigo, nombre, tipo_medida, factor_conversion, unidad_base, activo, creado_en, actualizado_en`

func scanUM(row interface {
	Scan(...any) error
}) (*UnidadMedida, error) {
	var u UnidadMedida
	var unidadBase, activo int
	err := row.Scan(
		&u.ID, &u.Codigo, &u.Nombre, &u.TipoMedida, &u.FactorConversion,
		&unidadBase, &activo, &u.CreadoEn, &u.ActualizadoEn,
	)
	if err != nil {
		return nil, err
	}
	u.UnidadBase = unidadBase == 1
	u.Activo = activo == 1
	return &u, nil
}

// Crear inserta una unidad nueva y retorna la entidad persistida.
func (r *mysqlUMRepository) Crear(req *CrearUMRequest) (*UnidadMedida, error) {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO unidades_medida
		  (codigo, nombre, tipo_medida, factor_conversion, unidad_base, activo, creado_en, actualizado_en)
		 VALUES (?, ?, ?, ?, 0, 1, ?, ?)`,
		req.Codigo, req.Nombre, req.TipoMedida, req.FactorConversion, now, now,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de unidad creada: %w", err)
	}
	return r.ObtenerPorID(uint64(id))
}

// ExisteConCodigo comprueba si ya existe una unidad con el código dado.
func (r *mysqlUMRepository) ExisteConCodigo(codigo string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(1) FROM unidades_medida WHERE codigo = ?`, codigo,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ObtenerPorID recupera una unidad por su PK.
func (r *mysqlUMRepository) ObtenerPorID(id uint64) (*UnidadMedida, error) {
	row := r.db.QueryRow(
		`SELECT `+selectCols+` FROM unidades_medida WHERE id = ?`, id,
	)
	u, err := scanUM(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnidadNoEncontrada
	}
	return u, err
}

// ObtenerPorIDConItems recupera una unidad + conteo de items que la usan como canónica.
func (r *mysqlUMRepository) ObtenerPorIDConItems(id uint64) (*DetalleUMResponse, error) {
	u, err := r.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	count, err := r.ContarItemsConUnidadCanonica(id)
	if err != nil {
		return nil, err
	}
	return &DetalleUMResponse{UnidadMedida: *u, ItemsConUnidadCanonica: count}, nil
}

// Listar retorna unidades paginadas con filtros opcionales de tipo y estado.
func (r *mysqlUMRepository) Listar(params *ListarUMParams) (*ListarUMResponse, error) {
	var where []string
	var args []any

	if params.Tipo != "" {
		where = append(where, "tipo_medida = ?")
		args = append(args, params.Tipo)
	}
	if params.Activo != nil {
		activo := 0
		if *params.Activo {
			activo = 1
		}
		where = append(where, "activo = ?")
		args = append(args, activo)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.db.QueryRow(
		fmt.Sprintf("SELECT COUNT(1) FROM unidades_medida %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, err
	}

	offset := (params.Page - 1) * params.Limit
	queryArgs := append(args, params.Limit, offset)
	rows, err := r.db.Query(
		fmt.Sprintf(
			`SELECT `+selectCols+` FROM unidades_medida %s ORDER BY tipo_medida ASC, nombre ASC LIMIT ? OFFSET ?`,
			whereClause,
		),
		queryArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var unidades []UnidadMedida
	for rows.Next() {
		u, err := scanUM(rows)
		if err != nil {
			return nil, err
		}
		unidades = append(unidades, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if unidades == nil {
		unidades = []UnidadMedida{}
	}
	return &ListarUMResponse{
		UnidadesMedida: unidades,
		Total:          total,
		Page:           params.Page,
		Limit:          params.Limit,
	}, nil
}

// Editar actualiza nombre y/o factor_conversion de una unidad existente.
func (r *mysqlUMRepository) Editar(id uint64, req *EditarUMRequest) (*UnidadMedida, error) {
	now := time.Now()
	var sets []string
	var args []any

	if req.Nombre != nil {
		sets = append(sets, "nombre = ?")
		args = append(args, *req.Nombre)
	}
	if req.FactorConversion != nil {
		sets = append(sets, "factor_conversion = ?")
		args = append(args, *req.FactorConversion)
	}
	sets = append(sets, "actualizado_en = ?")
	args = append(args, now)
	args = append(args, id)

	_, err := r.db.Exec(
		fmt.Sprintf("UPDATE unidades_medida SET %s WHERE id = ?", strings.Join(sets, ", ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	return r.ObtenerPorID(id)
}

// Inactivar pone activo=0 en la unidad indicada.
func (r *mysqlUMRepository) Inactivar(id uint64) error {
	_, err := r.db.Exec(
		`UPDATE unidades_medida SET activo = 0, actualizado_en = ? WHERE id = ?`,
		time.Now(), id,
	)
	return err
}

// ContarItemsConUnidadCanonica retorna cuántos items activos usan esta unidad.
// Retorna 0 graciosamente si la tabla items aún no existe (007 no implementado).
func (r *mysqlUMRepository) ContarItemsConUnidadCanonica(id uint64) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(1) FROM items WHERE unidad_id = ? AND activo = 1`, id,
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

// ContarUnidadesActivasPorTipo cuenta unidades activas del mismo tipo, excluyendo excludeID.
func (r *mysqlUMRepository) ContarUnidadesActivasPorTipo(tipo string, excludeID uint64) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(1) FROM unidades_medida WHERE tipo_medida = ? AND activo = 1 AND id != ?`,
		tipo, excludeID,
	).Scan(&count)
	return count, err
}

func mapMySQLErr(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrCodigoDuplicado
	}
	return err
}
