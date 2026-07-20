package core

import (
	"context"
	"database/sql"
)

// Repository define los métodos compartidos por múltiples sub-dominios
type Repository interface {
	// SnapshotStockActual obtiene el snapshot de stock actual para todos los items de una tienda
	// Usado por: iniciar (al cargar items), completar (al confirmar)
	SnapshotStockActual(ctx context.Context, tiendaID int64) (map[int64]float64, error)

	// GetInventarioDetalle obtiene los detalles completos de un inventario con items
	// Usado por: realizar, completar, historial, editar
	GetInventarioDetalle(ctx context.Context, inventarioID int64) (*Inventario, error)

	// CanRecordMovimiento verifica si se puede registrar un movimiento en la tienda (RF-INV-05)
	// Devuelve nil si no hay conteo en progreso, error si hay bloqueo
	CanRecordMovimiento(ctx context.Context, tiendaID int64) error

	// RecordMovimiento registra un movimiento de stock en la tabla de auditoría
	// Usado por: completar (al ajustar stock)
	RecordMovimiento(ctx context.Context, mov *StockMovimiento) error
}

// Concrete repository implementation (placeholder para TDD)
type concreteRepository struct {
	db *sql.DB
}

// NewRepository crea una instancia concreta del repositorio compartido
func NewRepository(db *sql.DB) Repository {
	return &concreteRepository{db: db}
}

// SnapshotStockActual implementa obtener stock snapshot para una tienda
func (r *concreteRepository) SnapshotStockActual(ctx context.Context, tiendaID int64) (map[int64]float64, error) {
	query := `
		SELECT item_id, valor_snapshot
		FROM stock_actual
		WHERE tienda_id = ?
	`
	rows, err := r.db.QueryContext(ctx, query, tiendaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]float64)
	for rows.Next() {
		var itemID int64
		var valor float64
		if err := rows.Scan(&itemID, &valor); err != nil {
			return nil, err
		}
		result[itemID] = valor
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetInventarioDetalle implementa obtener inventario completo con detalles
func (r *concreteRepository) GetInventarioDetalle(ctx context.Context, inventarioID int64) (*Inventario, error) {
	query := `
		SELECT id, tienda_id, fecha, tipo, horario, estado, responsable_id,
		       iniciado_en, completado_en, creado_en, actualizado_en
		FROM inventarios
		WHERE id = ?
	`
	inv := &Inventario{}
	if err := r.db.QueryRowContext(ctx, query, inventarioID).Scan(
		&inv.ID, &inv.TiendaID, &inv.Fecha, &inv.Tipo, &inv.Horario,
		&inv.Estado, &inv.ResponsableID, &inv.IniciadoEn, &inv.CompletadoEn,
		&inv.CreadoEn, &inv.ActualizadoEn,
	); err != nil {
		return nil, err
	}

	// Cargar detalles
	detQuery := `
		SELECT id, inventario_id, item_id, nombre, unidad_medida_id,
		       inventario_referencia_id, valor_esperado, valor_real, diferencia,
		       creado_en, actualizado_en
		FROM detalle_inventario
		WHERE inventario_id = ?
	`
	rows, err := r.db.QueryContext(ctx, detQuery, inventarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inv.Items = make([]DetalleInventario, 0)
	for rows.Next() {
		det := DetalleInventario{}
		if err := rows.Scan(
			&det.ID, &det.InventarioID, &det.ItemID, &det.Nombre, &det.UnidadMedidaID,
			&det.InventarioReferenciaID, &det.ValorEsperado, &det.ValorReal, &det.Diferencia,
			&det.CreadoEn, &det.ActualizadoEn,
		); err != nil {
			return nil, err
		}
		inv.Items = append(inv.Items, det)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inv, nil
}

// CanRecordMovimiento verifica si hay un conteo en progreso (bloquea movimientos durante conteo)
func (r *concreteRepository) CanRecordMovimiento(ctx context.Context, tiendaID int64) error {
	query := `
		SELECT 1 FROM inventarios
		WHERE tienda_id = ? AND estado = 'en_progreso'
		LIMIT 1
	`
	var exists int
	err := r.db.QueryRowContext(ctx, query, tiendaID).Scan(&exists)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	// Si hay un conteo en progreso, bloquear el movimiento
	return ErrInventarioEnProgreso
}

// RecordMovimiento registra un movimiento en la tabla de auditoría
func (r *concreteRepository) RecordMovimiento(ctx context.Context, mov *StockMovimiento) error {
	query := `
		INSERT INTO stock_movimientos
		(tienda_id, item_id, tipo_movimiento, cantidad_antes, cantidad_despues, cantidad_delta,
		 referencia_id, referencia_tipo, usuario_id, motivo, creado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		mov.TiendaID, mov.ItemID, mov.TipoMovimiento,
		mov.CantidadAntes, mov.CantidadDespues, mov.CantidadDelta,
		mov.ReferenciaID, mov.ReferenciaTipo, mov.UsuarioID, mov.Motivo, mov.CreadoEn,
	)
	return err
}
