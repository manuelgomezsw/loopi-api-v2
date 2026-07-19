package empleados_test

import (
	"context"
	"database/sql"

	"github.com/manuelgomezsw/loopi-api-v2/internal/empleados"
)

// mockRepo implementa empleados.EmpleadoRepository para tests.
type mockRepo struct {
	obtenerPorIDFn                  func(ctx context.Context, id uint64) (*empleados.Empleado, error)
	existePorUsuarioFn              func(ctx context.Context, usuario string) (bool, error)
	insertarEmpleadoFn              func(ctx context.Context, tx *sql.Tx, emp empleados.Empleado, hash string) (uint64, error)
	actualizarEmpleadoFn            func(ctx context.Context, tx *sql.Tx, id uint64, emp empleados.Empleado) error
	actualizarActivoFn              func(ctx context.Context, tx *sql.Tx, id uint64, activo bool) error
	contarAdminsActivosExcluyendoFn func(ctx context.Context, tx *sql.Tx, id uint64) (int, error)
	actualizarContrasenaFn          func(ctx context.Context, id uint64, hash string) error
	marcarCambioCompletadoFn        func(ctx context.Context, id uint64) error
	listarEmpleadosFn               func(ctx context.Context, p empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error)
	obtenerTiendaActivaPorIDFn      func(ctx context.Context, id uint64) error
	registrarLogFn                  func(ctx context.Context, tx *sql.Tx, actorID, empleadoID uint64, accion string, detalle map[string]any) error
	beginTxFn                       func(ctx context.Context) (*sql.Tx, error)
}

func (m *mockRepo) ObtenerPorID(ctx context.Context, id uint64) (*empleados.Empleado, error) {
	return m.obtenerPorIDFn(ctx, id)
}
func (m *mockRepo) ExistePorUsuario(ctx context.Context, usuario string) (bool, error) {
	return m.existePorUsuarioFn(ctx, usuario)
}
func (m *mockRepo) InsertarEmpleado(ctx context.Context, tx *sql.Tx, emp empleados.Empleado, hash string) (uint64, error) {
	return m.insertarEmpleadoFn(ctx, tx, emp, hash)
}
func (m *mockRepo) ActualizarEmpleado(ctx context.Context, tx *sql.Tx, id uint64, emp empleados.Empleado) error {
	return m.actualizarEmpleadoFn(ctx, tx, id, emp)
}
func (m *mockRepo) ActualizarActivo(ctx context.Context, tx *sql.Tx, id uint64, activo bool) error {
	return m.actualizarActivoFn(ctx, tx, id, activo)
}
func (m *mockRepo) ContarAdminsActivosExcluyendo(ctx context.Context, tx *sql.Tx, id uint64) (int, error) {
	return m.contarAdminsActivosExcluyendoFn(ctx, tx, id)
}
func (m *mockRepo) ActualizarContrasena(ctx context.Context, id uint64, hash string) error {
	return m.actualizarContrasenaFn(ctx, id, hash)
}
func (m *mockRepo) MarcarCambioCompletado(ctx context.Context, id uint64) error {
	return m.marcarCambioCompletadoFn(ctx, id)
}
func (m *mockRepo) ListarEmpleados(ctx context.Context, p empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
	return m.listarEmpleadosFn(ctx, p)
}
func (m *mockRepo) ObtenerTiendaActivaPorID(ctx context.Context, id uint64) error {
	return m.obtenerTiendaActivaPorIDFn(ctx, id)
}
func (m *mockRepo) RegistrarLog(ctx context.Context, tx *sql.Tx, actorID, empleadoID uint64, accion string, detalle map[string]any) error {
	if m.registrarLogFn != nil {
		return m.registrarLogFn(ctx, tx, actorID, empleadoID, accion, detalle)
	}
	return nil
}
func (m *mockRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if m.beginTxFn != nil {
		return m.beginTxFn(ctx)
	}
	return nil, nil
}

// empleadoBarista retorna un empleado barista de prueba.
func empleadoBarista() *empleados.Empleado {
	tid := uint64(1)
	return &empleados.Empleado{
		ID:       10,
		Nombre:   "Ana",
		Apellido: "Gómez",
		Usuario:  "ana.gomez",
		Rol:      "barista",
		TiendaID: &tid,
		Activo:   true,
	}
}

// empleadoAdmin retorna un empleado admin de prueba.
func empleadoAdmin() *empleados.Empleado {
	return &empleados.Empleado{
		ID:       1,
		Nombre:   "Super",
		Apellido: "Admin",
		Usuario:  "admin",
		Rol:      "admin",
		Activo:   true,
	}
}

// noopBeginTx devuelve nil — el service usa rollbackTx/commitTx que son nil-safe.
func noopBeginTx(_ context.Context) (*sql.Tx, error) { return nil, nil }
