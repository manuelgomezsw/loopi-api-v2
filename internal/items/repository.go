package items

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	// ErrItemNoEncontrado se retorna cuando el item no existe en la BD.
	ErrItemNoEncontrado = errors.New("item_no_encontrado")
	// ErrCodigoDuplicado se retorna cuando el código ya existe.
	ErrCodigoDuplicado = errors.New("codigo_duplicado")
	// ErrNombreDuplicado se retorna cuando el nombre ya existe (case-insensitive).
	ErrNombreDuplicado = errors.New("nombre_duplicado")
)

// Repository define todas las operaciones de persistencia para items y costos por tienda.
type Repository interface {
	Crear(req *CrearItemRequest, userID uint64) (*Item, error)
	ExisteCodigo(codigo string) (bool, error)
	ExisteNombre(nombre string, excludeID *uint64) (bool, error)
	ObtenerPorID(id uint64) (*Item, error)
	ObtenerDetallePorID(id uint64) (*ItemConNombres, error)
	Listar(filtros *FiltrosListado) (*ListarItemsResponse, error)
	Actualizar(id uint64, req *EditarItemRequest, userID uint64) (*Item, error)
	CambiarEstado(id uint64, activo bool, userID uint64) (*Item, error)
	EstaEnUso(itemID uint64) (bool, error)

	VerificarSubcategoria(id uint64) (existe, activo bool, err error)
	VerificarProveedor(id uint64) (existe, activo bool, err error)
	VerificarUnidadMedida(id uint64) (existe, activo bool, err error)
	VerificarTienda(id uint64) (existe, activo bool, err error)

	InsertarCostoTienda(itemID uint64, req *RegistrarCostoTiendaRequest, userID uint64) (*ItemCostoTienda, error)
	ListarCostosTienda(itemID uint64) ([]CostoPorTienda, error)
}

type mysqlRepository struct {
	db *sql.DB
}

// NewRepository crea un nuevo repositorio de items.
func NewRepository(db *sql.DB) Repository {
	return &mysqlRepository{db: db}
}

const detalleSelect = `
	SELECT i.id, i.codigo, i.nombre, i.tipo, i.subcategoria_id,
	       CONCAT(c.nombre, ' > ', s.nombre) AS subcategoria_nombre,
	       i.proveedor_id, p.razon_social,
	       i.unidad_medida_id, um.codigo AS unidad_medida_simbolo,
	       i.costo_unitario, i.frecuencia_inventario, i.stock_seguridad, i.tiempo_entrega_dias,
	       i.activo, i.creado_por, i.creado_en, i.actualizado_por, i.actualizado_en
	  FROM items i
	  JOIN subcategorias s ON i.subcategoria_id = s.id
	  JOIN categorias c ON s.categoria_id = c.id
	  JOIN unidades_medida um ON i.unidad_medida_id = um.id
	  LEFT JOIN proveedores p ON i.proveedor_id = p.id
`

const itemSelectCols = `id, codigo, nombre, tipo, subcategoria_id, proveedor_id, unidad_medida_id,
	costo_unitario, frecuencia_inventario, stock_seguridad, tiempo_entrega_dias,
	activo, creado_por, creado_en, actualizado_por, actualizado_en`

func scanItem(row interface{ Scan(...any) error }) (*Item, error) {
	var it Item
	var proveedorID, costoUnitario, tiempoEntrega sql.NullInt64
	var activo int
	err := row.Scan(
		&it.ID, &it.Codigo, &it.Nombre, &it.Tipo, &it.SubcategoriaID, &proveedorID, &it.UnidadMedidaID,
		&costoUnitario, &it.FrecuenciaInventario, &it.StockSeguridad, &tiempoEntrega,
		&activo, &it.CreadoPor, &it.CreadoEn, &it.ActualizadoPor, &it.ActualizadoEn,
	)
	if err != nil {
		return nil, err
	}
	it.Activo = activo == 1
	if proveedorID.Valid {
		v := uint64(proveedorID.Int64)
		it.ProveedorID = &v
	}
	if costoUnitario.Valid {
		v := int(costoUnitario.Int64)
		it.CostoUnitario = &v
	}
	if tiempoEntrega.Valid {
		v := uint16(tiempoEntrega.Int64)
		it.TiempoEntregaDias = &v
	}
	return &it, nil
}

