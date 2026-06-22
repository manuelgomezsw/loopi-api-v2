package categorias_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	cat "github.com/manuelgomezsw/loopi-api-v2/internal/categorias"
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

var catCols = []string{"id", "nombre", "activo", "creado_por", "creado_en", "actualizado_por", "actualizado_en"}
var subcatCols = []string{"id", "nombre", "categoria_id", "activo", "creado_por", "creado_en", "actualizado_por", "actualizado_en"}

// --- InsertarCategoria ---

func TestRepoInsertarCategoria_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectExec(`INSERT INTO categorias`).
		WithArgs("Lácteo", uint64(1), sqlmock.AnyArg(), uint64(1), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(5, 1))

	mock.ExpectQuery(`SELECT .* FROM categorias WHERE id = ?`).
		WithArgs(uint64(5)).
		WillReturnRows(sqlmock.NewRows(catCols).
			AddRow(5, "Lácteo", 1, 1, now, 1, now))

	repo := cat.NewRepository(db)
	c, err := repo.InsertarCategoria("Lácteo", 1)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if c.ID != 5 {
		t.Errorf("id esperado 5, obtuvo %d", c.ID)
	}
	if !c.Activo {
		t.Error("esperaba activo=true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestRepoInsertarCategoria_Error1062_Duplicado(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec(`INSERT INTO categorias`).
		WithArgs("Lácteo", uint64(1), sqlmock.AnyArg(), uint64(1), sqlmock.AnyArg()).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry"})

	repo := cat.NewRepository(db)
	_, err := repo.InsertarCategoria("Lácteo", 1)
	if !errors.Is(err, cat.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

// --- ObtenerCategoriaPorID ---

func TestRepoObtenerCategoriaPorID_NoExiste(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT .* FROM categorias WHERE id = ?`).
		WithArgs(uint64(999)).
		WillReturnRows(sqlmock.NewRows(catCols))

	repo := cat.NewRepository(db)
	_, err := repo.ObtenerCategoriaPorID(999)
	if !errors.Is(err, cat.ErrCategoriaNoEncontrada) {
		t.Errorf("esperaba ErrCategoriaNoEncontrada, obtuvo: %v", err)
	}
	_ = mock.ExpectationsWereMet()
}

// --- InsertarSubcategoria ---

func TestRepoInsertarSubcategoria_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectExec(`INSERT INTO subcategorias`).
		WithArgs("Quesos", uint64(1), uint64(42), sqlmock.AnyArg(), uint64(42), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(3, 1))

	mock.ExpectQuery(`SELECT .* FROM subcategorias WHERE id = ?`).
		WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows(subcatCols).
			AddRow(3, "Quesos", 1, 1, 42, now, 42, now))

	repo := cat.NewRepository(db)
	s, err := repo.InsertarSubcategoria("Quesos", 1, 42)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if s.ID != 3 {
		t.Errorf("id esperado 3, obtuvo %d", s.ID)
	}
	_ = mock.ExpectationsWereMet()
}

func TestRepoInsertarSubcategoria_Error1062_Duplicado(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec(`INSERT INTO subcategorias`).
		WithArgs("quesos", uint64(1), uint64(42), sqlmock.AnyArg(), uint64(42), sqlmock.AnyArg()).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry"})

	repo := cat.NewRepository(db)
	_, err := repo.InsertarSubcategoria("quesos", 1, 42)
	if !errors.Is(err, cat.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

// --- ObtenerSubcategoriaPorID ---

func TestRepoObtenerSubcategoriaPorID_NoExiste(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT .* FROM subcategorias WHERE id = ?`).
		WithArgs(uint64(999)).
		WillReturnRows(sqlmock.NewRows(subcatCols))

	repo := cat.NewRepository(db)
	_, err := repo.ObtenerSubcategoriaPorID(999)
	if !errors.Is(err, cat.ErrSubcategoriaNoEncontrada) {
		t.Errorf("esperaba ErrSubcategoriaNoEncontrada, obtuvo: %v", err)
	}
	_ = mock.ExpectationsWereMet()
}

// --- ActualizarCategoria ---

func TestRepoActualizarCategoria_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	now := time.Now()

	mock.ExpectExec(`UPDATE categorias SET nombre`).
		WithArgs("Lácteo Actualizado", uint64(42), sqlmock.AnyArg(), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`SELECT .* FROM categorias WHERE id = ?`).
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(catCols).
			AddRow(1, "Lácteo Actualizado", 1, 42, now, 42, now))

	repo := cat.NewRepository(db)
	c, err := repo.ActualizarCategoria(1, "Lácteo Actualizado", 42)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if c.Nombre != "Lácteo Actualizado" {
		t.Errorf("nombre esperado 'Lácteo Actualizado', obtuvo %q", c.Nombre)
	}
	_ = mock.ExpectationsWereMet()
}

// --- InactivarCategoria ---

func TestRepoInactivarCategoria_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE categorias SET activo = 0`).
		WithArgs(uint64(42), sqlmock.AnyArg(), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := cat.NewRepository(db)
	err := repo.InactivarCategoria(1, 42)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	_ = mock.ExpectationsWereMet()
}

// --- ContarSubcategoriasActivas ---

func TestRepoContarSubcategoriasActivas(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM subcategorias WHERE categoria_id = \? AND activo = 1`).
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	repo := cat.NewRepository(db)
	n, err := repo.ContarSubcategoriasActivas(1)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if n != 3 {
		t.Errorf("esperaba 3, obtuvo %d", n)
	}
	_ = mock.ExpectationsWereMet()
}

