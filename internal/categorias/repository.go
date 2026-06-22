package categorias

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrCategoriaNoEncontrada   = errors.New("categoria_no_encontrada")
	ErrSubcategoriaNoEncontrada = errors.New("subcategoria_no_encontrada")
	ErrNombreDuplicado         = errors.New("nombre_duplicado")
)

// Repository define todas las operaciones de persistencia del módulo categorias.
type Repository interface {
	// Categorías
	InsertarCategoria(nombre string, creadoPor uint64) (*Categoria, error)
	ObtenerCategoriaPorID(id uint64) (*Categoria, error)
	ListarConItems(soloActivas *bool) (*CatalogoResponse, error)
	ActualizarCategoria(id uint64, nombre string, actualizadoPor uint64) (*Categoria, error)
	ContarSubcategoriasActivas(categoriaID uint64) (int, error)
	InactivarCategoria(id, actualizadoPor uint64) error
	InactivarSubcategoriasDeCategoria(categoriaID, actualizadoPor uint64) (int, error)
	ReactivarCategoria(id, actualizadoPor uint64) (*Categoria, error)

	// Subcategorías
	InsertarSubcategoria(nombre string, categoriaID, creadoPor uint64) (*Subcategoria, error)
	ObtenerSubcategoriaPorID(id uint64) (*Subcategoria, error)
	ActualizarSubcategoria(id uint64, nombre string, actualizadoPor uint64) (*Subcategoria, error)
	InactivarSubcategoria(id, actualizadoPor uint64) error
	ReactivarSubcategoria(id, actualizadoPor uint64) (*Subcategoria, error)
	ContarItemsPorSubcategoria(subcategoriaID uint64) (int, error)
}

type mysqlRepository struct {
	db *sql.DB
}

// NewRepository crea un nuevo repositorio de categorías.
func NewRepository(db *sql.DB) Repository {
	return &mysqlRepository{db: db}
}

// --- Categorías ---

