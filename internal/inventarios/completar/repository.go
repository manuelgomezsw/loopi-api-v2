package completar

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
)

// RepositoryImpl implementa los métodos de acceso a datos para completar conteo
type RepositoryImpl struct {
	db *sql.DB
}

// NewRepository crea una nueva instancia del repository
func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

// GetInventarioDetalle obtiene un inventario con todos sus detalles
// T036: Copiar desde 009 repository.go
func (r *RepositoryImpl) GetInventarioDetalle(ctx context.Context, inventarioID int64) (*core.Inventario, error) {
	// Obtener inventario base
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
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error obteniendo inventario: %w", err)
	}

	// Obtener detalles
	detailQuery := `
		SELECT di.id, di.inventario_id, di.item_id, di.valor_esperado,
		       di.valor_real, di.diferencia, di.nombre, di.unidad_medida_id,
		       di.creado_en, di.actualizado_en
		FROM detalle_inventario di
		WHERE di.inventario_id = ?
		ORDER BY di.id
	`

	rows, err := r.db.QueryContext(ctx, detailQuery, inventarioID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo detalles: %w", err)
	}
	defer rows.Close()

	inv.Items = []core.DetalleInventario{}
	for rows.Next() {
		detail := core.DetalleInventario{}
		err := rows.Scan(
			&detail.ID, &detail.InventarioID, &detail.ItemID, &detail.ValorEsperado,
			&detail.ValorReal, &detail.Diferencia, &detail.Nombre, &detail.UnidadMedidaID,
			&detail.CreadoEn, &detail.ActualizadoEn,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanneando detalle: %w", err)
		}
		inv.Items = append(inv.Items, detail)
	}

	return inv, rows.Err()
}

// ConfirmarInventario cambia el estado a completado
// T037: Copiar desde 009 ConfirmarInventario
func (r *RepositoryImpl) ConfirmarInventario(ctx context.Context, tx *sql.Tx, inventarioID int64) error {
	now := time.Now()
	query := `
		UPDATE inventarios
		SET estado = ?, completado_en = ?, actualizado_en = ?
		WHERE id = ?
	`

	result, err := tx.ExecContext(ctx, query, core.EstadoCompletado, now, now, inventarioID)
	if err != nil {
		return fmt.Errorf("error actualizando inventario: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("inventario no encontrado")
	}

	return nil
}

// UpdateStockActual ajusta el stock de un item
// T038: Copiar desde 009 y adaptar para transacción
func (r *RepositoryImpl) UpdateStockActual(ctx context.Context, tx *sql.Tx, tiendaID, itemID int64, nuevoValor float64) error {
	query := `
		UPDATE stock_actual
		SET cantidad = ?, actualizado_en = ?
		WHERE tienda_id = ? AND item_id = ?
	`

	result, err := tx.ExecContext(ctx, query, nuevoValor, time.Now(), tiendaID, itemID)
	if err != nil {
		return fmt.Errorf("error actualizando stock: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("stock no encontrado para item")
	}

	return nil
}

// RecordDifference registra la diferencia en detalle_inventario
// T039: Copiar desde 009 y adaptar
func (r *RepositoryImpl) RecordDifference(ctx context.Context, tx *sql.Tx, detalleID int64, diferencia float64) error {
	query := `
		UPDATE detalle_inventario
		SET diferencia = ?, actualizado_en = ?
		WHERE id = ?
	`

	result, err := tx.ExecContext(ctx, query, diferencia, time.Now(), detalleID)
	if err != nil {
		return fmt.Errorf("error registrando diferencia: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("detalle no encontrado")
	}

	return nil
}

// BeginTx inicia una transacción
// T040: Helper para transacciones atómicas
func (r *RepositoryImpl) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción: %w", err)
	}
	return tx, nil
}

// CommitTx confirma la transacción
func (r *RepositoryImpl) CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

// RollbackTx revierte la transacción
func (r *RepositoryImpl) RollbackTx(tx *sql.Tx) error {
	return tx.Rollback()
}
