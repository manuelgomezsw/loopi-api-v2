package auth_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
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

// --- InsertTokenRevocado ---

func TestRepository_InsertTokenRevocado_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	jti := "test-jti-insert"
	expira := time.Now().UTC().Add(1 * time.Hour)

	mock.ExpectExec(`INSERT INTO tokens_revocados`).
		WithArgs(jti, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.InsertTokenRevocado(jti, expira); err != nil {
		t.Errorf("no esperaba error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_InsertTokenRevocado_ErrorBD(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	errBD := errors.New("fallo de BD")
	mock.ExpectExec(`INSERT INTO tokens_revocados`).
		WillReturnError(errBD)

	err := repo.InsertTokenRevocado("jti-err", time.Now().UTC())
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
}

// --- ExisteTokenRevocado ---

func TestRepository_ExisteTokenRevocado_Existe(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT COUNT\(1\) FROM tokens_revocados`).
		WithArgs("jti-existente").
		WillReturnRows(rows)

	existe, err := repo.ExisteTokenRevocado("jti-existente")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !existe {
		t.Error("esperaba que el token existiera")
	}
}

func TestRepository_ExisteTokenRevocado_NoExiste(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT\(1\) FROM tokens_revocados`).
		WithArgs("jti-nuevo").
		WillReturnRows(rows)

	existe, err := repo.ExisteTokenRevocado("jti-nuevo")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if existe {
		t.Error("no esperaba que el token existiera")
	}
}

func TestRepository_ExisteTokenRevocado_ErrorBD(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	errBD := errors.New("timeout")
	mock.ExpectQuery(`SELECT COUNT\(1\) FROM tokens_revocados`).
		WillReturnError(errBD)

	_, err := repo.ExisteTokenRevocado("jti-err")
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
}

// --- LimpiarTokensExpirados ---

func TestRepository_LimpiarTokensExpirados_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	mock.ExpectExec(`DELETE FROM tokens_revocados`).
		WillReturnResult(sqlmock.NewResult(0, 3)) // 3 filas eliminadas

	eliminados, err := repo.LimpiarTokensExpirados()
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if eliminados != 3 {
		t.Errorf("esperaba 3 eliminados, obtuvo %d", eliminados)
	}
}

func TestRepository_LimpiarTokensExpirados_ErrorBD(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	mock.ExpectExec(`DELETE FROM tokens_revocados`).
		WillReturnError(errors.New("fallo"))

	_, err := repo.LimpiarTokensExpirados()
	if err == nil {
		t.Error("esperaba error")
	}
}

// --- BuscarUsuarioPorNombre ---

func TestRepository_BuscarUsuarioPorNombre_Encontrado(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	tiendaID := 5
	rows := sqlmock.NewRows([]string{
		"id", "contrasena_hash", "rol", "tienda_id", "activo", "bloqueado_hasta", "intentos_fallidos",
	}).AddRow(42, "hash-bcrypt", "barista", tiendaID, true, nil, 0)

	mock.ExpectQuery(`SELECT id, contrasena_hash, rol, tienda_id, activo, bloqueado_hasta, intentos_fallidos`).
		WithArgs("juan").
		WillReturnRows(rows)

	u, err := repo.BuscarUsuarioPorNombre("juan")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if u.ID != 42 {
		t.Errorf("ID esperado 42, obtenido %d", u.ID)
	}
	if u.Rol != "barista" {
		t.Errorf("rol esperado 'barista', obtenido %q", u.Rol)
	}
	if u.TiendaID == nil || *u.TiendaID != 5 {
		t.Error("tienda_id incorrecto")
	}
}

func TestRepository_BuscarUsuarioPorNombre_NoExiste(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	mock.ExpectQuery(`SELECT id, contrasena_hash, rol, tienda_id, activo, bloqueado_hasta, intentos_fallidos`).
		WithArgs("fantasma").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.BuscarUsuarioPorNombre("fantasma")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("esperaba sql.ErrNoRows, obtuvo: %v", err)
	}
}

// --- IncrementarIntentosFallidos ---

func TestRepository_IncrementarIntentosFallidos_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	mock.ExpectExec(`UPDATE usuarios SET intentos_fallidos`).
		WithArgs(3, 99).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.IncrementarIntentosFallidos(99, 3); err != nil {
		t.Errorf("no esperaba error: %v", err)
	}
}

func TestRepository_IncrementarIntentosFallidos_ErrorBD(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	mock.ExpectExec(`UPDATE usuarios SET intentos_fallidos`).
		WillReturnError(errors.New("fallo"))

	if err := repo.IncrementarIntentosFallidos(1, 1); err == nil {
		t.Error("esperaba error")
	}
}

// --- BloquearUsuario ---

func TestRepository_BloquearUsuario_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	hasta := time.Now().UTC().Add(5 * time.Minute)
	mock.ExpectExec(`UPDATE usuarios SET intentos_fallidos = 0, bloqueado_hasta`).
		WithArgs(sqlmock.AnyArg(), 7).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.BloquearUsuario(7, hasta); err != nil {
		t.Errorf("no esperaba error: %v", err)
	}
}

func TestRepository_BloquearUsuario_ErrorBD(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	mock.ExpectExec(`UPDATE usuarios SET intentos_fallidos = 0, bloqueado_hasta`).
		WillReturnError(errors.New("fallo"))

	if err := repo.BloquearUsuario(1, time.Now().UTC()); err == nil {
		t.Error("esperaba error")
	}
}

// --- ResetearIntentosLogin ---

func TestRepository_ResetearIntentosLogin_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	mock.ExpectExec(`UPDATE usuarios SET intentos_fallidos = 0, bloqueado_hasta = NULL`).
		WithArgs(12).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.ResetearIntentosLogin(12); err != nil {
		t.Errorf("no esperaba error: %v", err)
	}
}

func TestRepository_ResetearIntentosLogin_ErrorBD(t *testing.T) {
	db, mock := newMockDB(t)
	repo := auth.NewRepository(db)

	mock.ExpectExec(`UPDATE usuarios SET intentos_fallidos = 0, bloqueado_hasta = NULL`).
		WillReturnError(errors.New("fallo"))

	if err := repo.ResetearIntentosLogin(1); err == nil {
		t.Error("esperaba error")
	}
}