func (r *mysqlRepository) InsertarCategoria(nombre string, creadoPor uint64) (*Categoria, error) {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO categorias (nombre, activo, creado_por, creado_en, actualizado_por, actualizado_en)
		 VALUES (?, 1, ?, ?, ?, ?)`,
		nombre, creadoPor, now, creadoPor, now,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de categoría creada: %w", err)
	}
	return r.ObtenerCategoriaPorID(uint64(id))
}

func (r *mysqlRepository) ObtenerCategoriaPorID(id uint64) (*Categoria, error) {
	row := r.db.QueryRow(
		`SELECT id, nombre, activo, creado_por, creado_en, actualizado_por, actualizado_en
		 FROM categorias WHERE id = ?`, id,
	)
	return scanCategoria(row)
}

func (r *mysqlRepository) ListarConItems(soloActivas *bool) (*CatalogoResponse, error) {
	whereClause := ""
	var args []any
	if soloActivas != nil {
		activo := 0
		if *soloActivas {
			activo = 1
		}
		whereClause = "WHERE c.activo = ?"
		args = append(args, activo)
	}

	// Consulta principal: JOIN categorias + subcategorias + conteo de items
	query := fmt.Sprintf(`
		SELECT
			c.id, c.nombre, c.activo, c.creado_por, c.creado_en, c.actualizado_por, c.actualizado_en,
			s.id, s.nombre, s.categoria_id,
			s.activo, s.creado_por, s.creado_en,
			s.actualizado_por, s.actualizado_en,
			COALESCE(item_count.total, 0)
		FROM categorias c
		LEFT JOIN subcategorias s ON s.categoria_id = c.id
		LEFT JOIN (
			SELECT subcategoria_id, COUNT(*) AS total
			FROM items
			GROUP BY subcategoria_id
		) item_count ON item_count.subcategoria_id = s.id
		%s
		ORDER BY c.nombre ASC, s.nombre ASC`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		// tabla items aún no existe (007 no implementado)
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
			return r.listarSinItems(soloActivas)
		}
		return nil, err
	}
	defer rows.Close()

	return buildCatalogo(rows)
}

// listarSinItems se usa cuando la tabla items aún no existe.
func (r *mysqlRepository) listarSinItems(soloActivas *bool) (*CatalogoResponse, error) {
	whereClause := ""
	var args []any
	if soloActivas != nil {
		activo := 0
		if *soloActivas {
			activo = 1
		}
		whereClause = "WHERE c.activo = ?"
		args = append(args, activo)
	}

	query := fmt.Sprintf(`
		SELECT
			c.id, c.nombre, c.activo, c.creado_por, c.creado_en, c.actualizado_por, c.actualizado_en,
			s.id, s.nombre, s.categoria_id,
			s.activo, s.creado_por, s.creado_en,
			s.actualizado_por, s.actualizado_en,
			0
		FROM categorias c
		LEFT JOIN subcategorias s ON s.categoria_id = c.id
		%s
		ORDER BY c.nombre ASC, s.nombre ASC`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return buildCatalogo(rows)
}

func buildCatalogo(rows *sql.Rows) (*CatalogoResponse, error) {
	catMap := make(map[uint64]*CategoriaResponse)
	var catOrder []uint64

	for rows.Next() {
		var (
			c          Categoria
			activoInt  int
			sID        sql.NullInt64
			sNombre    sql.NullString
			sCatID     sql.NullInt64
			sActivoInt sql.NullInt64
			sCreadoPor sql.NullInt64
			sCreadoEn  sql.NullTime
			sActPor    sql.NullInt64
			sActEn     sql.NullTime
			totalItems int
		)
		if err := rows.Scan(
			&c.ID, &c.Nombre, &activoInt, &c.CreadoPor, &c.CreadoEn, &c.ActualizadoPor, &c.ActualizadoEn,
			&sID, &sNombre, &sCatID, &sActivoInt, &sCreadoPor, &sCreadoEn, &sActPor, &sActEn,
			&totalItems,
		); err != nil {
			return nil, err
		}
		c.Activo = activoInt == 1

		if _, ok := catMap[c.ID]; !ok {
			cr := &CategoriaResponse{Categoria: c, Subcategorias: []SubcategoriaResponse{}}
			catMap[c.ID] = cr
			catOrder = append(catOrder, c.ID)
		}

		if sID.Valid {
			sub := SubcategoriaResponse{
				Subcategoria: Subcategoria{
					ID:             uint64(sID.Int64),
					Nombre:         sNombre.String,
					CategoriaID:    uint64(sCatID.Int64),
					Activo:         sActivoInt.Int64 == 1,
					CreadoPor:      uint64(sCreadoPor.Int64),
					CreadoEn:       sCreadoEn.Time,
					ActualizadoPor: uint64(sActPor.Int64),
					ActualizadoEn:  sActEn.Time,
				},
				TotalItems: totalItems,
			}
			catMap[c.ID].Subcategorias = append(catMap[c.ID].Subcategorias, sub)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]CategoriaResponse, 0, len(catOrder))
	for _, id := range catOrder {
		result = append(result, *catMap[id])
	}
	return &CatalogoResponse{Categorias: result, Total: len(result)}, nil
}

func (r *mysqlRepository) ActualizarCategoria(id uint64, nombre string, actualizadoPor uint64) (*Categoria, error) {
	_, err := r.db.Exec(
		`UPDATE categorias SET nombre = ?, actualizado_por = ?, actualizado_en = ? WHERE id = ?`,
		nombre, actualizadoPor, time.Now(), id,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	return r.ObtenerCategoriaPorID(id)
}

func (r *mysqlRepository) ContarSubcategoriasActivas(categoriaID uint64) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM subcategorias WHERE categoria_id = ? AND activo = 1`, categoriaID,
	).Scan(&count)
	return count, err
}

func (r *mysqlRepository) InactivarCategoria(id, actualizadoPor uint64) error {
	_, err := r.db.Exec(
		`UPDATE categorias SET activo = 0, actualizado_por = ?, actualizado_en = ? WHERE id = ?`,
		actualizadoPor, time.Now(), id,
	)
	return err
}

