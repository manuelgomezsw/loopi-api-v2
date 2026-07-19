package inventarios

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository define la interfaz de acceso a datos
type Repository interface {
	// CreateInventario crea un nuevo inventario
	CreateInventario(ctx context.Context, inventario *Inventario) (*Inventario, error)

	// CreateDetalleInventario crea líneas de detalle para un inventario
	CreateDetalleInventario(ctx context.Context, detalles []DetalleInventario) error

	// GetInventario obtiene un inventario por ID
	GetInventario(ctx context.Context, id int64) (*Inventario, error)

	// GetInventarioDetalle obtiene un inventario con todos sus detalles
	GetInventarioDetalle(ctx context.Context, id int64) (*Inventario, error)

	// ListInventarios lista inventarios con paginación y filtros
	ListInventarios(ctx context.Context, filtros *FiltrosInventario) ([]*Inventario, int64, error)

	// UpdateDetalle actualiza el valor_real y diferencia de una línea en un inventario en_progreso
	UpdateDetalle(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*DetalleInventario, error)

	// UpdateDetalleCompletado actualiza una línea de un inventario completado
	UpdateDetalleCompletado(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*DetalleInventario, error)

	// ConfirmarInventario marca un inventario como completado
	ConfirmarInventario(ctx context.Context, id int64) (*Inventario, error)

	// DeleteInventario elimina un inventario en progreso
	DeleteInventario(ctx context.Context, id int64) error

	// GetStockReferenciaByTipo obtiene el stock de referencia para un tipo de inventario
	GetStockReferenciaByTipo(ctx context.Context, tiendaID int64, tipo Tipo, itemID int64) (*Inventario, float64, error)

	// SumarComprasPeriodo suma las compras en un período
	SumarComprasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error)

	// SumarVentasPeriodo suma las ventas en un período
	SumarVentasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error)

	// SumarMermasPeriodo suma las mermas en un período
	SumarMermasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error)

	// CanRecordMovimiento verifica si se puede registrar un movimiento (no hay conteo activo)
	CanRecordMovimiento(ctx context.Context, tiendaID int64) (bool, *int64, error)

	// SnapshotStockActual toma snapshot del stock actual al iniciar conteo
	SnapshotStockActual(ctx context.Context, inventario *Inventario, items []int64) error

	// RecordMovimiento registra un movimiento en la auditoría
	RecordMovimiento(ctx context.Context, movimiento *StockMovimiento) error

	// GetItemsActivosPorTipo obtiene los items activos para un tipo de inventario
	GetItemsActivosPorTipo(ctx context.Context, tiendaID int64, tipo Tipo) ([]int64, error)

	// GetStockSnapshot obtiene el valor_snapshot de items desde stock_actual
	GetStockSnapshot(ctx context.Context, tiendaID int64, itemIDs []int64) (map[int64]float64, error)
}

// RepositoryImpl implementa la interfaz Repository
type RepositoryImpl struct {
	db *sql.DB
}