func scanItemConNombres(row interface{ Scan(...any) error }) (*ItemConNombres, error) {
	var d ItemConNombres
	var proveedorID, costoUnitario, tiempoEntrega sql.NullInt64
	var proveedorNombre sql.NullString
	var activo int
	err := row.Scan(
		&d.ID, &d.Codigo, &d.Nombre, &d.Tipo, &d.SubcategoriaID, &d.SubcategoriaNombre,
		&proveedorID, &proveedorNombre,
		&d.UnidadMedidaID, &d.UnidadMedidaSimbolo,
		&costoUnitario, &d.FrecuenciaInventario, &d.StockSeguridad, &tiempoEntrega,
		&activo, &d.CreadoPor, &d.CreadoEn, &d.ActualizadoPor, &d.ActualizadoEn,
	)
	if err != nil {
		return nil, err
	}
	d.Activo = activo == 1
	if proveedorID.Valid {
		v := uint64(proveedorID.Int64)
		d.ProveedorID = &v
	}
	if proveedorNombre.Valid {
		d.ProveedorNombre = &proveedorNombre.String
	}
	if costoUnitario.Valid {
		v := int(costoUnitario.Int64)
		d.CostoUnitario = &v
	}
	if tiempoEntrega.Valid {
		v := uint16(tiempoEntrega.Int64)
		d.TiempoEntregaDias = &v
	}
	return &d, nil
}

// Crear inserta un item nuevo (activo=1 por defecto) y retorna la entidad persistida.
func (r *mysqlRepository) Crear(req *CrearItemRequest, userID uint64) (*Item, error) {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO items
		  (codigo, nombre, tipo, subcategoria_id, proveedor_id, unidad_medida_id,
		   costo_unitario, frecuencia_inventario, stock_seguridad, tiempo_entrega_dias,
		   activo, creado_por, creado_en, actualizado_por, actualizado_en)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?)`,
		req.Codigo, req.Nombre, req.Tipo, req.SubcategoriaID, req.ProveedorID, req.UnidadMedidaID,
		req.CostoUnitario, req.FrecuenciaInventario, req.StockSeguridad, req.TiempoEntregaDias,
		userID, now, userID, now,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de item creado: %w", err)
	}
	return r.ObtenerPorID(uint64(id))
}

// ExisteCodigo comprueba si ya existe un item con el código dado.
func (r *mysqlRepository) ExisteCodigo(codigo string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(1) FROM items WHERE codigo = ?`, codigo).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExisteNombre comprueba si ya existe otro item con el nombre dado (case-insensitive).
// excludeID, si no es nil, excluye ese id de la búsqueda (uso en edición).
func (r *mysqlRepository) ExisteNombre(nombre string, excludeID *uint64) (bool, error) {
	var count int
	var err error
	if excludeID != nil {
		err = r.db.QueryRow(
			`SELECT COUNT(1) FROM items WHERE nombre = ? AND id != ?`, nombre, *excludeID,
		).Scan(&count)
	} else {
		err = r.db.QueryRow(`SELECT COUNT(1) FROM items WHERE nombre = ?`, nombre).Scan(&count)
	}
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ObtenerPorID recupera un item por su PK, sin joins.
func (r *mysqlRepository) ObtenerPorID(id uint64) (*Item, error) {
	row := r.db.QueryRow(`SELECT `+itemSelectCols+` FROM items WHERE id = ?`, id)
	it, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrItemNoEncontrado
	}
	return it, err
}

// ObtenerDetallePorID recupera un item con nombres desnormalizados (subcategoría, proveedor, unidad).
// EstaEnUso NO se completa aquí — es responsabilidad del service.
func (r *mysqlRepository) ObtenerDetallePorID(id uint64) (*ItemConNombres, error) {
	row := r.db.QueryRow(detalleSelect+" WHERE i.id = ?", id)
	d, err := scanItemConNombres(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrItemNoEncontrado
	}
	return d, err
}

