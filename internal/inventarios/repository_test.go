package inventarios

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateInventario_Success verifica que se puede crear un inventario exitosamente
func TestCreateInventario_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now()
	horario := HorarioApertura

	inv := &Inventario{
		TiendaID:      1,
		Fecha:         now,
		Tipo:          TipoDiario,
		Horario:       &horario,
		Estado:        EstadoEnProgreso,
		ResponsableID: 10,
		IniciadoEn:    now,
	}

	mock.ExpectExec("INSERT INTO inventarios").
		WithArgs(
			inv.TiendaID,
			inv.Fecha,
			inv.Tipo,
			inv.Horario,
			inv.Estado,
			inv.ResponsableID,
			inv.IniciadoEn,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := repo.CreateInventario(context.Background(), inv)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateInventario_DuplicateConstraint verifica que se rechaza duplicados
func TestCreateInventario_DuplicateConstraint(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now()
	horario := HorarioApertura

	inv := &Inventario{
		TiendaID:      1,
		Fecha:         now,
		Tipo:          TipoDiario,
		Horario:       &horario,
		Estado:        EstadoEnProgreso,
		ResponsableID: 10,
		IniciadoEn:    now,
	}

	mock.ExpectExec("INSERT INTO inventarios").
		WillReturnError(sql.ErrNoRows)

	_, err = repo.CreateInventario(context.Background(), inv)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetInventario_Success verifica que se puede obtener un inventario
func TestGetInventario_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now()

	rows := sqlmock.NewRows(
		[]string{"id", "tienda_id", "fecha", "tipo", "horario", "estado", "responsable_id", "iniciado_en", "completado_en", "creado_en", "actualizado_en"},
	).AddRow(
		1, 1, now, "diario", "apertura", "en_progreso", 10, now, nil, now, now,
	)

	mock.ExpectQuery("SELECT .* FROM inventarios WHERE id").
		WithArgs(1).
		WillReturnRows(rows)

	result, err := repo.GetInventario(context.Background(), 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetInventarioDetalle_WithItems verifica que trae inventario con detalles
func TestGetInventarioDetalle_WithItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now()

	invRows := sqlmock.NewRows(
		[]string{"id", "tienda_id", "fecha", "tipo", "horario", "estado", "responsable_id", "iniciado_en", "completado_en", "creado_en", "actualizado_en"},
	).AddRow(
		1, 1, now, "diario", "apertura", "en_progreso", 10, now, nil, now, now,
	)

	itemRows := sqlmock.NewRows(
		[]string{"id", "inventario_id", "item_id", "inventario_referencia_id", "valor_sugerido", "valor_esperado", "valor_real", "diferencia", "creado_en", "actualizado_en"},
	).AddRow(
		1, 1, 100, nil, 10.0, 10.0, nil, nil, now, now,
	)

	mock.ExpectQuery("SELECT .* FROM inventarios WHERE id").
		WithArgs(1).
		WillReturnRows(invRows)

	mock.ExpectQuery("SELECT .* FROM detalle_inventario WHERE inventario_id").
		WithArgs(1).
		WillReturnRows(itemRows)

	result, err := repo.GetInventarioDetalle(context.Background(), 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.ID)
	assert.Len(t, result.Items, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateDetalle_Success verifica que se puede actualizar valor_real
func TestUpdateDetalle_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now()

	mock.ExpectExec("UPDATE detalle_inventario SET valor_real").
		WithArgs(12.5, 12.5, int64(1), int64(100)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rows := sqlmock.NewRows(
		[]string{"id", "inventario_id", "item_id", "valor_sugerido", "valor_esperado", "valor_real", "diferencia", "creado_en", "actualizado_en"},
	).AddRow(
		1, 1, 100, 10.0, 10.0, 12.5, 2.5, now, now,
	)

	mock.ExpectQuery("SELECT .* FROM detalle_inventario WHERE inventario_id").
		WithArgs(int64(1), int64(100)).
		WillReturnRows(rows)

	result, err := repo.UpdateDetalle(context.Background(), 1, 100, 12.5)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 12.5, *result.ValorReal)
	assert.Equal(t, 2.5, *result.Diferencia)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListInventarios_WithFilters verifica paginación y filtros
func TestListInventarios_WithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now()

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(countRows)

	dataRows := sqlmock.NewRows(
		[]string{"id", "tienda_id", "fecha", "tipo", "horario", "estado", "responsable_id", "iniciado_en", "completado_en", "creado_en", "actualizado_en"},
	).AddRow(
		1, 1, now, "diario", "apertura", "completado", 10, now, now, now, now,
	)

	mock.ExpectQuery("SELECT .* FROM inventarios").
		WillReturnRows(dataRows)

	tiendaID := int64(1)
	filtros := &FiltrosInventario{
		TiendaID:  &tiendaID,
		Pagina:    1,
		PorPagina: 50,
	}

	results, total, err := repo.ListInventarios(context.Background(), filtros)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetStockReferenciaByTipo_Success verifica búsqueda de stock referencia
func TestGetStockReferenciaByTipo_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	// Mock query para búsqueda por tipo
	stockRows := sqlmock.NewRows([]string{"id", "stock"}).
		AddRow(5, 11.0)

	mock.ExpectQuery("SELECT i.id, COALESCE").
		WithArgs(int64(1), "diario", int64(100)).
		WillReturnRows(stockRows)

	inv, stock, err := repo.GetStockReferenciaByTipo(context.Background(), 1, "diario", 100)

	assert.NoError(t, err)
	assert.NotNil(t, inv)
	assert.Equal(t, 11.0, stock)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetItemsActivosPorTipo_Success verifica que se obtienen items activos para un tipo
// T159: Unit test para GetItemsActivosPorTipo()
func TestGetItemsActivosPorTipo_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	// Mock query para items activos con frecuencia_inventario='diario'
	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(int64(501)).
		AddRow(int64(502)).
		AddRow(int64(503))

	mock.ExpectQuery("SELECT id FROM items WHERE tienda_id").
		WithArgs(int64(1), "diario").
		WillReturnRows(rows)

	result, err := repo.GetItemsActivosPorTipo(context.Background(), 1, TipoDiario)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)
	assert.Equal(t, int64(501), result[0])
	assert.Equal(t, int64(502), result[1])
	assert.Equal(t, int64(503), result[2])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetItemsActivosPorTipo_EmptyList verifica que retorna lista vacía si no hay items
// T159: Verifica que retorna lista vacía cuando no hay items para tipo
func TestGetItemsActivosPorTipo_EmptyList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	// Mock query que retorna cero rows
	rows := sqlmock.NewRows([]string{"id"})

	mock.ExpectQuery("SELECT id FROM items WHERE tienda_id").
		WithArgs(int64(1), "semanal").
		WillReturnRows(rows)

	result, err := repo.GetItemsActivosPorTipo(context.Background(), 1, TipoSemanal)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetItemsActivosPorTipo_ExcludesInactive verifica que excluye items inactivos
// T159: Verifica que solo retorna items con activo=1
func TestGetItemsActivosPorTipo_ExcludesInactive(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	// Mock query - la cláusula WHERE activo=1 está en la query, así que no hay inactivos aquí
	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(int64(501))

	mock.ExpectQuery("SELECT id FROM items WHERE tienda_id = \\? AND activo = 1 AND frecuencia_inventario = \\?").
		WithArgs(int64(2), "diario").
		WillReturnRows(rows)

	result, err := repo.GetItemsActivosPorTipo(context.Background(), 2, TipoDiario)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(501), result[0])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetStockSnapshot_Success verifica que obtiene valores desde stock_actual
// T160: Unit test para GetStockSnapshot()
func TestGetStockSnapshot_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	// Mock query para stock_actual
	rows := sqlmock.NewRows([]string{"item_id", "valor_snapshot"}).
		AddRow(int64(501), 50.0).
		AddRow(int64(502), 45.5)

	mock.ExpectQuery("SELECT item_id, valor_snapshot FROM stock_actual WHERE tienda_id").
		WithArgs(int64(1), int64(501), int64(502)).
		WillReturnRows(rows)

	itemIDs := []int64{501, 502}
	result, err := repo.GetStockSnapshot(context.Background(), 1, itemIDs)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 50.0, result[501])
	assert.Equal(t, 45.5, result[502])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetStockSnapshot_DefaultZero verifica que default es 0 para items no en stock_actual
// T160: Verifica que default 0 si item no existe en stock_actual
func TestGetStockSnapshot_DefaultZero(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	// Mock: solo item 501 está en stock_actual, item 502 no
	rows := sqlmock.NewRows([]string{"item_id", "valor_snapshot"}).
		AddRow(int64(501), 50.0)

	mock.ExpectQuery("SELECT item_id, valor_snapshot FROM stock_actual WHERE tienda_id").
		WithArgs(int64(1), int64(501), int64(502)).
		WillReturnRows(rows)

	itemIDs := []int64{501, 502}
	result, err := repo.GetStockSnapshot(context.Background(), 1, itemIDs)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 50.0, result[501])
	assert.Equal(t, 0.0, result[502]) // Default 0
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetStockSnapshot_EmptyList verifica que maneja lista vacía de items
// T160: Verifica que retorna mapa vacío para lista vacía de items
func TestGetStockSnapshot_EmptyList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	itemIDs := []int64{}
	result, err := repo.GetStockSnapshot(context.Background(), 1, itemIDs)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	// No mock expectations needed for empty list - se retorna early
	assert.NoError(t, mock.ExpectationsWereMet())
}
