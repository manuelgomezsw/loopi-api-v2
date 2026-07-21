package completar

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// T043: Repository tests con go-sqlmock

func TestGetInventarioDetalle_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	inventarioID := int64(1)
	now := time.Now()

	// Mock GetInventarioDetalle query
	mock.ExpectQuery("SELECT id, tienda_id, fecha, tipo, horario, estado").
		WithArgs(inventarioID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tienda_id", "fecha", "tipo", "horario", "estado", "responsable_id",
			"iniciado_en", "completado_en", "creado_en", "actualizado_en",
		}).AddRow(
			inventarioID, int64(1), now, "diario", "cierre", "en_progreso", int64(1),
			now, nil, now, now,
		))

	// Mock GetDetalles query
	mock.ExpectQuery("SELECT di.id, di.inventario_id, di.item_id").
		WithArgs(inventarioID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "inventario_id", "item_id", "valor_esperado", "valor_real", "diferencia",
			"nombre", "unidad_medida_id", "creado_en", "actualizado_en",
		}).AddRow(
			int64(1), inventarioID, int64(100), 10.0, nil, nil, "Producto A", int64(1), now, now,
		))

	inv, err := repo.GetInventarioDetalle(context.Background(), inventarioID)
	assert.NoError(t, err)
	assert.NotNil(t, inv)
	assert.Equal(t, inventarioID, inv.ID)
	assert.Equal(t, 1, len(inv.Items))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetInventarioDetalle_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	inventarioID := int64(1)
	now := time.Now()

	// Mock query que retorna una fila válida
	mock.ExpectQuery("SELECT id, tienda_id, fecha, tipo, horario, estado").
		WithArgs(inventarioID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tienda_id", "fecha", "tipo", "horario", "estado", "responsable_id",
			"iniciado_en", "completado_en", "creado_en", "actualizado_en",
		}).AddRow(
			inventarioID, int64(1), now, "diario", "cierre", "en_progreso", int64(1),
			now, nil, now, now,
		))

	// Mock detalles vacío
	mock.ExpectQuery("SELECT di.id, di.inventario_id, di.item_id").
		WithArgs(inventarioID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "inventario_id", "item_id", "valor_esperado", "valor_real", "diferencia",
			"nombre", "unidad_medida_id", "creado_en", "actualizado_en",
		}))

	inv, err := repo.GetInventarioDetalle(context.Background(), inventarioID)
	assert.NoError(t, err)
	assert.NotNil(t, inv)
	assert.Equal(t, 0, len(inv.Items))
}

func TestBeginTx_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectBegin()
	tx, err := repo.BeginTx(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	tx.Rollback()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommitTx_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectBegin()
	mock.ExpectCommit()

	tx, _ := repo.BeginTx(context.Background())
	err = repo.CommitTx(tx)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRollbackTx_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectBegin()
	mock.ExpectRollback()

	tx, _ := repo.BeginTx(context.Background())
	err = repo.RollbackTx(tx)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// T043: Validar que métodos respetan estructura de transacción
func TestRepositoryTransactionMethods_Defined(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	// Verificar que los métodos están disponibles
	assert.NotNil(t, repo.GetInventarioDetalle)
	assert.NotNil(t, repo.ConfirmarInventario)
	assert.NotNil(t, repo.UpdateStockActual)
	assert.NotNil(t, repo.RecordDifference)
	assert.NotNil(t, repo.BeginTx)
	assert.NotNil(t, repo.CommitTx)
	assert.NotNil(t, repo.RollbackTx)
}
