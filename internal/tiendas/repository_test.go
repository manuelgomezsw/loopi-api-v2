package tiendas_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/manuelgomezsw/loopi-api-v2/internal/tiendas"
)

// newMockDB crea un *sql.DB con sqlmock para cada test.
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creando sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

// tiendaRows devuelve sqlmock.Rows con las columnas de la tabla tiendas.
func tiendaRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "codigo", "nombre", "direccion", "ciudad", "telefono", "activo",
		"creado_por", "creado_en", "actualizado_por", "actualizado_en",
	})
}

// addTiendaRow añade una fila de tienda a los rows de sqlmock.
func addTiendaRow(rows *sqlmock.Rows, t tiendas.Tienda) *sqlmock.Rows {
	activo := 0
	if t.Activo {
		activo = 1
	}
	return rows.AddRow(
		t.ID, t.Codigo, t.Nombre, t.Direccion, t.Ciudad, t.Telefono, activo,
		t.CreadoPor, t.CreadoEn, t.ActualizadoPor, t.ActualizadoEn,
	)
}

// expectObtenerPorID registra la expectativa de SELECT por ID en sqlmock.
func expectObtenerPorID(mock sqlmock.Sqlmock, t tiendas.Tienda) {
	rows := addTiendaRow(tiendaRows(), t)
	mock.ExpectQuery(`SELECT id, codigo`).
		WithArgs(t.ID).
		WillReturnRows(rows)
}

// --- ObtenerPorID ---