// NewRepository crea una nueva instancia del repositorio
func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) CreateInventario(ctx context.Context, inventario *Inventario) (*Inventario, error) {
	// Verificar si ya existe un duplicado ANTES de insertar para poder retornar el estado
	checkQuery := `
		SELECT id, estado FROM inventarios
		WHERE tienda_id = ? AND fecha = ? AND tipo = ? AND horario <=> ?
		LIMIT 1
	`
	var existingID int64
	var existingState string
	err := r.db.QueryRowContext(ctx, checkQuery,
		inventario.TiendaID,
		inventario.Fecha,
		inventario.Tipo,
		inventario.Horario,
	).Scan(&existingID, &existingState)

	if err == nil {
		// Existe un duplicado - retornar error con detalles del estado
		details := map[string]interface{}{
			"conflicting_inventory_id": existingID,
			"conflicting_state":        existingState,
		}
		return nil, NewErrorWithDetails("conteo_duplicado", "Ya existe un conteo para esta tienda, tipo y horario en esta fecha", details)
	} else if err != sql.ErrNoRows {
		// Error en la query
		return nil, fmt.Errorf("error verificando duplicados: %w", err)
	}

	// No hay duplicado, proceder con INSERT
	query := `
		INSERT INTO inventarios
		(tienda_id, fecha, tipo, horario, estado, responsable_id, iniciado_en, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		inventario.TiendaID,
		inventario.Fecha,
		inventario.Tipo,
		inventario.Horario,
		inventario.Estado,
		inventario.ResponsableID,
		inventario.IniciadoEn,
		now,
		now,
	)

	if err != nil {
		if strings.Contains(err.Error(), "1062") && strings.Contains(err.Error(), "uq_inventarios") {
			details := map[string]interface{}{
				"conflicting_state": "unknown",
			}
			return nil, NewErrorWithDetails("conteo_duplicado", "Ya existe un conteo para esta tienda, tipo y horario en esta fecha", details)
		}
		return nil, fmt.Errorf("error creando inventario: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo ID: %w", err)
	}

	inventario.ID = id
	inventario.CreadoEn = now
	inventario.ActualizadoEn = now

	return inventario, nil
}

func (r *RepositoryImpl) CreateDetalleInventario(ctx context.Context, detalles []DetalleInventario) error {
	query := `
		INSERT INTO detalle_inventario
		(inventario_id, item_id, inventario_referencia_id, valor_sugerido, valor_esperado, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("error preparando statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, detail := range detalles {
		_, err := stmt.ExecContext(ctx,
			detail.InventarioID,
			detail.ItemID,
			detail.InventarioReferenciaID,
			detail.ValorSugerido,
			detail.ValorEsperado,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("error insertando detalle: %w", err)
		}
	}

	return nil
}

func (r *RepositoryImpl) GetInventario(ctx context.Context, id int64) (*Inventario, error) {
	query := `
		SELECT id, tienda_id, fecha, tipo, horario, estado, responsable_id,
		       iniciado_en, completado_en, creado_en, actualizado_en
		FROM inventarios WHERE id = ?
	`

	inv := &Inventario{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&inv.ID, &inv.TiendaID, &inv.Fecha, &inv.Tipo, &inv.Horario,
		&inv.Estado, &inv.ResponsableID, &inv.IniciadoEn, &inv.CompletadoEn,
		&inv.CreadoEn, &inv.ActualizadoEn,
	)

	if err == sql.ErrNoRows {
		return nil, NewError("not_found", "inventario no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo inventario: %w", err)
	}

	return inv, nil
}

func (r *RepositoryImpl) GetInventarioDetalle(ctx context.Context, id int64) (*Inventario, error) {
	inv, err := r.GetInventario(ctx, id)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, inventario_id, item_id, inventario_referencia_id,
		       valor_sugerido, valor_esperado, valor_real, diferencia,
		       creado_en, actualizado_en
		FROM detalle_inventario WHERE inventario_id = ?
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo detalles: %w", err)
	}
	defer rows.Close()

	inv.Items = []DetalleInventario{}
	for rows.Next() {
		detail := DetalleInventario{}
		err := rows.Scan(
			&detail.ID, &detail.InventarioID, &detail.ItemID, &detail.InventarioReferenciaID,
			&detail.ValorSugerido, &detail.ValorEsperado, &detail.ValorReal, &detail.Diferencia,
			&detail.CreadoEn, &detail.ActualizadoEn,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanneando detalle: %w", err)
		}
		inv.Items = append(inv.Items, detail)
	}

	return inv, rows.Err()
}