// Listar retorna items paginados con filtros opcionales de tipo, frecuencia y estado.
func (r *mysqlRepository) Listar(filtros *FiltrosListado) (*ListarItemsResponse, error) {
	var where []string
	var args []any

	if filtros.Tipo != "" {
		where = append(where, "i.tipo = ?")
		args = append(args, filtros.Tipo)
	}
	if filtros.Frecuencia != "" {
		where = append(where, "i.frecuencia_inventario = ?")
		args = append(args, filtros.Frecuencia)
	}
	if filtros.Activo != nil {
		activo := 0
		if *filtros.Activo {
			activo = 1
		}
		where = append(where, "i.activo = ?")
		args = append(args, activo)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.db.QueryRow(
		fmt.Sprintf(`SELECT COUNT(1) FROM items i %s`, whereClause), args...,
	).Scan(&total); err != nil {
		return nil, err
	}

	offset := (filtros.Pagina - 1) * filtros.PorPagina
	queryArgs := append(append([]any{}, args...), filtros.PorPagina, offset)
	rows, err := r.db.Query(
		fmt.Sprintf(detalleSelect+` %s ORDER BY i.nombre ASC LIMIT ? OFFSET ?`, whereClause),
		queryArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var lista []ItemConNombres
	for rows.Next() {
		d, err := scanItemConNombres(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if lista == nil {
		lista = []ItemConNombres{}
	}

	totalPaginas := 0
	if filtros.PorPagina > 0 {
		totalPaginas = (total + filtros.PorPagina - 1) / filtros.PorPagina
	}

	return &ListarItemsResponse{
		Items:        lista,
		Total:        total,
		Pagina:       filtros.Pagina,
		TotalPaginas: totalPaginas,
	}, nil
}

// Actualizar reemplaza los campos editables de un item existente.
func (r *mysqlRepository) Actualizar(id uint64, req *EditarItemRequest, userID uint64) (*Item, error) {
	now := time.Now()
	_, err := r.db.Exec(
		`UPDATE items SET
		   codigo = ?, nombre = ?, subcategoria_id = ?, proveedor_id = ?, unidad_medida_id = ?,
		   costo_unitario = ?, frecuencia_inventario = ?, stock_seguridad = ?, tiempo_entrega_dias = ?,
		   actualizado_por = ?, actualizado_en = ?
		 WHERE id = ?`,
		req.Codigo, req.Nombre, req.SubcategoriaID, req.ProveedorID, req.UnidadMedidaID,
		req.CostoUnitario, req.FrecuenciaInventario, req.StockSeguridad, req.TiempoEntregaDias,
		userID, now, id,
	)
	if err != nil {
		return nil, mapMySQLErr(err)
	}
	return r.ObtenerPorID(id)
}

// CambiarEstado actualiza el flag activo del item indicado.
func (r *mysqlRepository) CambiarEstado(id uint64, activo bool, userID uint64) (*Item, error) {
	val := 0
	if activo {
		val = 1
	}
	_, err := r.db.Exec(
		`UPDATE items SET activo = ?, actualizado_por = ?, actualizado_en = ? WHERE id = ?`,
		val, userID, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}
	return r.ObtenerPorID(id)
}

// EstaEnUso verifica si el item tiene al menos un registro en inventarios, recetas o pedidos.
// Retorna false de forma segura si alguna de esas tablas aún no existe (módulos 008/009/012-013
// pendientes de implementación).
func (r *mysqlRepository) EstaEnUso(itemID uint64) (bool, error) {
	tablas := []struct{ tabla, columna string }{
		{"inventarios_conteos_items", "item_id"},
		{"recetas_ingredientes", "item_id"},
		{"pedidos_lineas", "item_id"},
	}
	for _, t := range tablas {
		var count int
		query := fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE %s = ? LIMIT 1", t.tabla, t.columna)
		err := r.db.QueryRow(query, itemID).Scan(&count)
		if err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
				continue // tabla no creada aún
			}
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

// VerificarSubcategoria retorna si la subcategoría existe y si está activa.
func (r *mysqlRepository) VerificarSubcategoria(id uint64) (existe, activo bool, err error) {
	return verificarActivo(r.db, "subcategorias", id)
}

// VerificarProveedor retorna si el proveedor existe y si está activo.
func (r *mysqlRepository) VerificarProveedor(id uint64) (existe, activo bool, err error) {
	return verificarActivo(r.db, "proveedores", id)
}

// VerificarUnidadMedida retorna si la unidad de medida existe y si está activa.
func (r *mysqlRepository) VerificarUnidadMedida(id uint64) (existe, activo bool, err error) {
	return verificarActivo(r.db, "unidades_medida", id)
}

// VerificarTienda retorna si la tienda existe y si está activa.
func (r *mysqlRepository) VerificarTienda(id uint64) (existe, activo bool, err error) {
	return verificarActivo(r.db, "tiendas", id)
}

// verificarActivo es el helper compartido para consultar existencia + estado activo
// de las entidades referenciadas por items (subcategorias, proveedores, unidades_medida, tiendas).
// tabla es siempre una constante interna, nunca input de usuario.
func verificarActivo(db *sql.DB, tabla string, id uint64) (existe, activo bool, err error) {
	var activoInt int
	query := "SELECT activo FROM " + tabla + " WHERE id = ?"
	err = db.QueryRow(query, id).Scan(&activoInt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, activoInt == 1, nil
}

// InsertarCostoTienda inserta un nuevo registro histórico de costo por tienda (append-only).
func (r *mysqlRepository) InsertarCostoTienda(itemID uint64, req *RegistrarCostoTiendaRequest, userID uint64) (*ItemCostoTienda, error) {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO items_costos_tienda (item_id, tienda_id, costo_unitario, vigente_desde, creado_por, creado_en)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		itemID, req.TiendaID, req.CostoUnitario, now, userID, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de costo_tienda creado: %w", err)
	}
	row := r.db.QueryRow(
		`SELECT id, item_id, tienda_id, costo_unitario, vigente_desde, creado_por, creado_en
		   FROM items_costos_tienda WHERE id = ?`, id,
	)
	var c ItemCostoTienda
	if err := row.Scan(&c.ID, &c.ItemID, &c.TiendaID, &c.CostoUnitario, &c.VigenteDesde, &c.CreadoPor, &c.CreadoEn); err != nil {
		return nil, err
	}
	return &c, nil
}

// ListarCostosTienda retorna el historial completo de costos por tienda de un item,
// agrupado por tienda y ordenado por vigente_desde DESC (el primero de cada grupo es el vigente).
func (r *mysqlRepository) ListarCostosTienda(itemID uint64) ([]CostoPorTienda, error) {
	rows, err := r.db.Query(
		`SELECT ict.tienda_id, t.nombre, ict.id, ict.costo_unitario, ict.vigente_desde, ict.creado_por, ict.creado_en
		   FROM items_costos_tienda ict
		   JOIN tiendas t ON ict.tienda_id = t.id
		  WHERE ict.item_id = ?
		  ORDER BY ict.tienda_id ASC, ict.vigente_desde DESC`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var resultado []CostoPorTienda
	var actual *CostoPorTienda
	for rows.Next() {
		var tiendaID uint64
		var tiendaNombre string
		var entrada HistorialEntry
		if err := rows.Scan(&tiendaID, &tiendaNombre, &entrada.ID, &entrada.CostoUnitario, &entrada.VigenteDesde, &entrada.CreadoPor, &entrada.CreadoEn); err != nil {
			return nil, err
		}

		if actual == nil || actual.TiendaID != tiendaID {
			if actual != nil {
				resultado = append(resultado, *actual)
			}
			actual = &CostoPorTienda{
				TiendaID:     tiendaID,
				TiendaNombre: tiendaNombre,
				CostoVigente: entrada.CostoUnitario,
				Historial:    []HistorialEntry{},
			}
		}
		actual.Historial = append(actual.Historial, entrada)
	}
	if actual != nil {
		resultado = append(resultado, *actual)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if resultado == nil {
		resultado = []CostoPorTienda{}
	}
	return resultado, nil
}

func mapMySQLErr(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		if strings.Contains(mysqlErr.Message, "uq_items_codigo") {
			return ErrCodigoDuplicado
		}
		if strings.Contains(mysqlErr.Message, "uq_items_nombre") {
			return ErrNombreDuplicado
		}
		return ErrCodigoDuplicado
	}
	return err
}

// parseStockSeguridad valida que el string representa un decimal >= 0.
func parseStockSeguridad(s string) (bool, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return false, err
	}
	return v >= 0, nil
}