func TestRepository_ObtenerPorID_Encontrada(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	ejemplo := tiendaEjemplo()

	expectObtenerPorID(mock, ejemplo)

	resultado, err := repo.ObtenerPorID(ejemplo.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.Codigo != ejemplo.Codigo {
		t.Errorf("codigo esperado %q, obtenido %q", ejemplo.Codigo, resultado.Codigo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_ObtenerPorID_NoExiste_RetornaErrNoEncontrada(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)

	mock.ExpectQuery(`SELECT id, codigo`).
		WithArgs(uint64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.ObtenerPorID(999)
	if !errors.Is(err, tiendas.ErrTiendaNoEncontrada) {
		t.Errorf("esperaba ErrTiendaNoEncontrada, obtuvo: %v", err)
	}
}

func TestRepository_ObtenerPorID_ErrorBD_Propaga(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	errBD := errors.New("fallo de BD")

	mock.ExpectQuery(`SELECT id, codigo`).
		WithArgs(uint64(1)).
		WillReturnError(errBD)

	_, err := repo.ObtenerPorID(1)
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
}

// --- Crear ---

func TestRepository_Crear_OK_RetornaTiendaConID(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	ahora := time.Now()
	nueva := tiendas.Tienda{
		Codigo: "TDA-001", Nombre: "Norte", Direccion: "Calle 1",
		Ciudad: "Bogotá", Telefono: "300", CreadoPor: 1, CreadoEn: ahora,
		ActualizadoPor: 1, ActualizadoEn: ahora,
	}

	mock.ExpectExec(`INSERT INTO tiendas`).
		WithArgs(nueva.Codigo, nueva.Nombre, nueva.Direccion, nueva.Ciudad, nueva.Telefono,
			nueva.CreadoPor, nueva.CreadoEn, nueva.ActualizadoPor, nueva.ActualizadoEn).
		WillReturnResult(sqlmock.NewResult(5, 1))

	nueva.ID = 5
	nueva.Activo = true
	expectObtenerPorID(mock, nueva)

	resultado, err := repo.Crear(nueva)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.ID != 5 {
		t.Errorf("id esperado 5, obtenido %d", resultado.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_Crear_NombreDuplicado_RetornaErrNombre(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)

	// Error MySQL 1062 con constraint uq_tiendas_nombre
	mock.ExpectExec(`INSERT INTO tiendas`).
		WillReturnError(dupError1062("uq_tiendas_nombre"))

	_, err := repo.Crear(tiendas.Tienda{Codigo: "X", Nombre: "Existe"})
	if !errors.Is(err, tiendas.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

func TestRepository_Crear_CodigoDuplicado_RetornaErrCodigo(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)

	mock.ExpectExec(`INSERT INTO tiendas`).
		WillReturnError(dupError1062("uq_tiendas_codigo"))

	_, err := repo.Crear(tiendas.Tienda{Codigo: "DUP"})
	if !errors.Is(err, tiendas.ErrCodigoDuplicado) {
		t.Errorf("esperaba ErrCodigoDuplicado, obtuvo: %v", err)
	}
}

// --- Listar ---

func TestRepository_Listar_Todas_RetornaFilas(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	ejemplo := tiendaEjemplo()

	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := addTiendaRow(tiendaRows(), ejemplo)
	mock.ExpectQuery(`SELECT id, codigo`).
		WithArgs(50, 0).
		WillReturnRows(rows)

	lista, total, err := repo.Listar("todas", 1, 50)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if total != 1 {
		t.Errorf("total esperado 1, obtenido %d", total)
	}
	if len(lista) != 1 {
		t.Errorf("esperaba 1 elemento, obtuvo %d", len(lista))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_Listar_Activas_UsaFiltroActivo1(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(1\) FROM tiendas WHERE activo = 1`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`FROM tiendas WHERE activo = 1`).
		WithArgs(50, 0).
		WillReturnRows(tiendaRows())

	lista, total, err := repo.Listar("activas", 1, 50)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if total != 0 {
		t.Errorf("total esperado 0, obtenido %d", total)
	}
	if len(lista) != 0 {
		t.Errorf("esperaba lista vacía, obtuvo %d elementos", len(lista))
	}
}

func TestRepository_Listar_Inactivas_UsaFiltroActivo0(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(1\) FROM tiendas WHERE activo = 0`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`FROM tiendas WHERE activo = 0`).
		WithArgs(50, 0).
		WillReturnRows(tiendaRows())

	_, _, err := repo.Listar("inactivas", 1, 50)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_Listar_ErrorEnCount_Propaga(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	errBD := errors.New("fallo count")

	mock.ExpectQuery(`SELECT COUNT`).WillReturnError(errBD)

	_, _, err := repo.Listar("todas", 1, 50)
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
}

// --- Actualizar ---

func TestRepository_Actualizar_OK_RetornaTiendaActualizada(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	ahora := time.Now()
	ejemplo := tiendaEjemplo()

	mock.ExpectExec(`UPDATE tiendas`).
		WithArgs("Nuevo nombre", ejemplo.Direccion, ejemplo.Ciudad, ejemplo.Telefono,
			uint64(42), ahora, ejemplo.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ejemplo.Nombre = "Nuevo nombre"
	expectObtenerPorID(mock, ejemplo)

	req := tiendas.TiendaUpdateRequest{
		Nombre: "Nuevo nombre", Direccion: ejemplo.Direccion,
		Ciudad: ejemplo.Ciudad, Telefono: ejemplo.Telefono,
	}
	resultado, err := repo.Actualizar(ejemplo.ID, req, 42, ahora)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.Nombre != "Nuevo nombre" {
		t.Errorf("nombre esperado 'Nuevo nombre', obtenido %q", resultado.Nombre)
	}
	if resultado.Codigo != "TDA-001" {
		t.Errorf("codigo no debe cambiar, obtenido %q", resultado.Codigo)
	}
}

func TestRepository_Actualizar_NombreDuplicado_RetornaErrNombre(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)

	mock.ExpectExec(`UPDATE tiendas`).
		WillReturnError(dupError1062("uq_tiendas_nombre"))

	_, err := repo.Actualizar(1, tiendas.TiendaUpdateRequest{Nombre: "Dup"}, 1, time.Now())
	if !errors.Is(err, tiendas.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

// --- CambiarActivo ---

func TestRepository_CambiarActivo_Inactivar_OK(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	ahora := time.Now()
	ejemplo := tiendaEjemplo()
	ejemplo.Activo = false

	mock.ExpectExec(`UPDATE tiendas`).
		WithArgs(0, uint64(42), ahora, ejemplo.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	expectObtenerPorID(mock, ejemplo)

	resultado, err := repo.CambiarActivo(ejemplo.ID, false, 42, ahora)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.Activo {
		t.Error("activo debe ser false tras inactivar")
	}
}

func TestRepository_CambiarActivo_Reactivar_OK(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	ahora := time.Now()
	ejemplo := tiendaEjemplo()
	ejemplo.Activo = true

	mock.ExpectExec(`UPDATE tiendas`).
		WithArgs(1, uint64(42), ahora, ejemplo.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	expectObtenerPorID(mock, ejemplo)

	resultado, err := repo.CambiarActivo(ejemplo.ID, true, 42, ahora)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !resultado.Activo {
		t.Error("activo debe ser true tras reactivar")
	}
}

func TestRepository_Listar_ErrorEnRows_Propaga(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	errRows := errors.New("fallo en scan")

	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Rows que fallan al escanear (columna extra fuerza error)
	rows := sqlmock.NewRows([]string{"id"}).AddRow(nil) // nil en campo NOT NULL fuerza scan error
	mock.ExpectQuery(`SELECT id, codigo`).
		WithArgs(50, 0).
		WillReturnRows(rows)

	_ = errRows // silenciar linter
	_, _, err := repo.Listar("todas", 1, 50)
	if err == nil {
		t.Error("esperaba error al escanear fila inválida")
	}
}

func TestRepository_CambiarActivo_ErrorBD_Propaga(t *testing.T) {
	db, mock := newMockDB(t)
	repo := tiendas.NewRepository(db)
	errBD := errors.New("fallo update")

	mock.ExpectExec(`UPDATE tiendas`).WillReturnError(errBD)

	_, err := repo.CambiarActivo(1, false, 1, time.Now())
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
}

// --- helpers de error MySQL ---

// dupError1062 retorna un *mysql.MySQLError que simula el error 1062 Duplicate entry
// para el constraint indicado, compatible con errors.As en mapMySQLError.
func dupError1062(constraint string) *mysql.MySQLError {
	return &mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry 'x' for key '" + constraint + "'",
	}
}
