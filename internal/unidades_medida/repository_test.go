package unidades_medida_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	mysql "github.com/go-sql-driver/mysql"
	um "github.com/manuelgomezsw/loopi-api-v2/internal/unidades_medida"
)

// newMockDB crea un *sql.DB con sqlmock para cada test.
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creando sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func umCols() []string {
	return []string{"id", "codigo", "nombre", "tipo_medida", "factor_conversion", "unidad_base", "activo", "creado_en", "actualizado_en"}
}

func addUMRow(rows *sqlmock.Rows, u *um.UnidadMedida) *sqlmock.Rows {
	ub := 0
	if u.UnidadBase {
		ub = 1
	}
	ac := 0
	if u.Activo {
		ac = 1
	}
	return rows.AddRow(u.ID, u.Codigo, u.Nombre, u.TipoMedida, u.FactorConversion, ub, ac, u.CreadoEn, u.ActualizadoEn)
}

func expectObtenerPorID(mock sqlmock.Sqlmock, u *um.UnidadMedida) {
	rows := addUMRow(sqlmock.NewRows(umCols()), u)
	mock.ExpectQuery(`SELECT`).WithArgs(u.ID).WillReturnRows(rows)
}

// --- ObtenerPorID ---

func TestRepository_ObtenerPorID_Encontrada(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	u := umEjemplo()
	expectObtenerPorID(mock, u)

	resultado, err := repo.ObtenerPorID(u.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.Codigo != u.Codigo {
		t.Errorf("codigo esperado %q, obtenido %q", u.Codigo, resultado.Codigo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_ObtenerPorID_NoExiste(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	mock.ExpectQuery(`SELECT`).WithArgs(uint64(999)).WillReturnError(sql.ErrNoRows)

	_, err := repo.ObtenerPorID(999)
	if !errors.Is(err, um.ErrUnidadNoEncontrada) {
		t.Errorf("esperaba ErrUnidadNoEncontrada, obtuvo: %v", err)
	}
}

// --- ExisteConCodigo ---

func TestRepository_ExisteConCodigo_Existe(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	mock.ExpectQuery(`SELECT COUNT`).WithArgs("kg").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	existe, err := repo.ExisteConCodigo("kg")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !existe {
		t.Error("esperaba existe=true")
	}
}

func TestRepository_ExisteConCodigo_NoExiste(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	mock.ExpectQuery(`SELECT COUNT`).WithArgs("oz").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	existe, err := repo.ExisteConCodigo("oz")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if existe {
		t.Error("esperaba existe=false")
	}
}

// --- Crear ---

func TestRepository_Crear_OK(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	u := umEjemplo()

	mock.ExpectExec(`INSERT INTO unidades_medida`).
		WillReturnResult(sqlmock.NewResult(int64(u.ID), 1))
	expectObtenerPorID(mock, u)

	req := &um.CrearUMRequest{
		Codigo: u.Codigo, Nombre: u.Nombre, TipoMedida: u.TipoMedida, FactorConversion: u.FactorConversion,
	}
	resultado, err := repo.Crear(req)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.ID != u.ID {
		t.Errorf("id esperado %d, obtenido %d", u.ID, resultado.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_Crear_CodigoDuplicado(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	mock.ExpectExec(`INSERT INTO unidades_medida`).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'kg' for key 'uq_unidades_medida_codigo'"})

	_, err := repo.Crear(&um.CrearUMRequest{Codigo: "kg"})
	if !errors.Is(err, um.ErrCodigoDuplicado) {
		t.Errorf("esperaba ErrCodigoDuplicado, obtuvo: %v", err)
	}
}

// --- Listar ---

func TestRepository_Listar_SinFiltros(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	u := umEjemplo()

	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := addUMRow(sqlmock.NewRows(umCols()), u)
	mock.ExpectQuery(`SELECT`).WithArgs(50, 0).WillReturnRows(rows)

	resp, err := repo.Listar(&um.ListarUMParams{Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total esperado 1, obtenido %d", resp.Total)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_Listar_FiltroTipo(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(1\) FROM unidades_medida WHERE tipo_medida`).
		WithArgs("peso").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	mock.ExpectQuery(`SELECT`).WithArgs("peso", 50, 0).
		WillReturnRows(sqlmock.NewRows(umCols()))

	_, err := repo.Listar(&um.ListarUMParams{Tipo: "peso", Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

// --- Editar ---

func TestRepository_Editar_OK(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	u := umEjemplo()
	u.Nombre = "Kilogramos"

	mock.ExpectExec(`UPDATE unidades_medida SET nombre`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectObtenerPorID(mock, u)

	n := "Kilogramos"
	resultado, err := repo.Editar(u.ID, &um.EditarUMRequest{Nombre: &n})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.Nombre != "Kilogramos" {
		t.Errorf("nombre esperado 'Kilogramos', obtenido %q", resultado.Nombre)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

// --- Inactivar ---

func TestRepository_Inactivar_OK(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)

	mock.ExpectExec(`UPDATE unidades_medida SET activo = 0`).
		WithArgs(sqlmock.AnyArg(), uint64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Inactivar(4)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

// --- ContarItemsConUnidadCanonica ---

func TestRepository_ContarItemsConUnidadCanonica_TablaNoExiste(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)

	mock.ExpectQuery(`SELECT COUNT`).WithArgs(uint64(4)).
		WillReturnError(&mysql.MySQLError{Number: 1146, Message: "Table 'items' doesn't exist"})

	count, err := repo.ContarItemsConUnidadCanonica(4)
	if err != nil {
		t.Fatalf("no esperaba error (graceful degradation): %v", err)
	}
	if count != 0 {
		t.Errorf("esperaba count=0, obtuvo %d", count)
	}
}

func TestRepository_ContarItemsConUnidadCanonica_OK(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)

	mock.ExpectQuery(`SELECT COUNT`).WithArgs(uint64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := repo.ContarItemsConUnidadCanonica(4)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if count != 5 {
		t.Errorf("esperaba count=5, obtuvo %d", count)
	}
}

// --- ContarUnidadesActivasPorTipo ---

func TestRepository_ContarUnidadesActivasPorTipo(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(1\) FROM unidades_medida WHERE tipo_medida`).
		WithArgs("peso", uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.ContarUnidadesActivasPorTipo("peso", 1)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if count != 3 {
		t.Errorf("esperaba count=3, obtuvo %d", count)
	}
}

// --- ObtenerPorIDConItems ---

func TestRepository_ObtenerPorIDConItems_OK(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	u := umEjemplo()

	// ObtenerPorID
	expectObtenerPorID(mock, u)
	// ContarItemsConUnidadCanonica
	mock.ExpectQuery(`SELECT COUNT`).WithArgs(u.ID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	resp, err := repo.ObtenerPorIDConItems(u.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.ItemsConUnidadCanonica != 3 {
		t.Errorf("items esperados 3, obtenido %d", resp.ItemsConUnidadCanonica)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_ObtenerPorIDConItems_NoExiste(t *testing.T) {
	db, mock := newMockDB(t)
	repo := um.NewRepository(db)
	mock.ExpectQuery(`SELECT`).WithArgs(uint64(999)).WillReturnError(sql.ErrNoRows)

	_, err := repo.ObtenerPorIDConItems(999)
	if !errors.Is(err, um.ErrUnidadNoEncontrada) {
		t.Errorf("esperaba ErrUnidadNoEncontrada, obtuvo: %v", err)
	}
}

// Silencia "declared but not used" para helpers de tiempo en tests futuros.
var _ = time.Now