func (r *RepositoryImpl) ListInventarios(ctx context.Context, filtros *FiltrosInventario) ([]*Inventario, int64, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if filtros.TiendaID != nil {
		whereClause += " AND tienda_id = ?"
		args = append(args, *filtros.TiendaID)
	}
	if filtros.Tipo != nil {
		whereClause += " AND tipo = ?"
		args = append(args, *filtros.Tipo)
	}
	if filtros.Estado != nil {
		whereClause += " AND estado = ?"
		args = append(args, *filtros.Estado)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM inventarios %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error contando inventarios: %w", err)
	}

	offset := (int64(filtros.Pagina) - 1) * int64(filtros.PorPagina)
	selectQuery := fmt.Sprintf(`
		SELECT id, tienda_id, fecha, tipo, horario, estado, responsable_id,
		       iniciado_en, completado_en, creado_en, actualizado_en
		FROM inventarios %s
		ORDER BY fecha DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, filtros.PorPagina, offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error listando inventarios: %w", err)
	}
	defer rows.Close()

	inventarios := []*Inventario{}
	for rows.Next() {
		inv := &Inventario{}
		err := rows.Scan(
			&inv.ID, &inv.TiendaID, &inv.Fecha, &inv.Tipo, &inv.Horario,
			&inv.Estado, &inv.ResponsableID, &inv.IniciadoEn, &inv.CompletadoEn,
			&inv.CreadoEn, &inv.ActualizadoEn,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanneando inventario: %w", err)
		}
		inventarios = append(inventarios, inv)
	}

	return inventarios, total, rows.Err()
}

func (r *RepositoryImpl) UpdateDetalle(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*DetalleInventario, error) {
	query := `
		UPDATE detalle_inventario
		SET valor_real = ?, diferencia = ? - valor_esperado, actualizado_en = NOW()
		WHERE inventario_id = ? AND item_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, valorReal, valorReal, inventarioID, itemID)
	if err != nil {
		return nil, fmt.Errorf("error actualizando detalle: %w", err)
	}

	// Obtener el registro actualizado
	selectQuery := `
		SELECT id, inventario_id, item_id, valor_sugerido, valor_esperado,
		       valor_real, diferencia, creado_en, actualizado_en
		FROM detalle_inventario
		WHERE inventario_id = ? AND item_id = ?
	`

	detail := &DetalleInventario{}
	err = r.db.QueryRowContext(ctx, selectQuery, inventarioID, itemID).Scan(
		&detail.ID, &detail.InventarioID, &detail.ItemID,
		&detail.ValorSugerido, &detail.ValorEsperado,
		&detail.ValorReal, &detail.Diferencia,
		&detail.CreadoEn, &detail.ActualizadoEn,
	)

	if err != nil {
		return nil, fmt.Errorf("error obteniendo detalle actualizado: %w", err)
	}

	return detail, nil
}

func (r *RepositoryImpl) UpdateDetalleCompletado(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*DetalleInventario, error) {
	// Mismo SQL que UpdateDetalle pero para inventarios completados (sin bloqueo de estado)
	return r.UpdateDetalle(ctx, inventarioID, itemID, valorReal)
}

func (r *RepositoryImpl) ConfirmarInventario(ctx context.Context, id int64) (*Inventario, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	query := `
		UPDATE inventarios
		SET estado = ?, completado_en = ?, actualizado_en = ?
		WHERE id = ?
	`

	result, err := tx.ExecContext(ctx, query, EstadoCompletado, now, now, id)
	if err != nil {
		return nil, fmt.Errorf("error actualizando inventario: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return nil, NewError("not_found", "inventario no encontrado")
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("error confirmando transacción: %w", err)
	}

	inv := &Inventario{
		ID:           id,
		Estado:       EstadoCompletado,
		CompletadoEn: &now,
		ActualizadoEn: now,
	}

	return inv, nil
}

func (r *RepositoryImpl) DeleteInventario(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	// Eliminar detalles
	_, err = tx.ExecContext(ctx, "DELETE FROM detalle_inventario WHERE inventario_id = ?", id)
	if err != nil {
		return fmt.Errorf("error eliminando detalles: %w", err)
	}

	// Eliminar inventario
	result, err := tx.ExecContext(ctx, "DELETE FROM inventarios WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("error eliminando inventario: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return NewError("not_found", "inventario no encontrado")
	}

	return tx.Commit()
}

func (r *RepositoryImpl) GetStockReferenciaByTipo(ctx context.Context, tiendaID int64, tipo Tipo, itemID int64) (*Inventario, float64, error) {
	// Intentar obtener stock del mismo tipo
	query := `
		SELECT i.id, COALESCE(di.valor_real, 0)
		FROM inventarios i
		JOIN detalle_inventario di ON i.id = di.inventario_id
		WHERE i.tienda_id = ? AND i.tipo = ? AND i.estado = 'completado' AND di.item_id = ?
		ORDER BY i.completado_en DESC LIMIT 1
	`

	var invID int64
	var stock float64

	err := r.db.QueryRowContext(ctx, query, tiendaID, tipo, itemID).Scan(&invID, &stock)
	if err == nil {
		inv := &Inventario{ID: invID}
		return inv, stock, nil
	}

	// Si no encuentra del mismo tipo, buscar de cualquier tipo (respaldo per RD-03)
	queryRespaldo := `
		SELECT i.id, COALESCE(di.valor_real, 0)
		FROM inventarios i
		JOIN detalle_inventario di ON i.id = di.inventario_id
		WHERE i.tienda_id = ? AND i.estado = 'completado' AND di.item_id = ?
		ORDER BY i.completado_en DESC LIMIT 1
	`

	err = r.db.QueryRowContext(ctx, queryRespaldo, tiendaID, itemID).Scan(&invID, &stock)
	if err == sql.ErrNoRows {
		return nil, 0, nil // Sin inventario de referencia - retornar 0
	}
	if err != nil {
		return nil, 0, fmt.Errorf("error obteniendo stock referencia: %w", err)
	}

	inv := &Inventario{ID: invID}
	return inv, stock, nil
}

func (r *RepositoryImpl) SumarComprasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error) {
	// Verificar que la tabla existe
	tableExists, err := r.tableExists(ctx, "compras_caja_menor")
	if err != nil || !tableExists {
		return 0, nil // Tabla no existe - retornar 0
	}

	var suma float64
	query := `SELECT COALESCE(SUM(cantidad), 0) FROM compras_caja_menor
	          WHERE tienda_id = ? AND item_id = ? AND fecha BETWEEN ? AND ?`

	err = r.db.QueryRowContext(ctx, query, tiendaID, itemID, desde, hasta).Scan(&suma)
	if err != nil {
		return 0, fmt.Errorf("error sumando compras: %w", err)
	}

	return suma, nil
}

