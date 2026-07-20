package iniciar

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
)

// RepositoryImpl implementa la interfaz Repository para iniciar
type RepositoryImpl struct {
	db *sql.DB
}

// NewRepository crea una instancia del repositorio de iniciar
func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

// CreateInventario crea un nuevo inventario con validación de duplicados
func (r *RepositoryImpl) CreateInventario(ctx context.Context, inv *core.Inventario) (*core.Inventario, error) {
	checkQuery := `
		SELECT id, estado FROM inventarios
		WHERE tienda_id = ? AND fecha = ? AND tipo = ? AND horario <=> ?
		LIMIT 1
	`
	var existingID int64
	var existingState string
	err := r.db.QueryRowContext(ctx, checkQuery,
		inv.TiendaID,
		inv.Fecha,
		inv.Tipo,
		inv.Horario,
	).Scan(&existingID, &existingState)

	if err == nil {
		details := map[string]interface{}{
			"conflicting_inventory_id": existingID,
			"conflicting_state":        existingState,
		}
		msg := "Ya existe un conteo para esta tienda, tipo y horario en esta fecha"
		switch existingState {
		case "en_progreso":
			msg = "Ya existe un conteo en progreso para esta tienda, tipo y horario."
		case "completado":
			msg = "Ya existe un conteo completado para esta tienda, tipo y horario en esta fecha."
		}
		return nil, NewErrorWithDetails("conteo_duplicado", msg, details)
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("error verificando duplicados: %w", err)
	}

	query := `
		INSERT INTO inventarios
		(tienda_id, fecha, tipo, horario, estado, responsable_id, iniciado_en, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		inv.TiendaID,
		inv.Fecha,
		inv.Tipo,
		inv.Horario,
		inv.Estado,
		inv.ResponsableID,
		inv.IniciadoEn,
		now,
		now,
	)

	if err != nil {
		if strings.Contains(err.Error(), "1062") && strings.Contains(err.Error(), "uq_inventarios") {
			details := map[string]interface{}{
				"conflicting_state": "unknown",
			}
			return nil, NewErrorWithDetails("conteo_duplicado", "Ya existe un conteo duplicado.", details)
		}
		return nil, fmt.Errorf("error creando inventario: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo ID: %w", err)
	}

	inv.ID = id
	inv.CreadoEn = now
	inv.ActualizadoEn = now

	return inv, nil
}

// CreateDetalleInventario crea los detalles de un inventario
func (r *RepositoryImpl) CreateDetalleInventario(ctx context.Context, detalles []core.DetalleInventario) error {
	query := `
		INSERT INTO detalle_inventario
		(inventario_id, item_id, inventario_referencia_id, valor_esperado, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("error preparando statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now()
	for _, detail := range detalles {
		_, err := stmt.ExecContext(ctx,
			detail.InventarioID,
			detail.ItemID,
			detail.InventarioReferenciaID,
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

// GetInventarioDetalle obtiene un inventario con todos sus detalles
func (r *RepositoryImpl) GetInventarioDetalle(ctx context.Context, id int64) (*core.Inventario, error) {
	query := `
		SELECT id, tienda_id, fecha, tipo, horario, estado, responsable_id,
		       iniciado_en, completado_en, creado_en, actualizado_en
		FROM inventarios WHERE id = ?
	`

	inv := &core.Inventario{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&inv.ID, &inv.TiendaID, &inv.Fecha, &inv.Tipo, &inv.Horario,
		&inv.Estado, &inv.ResponsableID, &inv.IniciadoEn, &inv.CompletadoEn,
		&inv.CreadoEn, &inv.ActualizadoEn,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventario no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo inventario: %w", err)
	}

	detQuery := `
		SELECT di.id, di.inventario_id, di.item_id, di.inventario_referencia_id,
		       di.valor_esperado, di.valor_real, di.diferencia,
		       di.creado_en, di.actualizado_en, i.nombre, i.unidad_medida_id
		FROM detalle_inventario di
		JOIN items i ON di.item_id = i.id
		WHERE di.inventario_id = ?
		ORDER BY di.id
	`

	rows, err := r.db.QueryContext(ctx, detQuery, id)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo detalles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	inv.Items = []core.DetalleInventario{}
	for rows.Next() {
		detail := core.DetalleInventario{}
		err := rows.Scan(
			&detail.ID, &detail.InventarioID, &detail.ItemID, &detail.InventarioReferenciaID,
			&detail.ValorEsperado, &detail.ValorReal, &detail.Diferencia,
			&detail.CreadoEn, &detail.ActualizadoEn, &detail.Nombre, &detail.UnidadMedidaID,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanneando detalle: %w", err)
		}
		inv.Items = append(inv.Items, detail)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inv, nil
}

// GetItemsActivosPorTipo obtiene los IDs de items activos para un tipo de inventario
func (r *RepositoryImpl) GetItemsActivosPorTipo(ctx context.Context, tiendaID int64, tipo core.Tipo) ([]int64, error) {
	query := `
		SELECT id FROM items
		WHERE tienda_id = ? AND activo = 1 AND frecuencia_inventario = ?
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, tiendaID, string(tipo))
	if err != nil {
		return nil, fmt.Errorf("error obteniendo items activos: %w", err)
	}
	defer func() { _ = rows.Close() }()

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
func (r *RepositoryImpl) GetStockSnapshot(ctx context.Context, tiendaID int64, itemIDs []int64) (map[int64]float64, error) {
	if len(itemIDs) == 0 {
		return make(map[int64]float64), nil
	}

	placeholders := strings.Repeat("?,", len(itemIDs))
	placeholders = placeholders[:len(placeholders)-1]

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
	defer func() { _ = rows.Close() }()

	stocks := make(map[int64]float64)

	for _, itemID := range itemIDs {
		stocks[itemID] = 0
	}

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

// ExisteInventarioCompletado verifica si existe al menos un inventario completado en una tienda
func (r *RepositoryImpl) ExisteInventarioCompletado(ctx context.Context, tiendaID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM inventarios
			WHERE tienda_id = ? AND estado = 'completado'
			LIMIT 1
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tiendaID).Scan(&exists)
	if err != nil {
		return false, nil
	}

	return exists, nil
}