func (r *mysqlRepository) InactivarSubcategoriasDeCategoria(categoriaID, actualizadoPor uint64) (int, error) {
	now := time.Now()
	result, err := r.db.Exec(
		`UPDATE subcategorias SET activo = 0, actualizado_por = ?, actualizado_en = ?
		 WHERE categoria_id = ? AND activo = 1`,
		actualizadoPor, now, categoriaID,
	)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	return int(n), err
}

func (r *mysqlRepository) ReactivarCategoria(id, actualizadoPor uint64) (*Categoria, error) {
	_, err := r.db.Exec(
		`UPDATE categorias SET activo = 1, actualizado_por = ?, actualizado_en = ? WHERE id = ?`,
		actualizadoPor, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}
	return r.ObtenerCategoriaPorID(id)
}

// --- Subcategorías ---

func (r *mysqlRepository) InsertarSubcategoria(nombre string, categoriaID, creadoPor uint64) (*Subcategoria, error) {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO subcategorias (nombre, categoria_id, activo, creado_por, creado_en, actualizado_por, actualizado_en)
		 VALUES (?, ?, 1, ?, ?, ?, ?)`,
		nombre, categoriaID, creadoPor, now, creadoPor, now,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de subcategoría creada: %w", err)
	}
	return r.ObtenerSubcategoriaPorID(uint64(id))
}

func (r *mysqlRepository) ObtenerSubcategoriaPorID(id uint64) (*Subcategoria, error) {
	row := r.db.QueryRow(
		`SELECT id, nombre, categoria_id, activo, creado_por, creado_en, actualizado_por, actualizado_en
		 FROM subcategorias WHERE id = ?`, id,
	)
	return scanSubcategoria(row)
}

func (r *mysqlRepository) ActualizarSubcategoria(id uint64, nombre string, actualizadoPor uint64) (*Subcategoria, error) {
	_, err := r.db.Exec(
		`UPDATE subcategorias SET nombre = ?, actualizado_por = ?, actualizado_en = ? WHERE id = ?`,
		nombre, actualizadoPor, time.Now(), id,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	return r.ObtenerSubcategoriaPorID(id)
}

func (r *mysqlRepository) InactivarSubcategoria(id, actualizadoPor uint64) error {
	_, err := r.db.Exec(
		`UPDATE subcategorias SET activo = 0, actualizado_por = ?, actualizado_en = ? WHERE id = ?`,
		actualizadoPor, time.Now(), id,
	)
	return err
}

func (r *mysqlRepository) ReactivarSubcategoria(id, actualizadoPor uint64) (*Subcategoria, error) {
	_, err := r.db.Exec(
		`UPDATE subcategorias SET activo = 1, actualizado_por = ?, actualizado_en = ? WHERE id = ?`,
		actualizadoPor, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}
	return r.ObtenerSubcategoriaPorID(id)
}

func (r *mysqlRepository) ContarItemsPorSubcategoria(subcategoriaID uint64) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM items WHERE subcategoria_id = ?`, subcategoriaID,
	).Scan(&count)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		// tabla items aún no existe (007 no implementado)
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// --- helpers ---

func scanCategoria(row interface{ Scan(...any) error }) (*Categoria, error) {
	var c Categoria
	var activoInt int
	err := row.Scan(&c.ID, &c.Nombre, &activoInt, &c.CreadoPor, &c.CreadoEn, &c.ActualizadoPor, &c.ActualizadoEn)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCategoriaNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	c.Activo = activoInt == 1
	return &c, nil
}

func scanSubcategoria(row interface{ Scan(...any) error }) (*Subcategoria, error) {
	var s Subcategoria
	var activoInt int
	err := row.Scan(&s.ID, &s.Nombre, &s.CategoriaID, &activoInt, &s.CreadoPor, &s.CreadoEn, &s.ActualizadoPor, &s.ActualizadoEn)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubcategoriaNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	s.Activo = activoInt == 1
	return &s, nil
}

func mapMySQLErr(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrNombreDuplicado
	}
	return err
}