func (r *RepositoryImpl) SumarVentasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error) {
	// Verificar que la tabla existe
	tableExists, err := r.tableExists(ctx, "ventas_lineas")
	if err != nil || !tableExists {
		return 0, nil
	}

	var suma float64
	query := `SELECT COALESCE(SUM(cantidad), 0) FROM ventas_lineas
	          WHERE tienda_id = ? AND item_id = ? AND fecha BETWEEN ? AND ?`

	err = r.db.QueryRowContext(ctx, query, tiendaID, itemID, desde, hasta).Scan(&suma)
	if err != nil {
		return 0, fmt.Errorf("error sumando ventas: %w", err)
	}

	return suma, nil
}

func (r *RepositoryImpl) SumarMermasPeriodo(ctx context.Context, tiendaID int64, desde, hasta time.Time, itemID int64) (float64, error) {
	// Verificar que la tabla existe
	tableExists, err := r.tableExists(ctx, "mermas")
	if err != nil || !tableExists {
		return 0, nil
	}

	var suma float64
	query := `SELECT COALESCE(SUM(cantidad), 0) FROM mermas
	          WHERE tienda_id = ? AND item_id = ? AND fecha BETWEEN ? AND ?`

	err = r.db.QueryRowContext(ctx, query, tiendaID, itemID, desde, hasta).Scan(&suma)
	if err != nil {
		return 0, fmt.Errorf("error sumando mermas: %w", err)
	}

	return suma, nil
}

// tableExists verifica si una tabla existe en la BD usando information_schema
func (r *RepositoryImpl) tableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (
		SELECT 1 FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
	)`

	err := r.db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	return exists, err
}

// CanRecordMovimiento verifica si se puede registrar un movimiento
// Retorna (canRecord, activeCountID, error)
// canRecord=false si hay un conteo activo en la tienda
func (r *RepositoryImpl) CanRecordMovimiento(ctx context.Context, tiendaID int64) (bool, *int64, error) {
	query := `
		SELECT id FROM inventarios
		WHERE tienda_id = ? AND estado = 'en_progreso'
		LIMIT 1
	`

	var activeCountID int64
	err := r.db.QueryRowContext(ctx, query, tiendaID).Scan(&activeCountID)

	if err == sql.ErrNoRows {
		// No hay conteo activo - se puede registrar
		return true, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("error verificando conteo activo: %w", err)
	}

	// Hay un conteo activo - no se puede registrar
	return false, &activeCountID, nil
}

// SnapshotStockActual toma snapshot del stock actual al iniciar conteo
func (r *RepositoryImpl) SnapshotStockActual(ctx context.Context, inventario *Inventario, items []int64) error {
	if len(items) == 0 {
		return nil
	}

	query := `
		INSERT INTO stock_actual
		(tienda_id, item_id, inventario_id, valor_snapshot, tomado_en, creado_en)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("error preparando snapshot statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, itemID := range items {
		_, err := stmt.ExecContext(ctx,
			inventario.TiendaID,
			itemID,
			inventario.ID,
			0, // valor_snapshot será actualizado por la lógica de service
			now,
			now,
		)
		if err != nil {
			// Log warning - no bloquea por RD-04
			fmt.Printf("WARNING: error tomando snapshot para item %d: %v\n", itemID, err)
		}
	}

	return nil
}

