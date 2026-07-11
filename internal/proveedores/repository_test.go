package proveedores_test

import (
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	mysql "github.com/go-sql-driver/mysql"
	pv "github.com/manuelgomezsw/loopi-api-v2/internal/proveedores"
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creando sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func pvCols() []string {
	return []string{"id", "razon_social", "nit", "nombre_contacto", "telefono_contacto", "email_contacto", "activo", "creado_en", "actualizado_en"}
}

func addPvRow(rows *sqlmock.Rows, p *pv.Proveedor) *sqlmock.Rows {
	ac := 0
	if p.Activo {
		ac = 1
	}
	return rows.AddRow(p.ID, p.RazonSocial, p.NIT, p.NombreContacto, p.TelefonoContacto, p.EmailContacto, ac, p.CreadoEn, p.ActualizadoEn)
}

func expectObtenerPorID(mock sqlmock.Sqlmock, p *pv.Proveedor) {
	rows := addPvRow(sqlmock.NewRows(pvCols()), p)
	mock.ExpectQuery(`SELECT`).WithArgs(p.ID).WillReturnRows(rows)
}

// --- Crear ---

func TestRepository_Crear_Exitoso(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)
	p := proveedorEjemplo()

	mock.ExpectExec(`INSERT INTO proveedores`).
		WithArgs(p.RazonSocial, p.NIT, p.NombreContacto, p.TelefonoContacto, p.EmailContacto, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectObtenerPorID(mock, p)

	resultado, err := repo.Crear(&pv.CrearProveedorRequest{RazonSocial: p.RazonSocial, NIT: p.NIT})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.NIT != p.NIT {
		t.Errorf("nit esperado %q, obtenido %q", p.NIT, resultado.NIT)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas no cumplidas: %v", err)
	}
}

func TestRepository_Crear_NITDuplicado(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)

	mock.ExpectExec(`INSERT INTO proveedores`).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry"})

	_, err := repo.Crear(&pv.CrearProveedorRequest{RazonSocial: "X", NIT: "900123456-7"})
	if !errors.Is(err, pv.ErrNITDuplicado) {
		t.Errorf("esperaba ErrNITDuplicado, obtuvo: %v", err)
	}
}

// --- ObtenerPorID ---

func TestRepository_ObtenerPorID_Encontrado(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)
	p := proveedorEjemplo()
	expectObtenerPorID(mock, p)

	resultado, err := repo.ObtenerPorID(p.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.RazonSocial != p.RazonSocial {
		t.Errorf("razon_social esperada %q, obtenida %q", p.RazonSocial, resultado.RazonSocial)
	}
}

func TestRepository_ObtenerPorID_NoEncontrado(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)
	mock.ExpectQuery(`SELECT`).WithArgs(uint64(999)).WillReturnError(sql.ErrNoRows)

	_, err := repo.ObtenerPorID(999)
	if !errors.Is(err, pv.ErrProveedorNoEncontrado) {
		t.Errorf("esperaba ErrProveedorNoEncontrado, obtuvo: %v", err)
	}
}

// --- ObtenerPorIDConItems ---

func TestRepository_ObtenerPorIDConItems(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)
	p := proveedorEjemplo()
	expectObtenerPorID(mock, p)
	mock.ExpectQuery(`SELECT COUNT\(1\) FROM items`).WithArgs(p.ID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	detalle, err := repo.ObtenerPorIDConItems(p.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if detalle.ItemsAsignados != 3 {
		t.Errorf("items_asignados esperado 3, obtenido %d", detalle.ItemsAsignados)
	}
}

// --- ExisteNIT ---

func TestRepository_ExisteNIT_Existe(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)
	mock.ExpectQuery(`SELECT COUNT\(1\) FROM proveedores WHERE nit = \?`).
		WithArgs("900123456-7").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	existe, err := repo.ExisteNIT("900123456-7", nil)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !existe {
		t.Error("esperaba existe=true")
	}
}

func TestRepository_ExisteNIT_ExcludeID(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)
	excludeID := uint64(1)
	mock.ExpectQuery(`SELECT COUNT\(1\) FROM proveedores WHERE nit = \? AND id != \?`).
		WithArgs("900123456-7", excludeID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	existe, err := repo.ExisteNIT("900123456-7", &excludeID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if existe {
		t.Error("esperaba existe=false")
	}
}

// --- Actualizar ---

func TestRepository_Actualizar(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)
	p := proveedorEjemplo()

	mock.ExpectExec(`UPDATE proveedores SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectObtenerPorID(mock, p)

	nombre := "María López"
	_, err := repo.Actualizar(p.ID, &pv.EditarProveedorRequest{NombreContacto: &nombre})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
}

// --- CambiarEstado ---

func TestRepository_CambiarEstado_Inactivar(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)

	mock.ExpectExec(`UPDATE proveedores SET activo = \?`).
		WithArgs(0, sqlmock.AnyArg(), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.CambiarEstado(1, false); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
}

// --- ContarItemsAsignados ---

func TestRepository_ContarItemsAsignados_TablaNoExiste(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(1\) FROM items`).
		WithArgs(uint64(1)).
		WillReturnError(&mysql.MySQLError{Number: 1146, Message: "Table doesn't exist"})

	count, err := repo.ContarItemsAsignados(1)
	if err != nil {
		t.Fatalf("no esperaba error (graceful degradation): %v", err)
	}
	if count != 0 {
		t.Errorf("esperaba count=0, obtuvo %d", count)
	}
}

// --- Listar ---

func TestRepository_Listar_ConFiltros(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)
	p := proveedorEjemplo()

	mock.ExpectQuery(`SELECT COUNT\(1\) FROM proveedores`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	rows := addPvRow(sqlmock.NewRows(pvCols()), p)
	mock.ExpectQuery(`SELECT id, razon_social`).WillReturnRows(rows)

	activo := true
	resp, err := repo.Listar(&pv.FiltrosListado{Activo: &activo, Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 1 || len(resp.Proveedores) != 1 {
		t.Errorf("esperaba 1 proveedor, obtuvo total=%d len=%d", resp.Total, len(resp.Proveedores))
	}
}

func TestRepository_Listar_SinResultados(t *testing.T) {
	db, mock := newMockDB(t)
	repo := pv.NewRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(1\) FROM proveedores`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT id, razon_social`).WillReturnRows(sqlmock.NewRows(pvCols()))

	resp, err := repo.Listar(&pv.FiltrosListado{Busqueda: "noexiste", Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 0 || len(resp.Proveedores) != 0 {
		t.Errorf("esperaba 0 proveedores, obtuvo total=%d len=%d", resp.Total, len(resp.Proveedores))
	}
}
