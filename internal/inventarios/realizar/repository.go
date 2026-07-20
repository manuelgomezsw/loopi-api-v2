package realizar

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
)

// RepositoryImpl implementa la interfaz Repository para realizar
type RepositoryImpl struct {
	db *sql.DB
}

// NewRepository crea una instancia del repositorio de realizar
func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

// GetInventario obtiene un inventario existente
func (r *RepositoryImpl) GetInventario(ctx context.Context, inventarioID int64) (*core.Inventario, error) {
	query := `
		SELECT id, tienda_id, fecha, tipo, horario, estado, responsable_id,
		       iniciado_en, completado_en, creado_en, actualizado_en
		FROM inventarios
		WHERE id = ?
	`

	inv := &core.Inventario{}
	err := r.db.QueryRowContext(ctx, query, inventarioID).Scan(
		&inv.ID, &inv.TiendaID, &inv.Fecha, &inv.Tipo, &inv.Horario, &inv.Estado,
		&inv.ResponsableID, &inv.IniciadoEn, &inv.CompletadoEn, &inv.CreadoEn, &inv.ActualizadoEn,
	)

	if err == sql.ErrNoRows {
		return nil, ErrConteoNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo inventario: %w", err)
	}

	return inv, nil
}

// GetDetalleItem obtiene un item específico del conteo
func (r *RepositoryImpl) GetDetalleItem(ctx context.Context, inventarioID, itemID int64) (*core.DetalleInventario, error) {
	query := `
		SELECT d.id, d.inventario_id, d.item_id, d.valor_esperado, d.valor_real,
		       d.diferencia, d.creado_en, d.actualizado_en,
		       i.codigo, i.descripcion, um.nombre
		FROM detalle_inventario d
		JOIN items i ON d.item_id = i.id
		LEFT JOIN unidades_medida um ON i.unidad_medida_id = um.id
		WHERE d.inventario_id = ? AND d.item_id = ?
	`

	detalle := &core.DetalleInventario{}
	var codigo, descripcion, unidad sql.NullString

	err := r.db.QueryRowContext(ctx, query, inventarioID, itemID).Scan(
		&detalle.ID, &detalle.InventarioID, &detalle.ItemID, &detalle.ValorEsperado, &detalle.ValorReal,
		&detalle.Diferencia, &detalle.CreadoEn, &detalle.ActualizadoEn,
		&codigo, &descripcion, &unidad,
	)

	if err == sql.ErrNoRows {
		return nil, ErrItemNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo detalle: %w", err)
	}

	// Asignar valores desde el LEFT JOIN
	if codigo.Valid {
		detalle.Nombre = codigo.String + " - " + descripcion.String
	}

	return detalle, nil
}

// UpdateDetalle actualiza el valor real de un item
func (r *RepositoryImpl) UpdateDetalle(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*core.DetalleInventario, error) {
	query := `
		UPDATE detalle_inventario
		SET valor_real = ?, actualizado_en = NOW(), diferencia = valor_real - valor_esperado
		WHERE inventario_id = ? AND item_id = ?
		RETURNING id, inventario_id, item_id, valor_esperado, valor_real, diferencia, actualizado_en
	`

	detalle := &core.DetalleInventario{}
	err := r.db.QueryRowContext(ctx, query, valorReal, inventarioID, itemID).Scan(
		&detalle.ID, &detalle.InventarioID, &detalle.ItemID, &detalle.ValorEsperado, &detalle.ValorReal,
		&detalle.Diferencia, &detalle.ActualizadoEn,
	)

	if err == sql.ErrNoRows {
		return nil, ErrItemNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error actualizando detalle: %w", err)
	}

	return detalle, nil
}

// GetItems obtiene todos los items de un inventario con sus valores
func (r *RepositoryImpl) GetItems(ctx context.Context, inventarioID int64) ([]ItemDetalle, error) {
	query := `
		SELECT d.item_id, i.codigo, i.descripcion, d.valor_esperado, d.valor_real,
		       i.unidad_medida_id, um.nombre
		FROM detalle_inventario d
		JOIN items i ON d.item_id = i.id
		LEFT JOIN unidades_medida um ON i.unidad_medida_id = um.id
		WHERE d.inventario_id = ?
		ORDER BY d.creado_en
	`

	rows, err := r.db.QueryContext(ctx, query, inventarioID)
	if err != nil {
		return nil, fmt.Errorf("error consultando items: %w", err)
	}
	defer rows.Close()

	var items []ItemDetalle
	for rows.Next() {
		item := ItemDetalle{}
		var codigo, descripcion, unidad sql.NullString
		var unidadMedidaID sql.NullInt64

		if err := rows.Scan(&item.ItemID, &codigo, &descripcion, &item.ValorEsperado,
			&item.ValorReal, &unidadMedidaID, &unidad); err != nil {
			return nil, fmt.Errorf("error escaneando fila: %w", err)
		}

		// Asignar valores desde el LEFT JOIN
		if codigo.Valid {
			item.ItemCodigo = codigo.String
			item.ItemDescripcion = descripcion.String
		}
		if unidadMedidaID.Valid {
			item.UnidadMedidaID = unidadMedidaID.Int64
		}
		if unidad.Valid {
			item.Unidad = unidad.String
		}

		// Calcular diferencia y porcentaje si hay valor_real
		if item.ValorReal != nil {
			item.Completado = true
			diferencia := *item.ValorReal - item.ValorEsperado
			item.Diferencia = &diferencia
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando rows: %w", err)
	}

	return items, nil
}

// GetItemsDetalle obtiene todos los detalles de un inventario
func (r *RepositoryImpl) GetItemsDetalle(ctx context.Context, inventarioID int64) ([]core.DetalleInventario, error) {
	query := `
		SELECT id, inventario_id, item_id, valor_esperado, valor_real, diferencia,
		       creado_en, actualizado_en
		FROM detalle_inventario
		WHERE inventario_id = ?
		ORDER BY creado_en
	`

	rows, err := r.db.QueryContext(ctx, query, inventarioID)
	if err != nil {
		return nil, fmt.Errorf("error consultando detalles: %w", err)
	}
	defer rows.Close()

	var detalles []core.DetalleInventario
	for rows.Next() {
		detalle := core.DetalleInventario{}
		if err := rows.Scan(&detalle.ID, &detalle.InventarioID, &detalle.ItemID,
			&detalle.ValorEsperado, &detalle.ValorReal, &detalle.Diferencia,
			&detalle.CreadoEn, &detalle.ActualizadoEn); err != nil {
			return nil, fmt.Errorf("error escaneando fila: %w", err)
		}
		detalles = append(detalles, detalle)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando rows: %w", err)
	}

	return detalles, nil
}