// RecordMovimiento registra un movimiento en la tabla de auditoría
func (r *RepositoryImpl) RecordMovimiento(ctx context.Context, movimiento *StockMovimiento) error {
	query := `
		INSERT INTO stock_movimientos
		(tienda_id, item_id, tipo_movimiento, cantidad_antes, cantidad_despues,
		 cantidad_delta, referencia_id, referencia_tipo, usuario_id, motivo, creado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		movimiento.TiendaID,
		movimiento.ItemID,
		movimiento.TipoMovimiento,
		movimiento.CantidadAntes,
		movimiento.CantidadDespues,
		movimiento.CantidadDelta,
		movimiento.ReferenciaID,
		movimiento.ReferenciaTipo,
		movimiento.UsuarioID,
		movimiento.Motivo,
		now,
	)

	if err != nil {
		// Log error pero no bloquea - auditoría es no-blocking per spec
		fmt.Printf("ERROR registrando movimiento: tienda=%d, item=%d, tipo=%s, error=%v\n",
			movimiento.TiendaID, movimiento.ItemID, movimiento.TipoMovimiento, err)
		return nil // No retornar error para no bloquear la operación original
	}

	return nil
}

// GetItemsActivosPorTipo obtiene los IDs de items activos para un tipo de inventario (per spec 009)
// Items son multi-tenant (por tienda, per principio P-II de constitution)
// Query: SELECT id FROM items WHERE tienda_id=? AND activo=1 AND frecuencia_inventario=?
func (r *RepositoryImpl) GetItemsActivosPorTipo(ctx context.Context, tiendaID int64, tipo Tipo) ([]int64, error) {
	query := `
		SELECT id FROM items
		WHERE tienda_id = ? AND activo = 1 AND frecuencia_inventario = ?
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, tiendaID, string(tipo))
	if err != nil {
		return nil, fmt.Errorf("error obteniendo items activos: %w", err)
	}
	defer rows.Close()

	var itemIDs []int64
	for rows.Next() {
		var itemID int64
		if err := rows.Scan(&itemID); err != nil {
			return nil, fmt.Errorf("error scanneando item: %w", err)
		}
		itemIDs = append(itemIDs, itemID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando items: %w", err)
	}

	return itemIDs, nil
}

// GetStockSnapshot obtiene el valor_snapshot de items desde stock_actual
// Retorna map[item_id]valor_snapshot. Si item no existe en stock_actual, default es 0
func (r *RepositoryImpl) GetStockSnapshot(ctx context.Context, tiendaID int64, itemIDs []int64) (map[int64]float64, error) {
	if len(itemIDs) == 0 {
		return make(map[int64]float64), nil
	}

	placeholders := strings.Repeat("?,", len(itemIDs))
	placeholders = placeholders[:len(placeholders)-1] // Remover última coma

	query := fmt.Sprintf(`
		SELECT item_id, valor_snapshot FROM stock_actual
		WHERE tienda_id = ? AND item_id IN (%s)
	`, placeholders)

	args := make([]interface{}, len(itemIDs)+1)
	args[0] = tiendaID
	for i, id := range itemIDs {
		args[i+1] = id
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo stock snapshot: %w", err)
	}
	defer rows.Close()

	stocks := make(map[int64]float64)

	// Inicializar todos los items a 0
	for _, itemID := range itemIDs {
		stocks[itemID] = 0
	}

	// Sobrescribir con valores desde stock_actual
	for rows.Next() {
		var itemID int64
		var valorSnapshot float64
		if err := rows.Scan(&itemID, &valorSnapshot); err != nil {
			return nil, fmt.Errorf("error scanneando stock: %w", err)
		}
		stocks[itemID] = valorSnapshot
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando stocks: %w", err)
	}

	return stocks, nil
}
