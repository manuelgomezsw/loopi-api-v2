package empleados_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/manuelgomezsw/loopi-api-v2/config"
	"github.com/manuelgomezsw/loopi-api-v2/internal/empleados"
)

// repoBase construye un mockRepo con stubs seguros por defecto (el test sobreescribe lo que necesita).
func repoBase() *mockRepo {
	return &mockRepo{
		beginTxFn:          noopBeginTx,
		registrarLogFn:     func(_ context.Context, _ *sql.Tx, _, _ uint64, _ string, _ map[string]any) error { return nil },
		obtenerPorIDFn:     func(_ context.Context, id uint64) (*empleados.Empleado, error) { return empleadoBarista(), nil },
		existePorUsuarioFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
		obtenerTiendaActivaPorIDFn: func(_ context.Context, _ uint64) error { return nil },
		insertarEmpleadoFn: func(_ context.Context, _ *sql.Tx, _ empleados.Empleado, _ string) (uint64, error) {
			return 10, nil
		},
		actualizarEmpleadoFn:            func(_ context.Context, _ *sql.Tx, _ uint64, _ empleados.Empleado) error { return nil },
		actualizarActivoFn:              func(_ context.Context, _ *sql.Tx, _ uint64, _ bool) error { return nil },
		contarAdminsActivosExcluyendoFn: func(_ context.Context, _ *sql.Tx, _ uint64) (int, error) { return 1, nil },
		actualizarContrasenaFn:          func(_ context.Context, _ uint64, _ string) error { return nil },
		marcarCambioCompletadoFn:        func(_ context.Context, _ uint64) error { return nil },
		listarEmpleadosFn: func(_ context.Context, _ empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
			return []empleados.Empleado{*empleadoBarista()}, 1, nil
		},
	}
}

func newSvc(r empleados.EmpleadoRepository) empleados.EmpleadoService {
	return empleados.NewServiceWithCost(r, config.BcryptCostTests)
}

// --- Casos negativos ---

func TestCrearEmpleadoBaristaSinTienda(t *testing.T) {
	svc := newSvc(repoBase())
	req := empleados.CrearEmpleadoRequest{
		Nombre:   "Ana",
		Apellido: "Gómez",
		Usuario:  "ana.gomez",
		Rol:      "barista",
		// TiendaID ausente
	}
	_, err := svc.CrearEmpleado(context.Background(), 1, req)
	if err == nil {
		t.Fatal("esperaba error tienda_requerida, no lo hubo")
	}
	var valErr *empleados.ValidationError
	if !errors.As(err, &valErr) || valErr.Codigo != "tienda_requerida" {
		t.Fatalf("error esperado tienda_requerida, obtenido: %v", err)
	}
}

func TestCrearEmpleadoUsuarioDuplicado(t *testing.T) {
	r := repoBase()
	r.existePorUsuarioFn = func(_ context.Context, _ string) (bool, error) { return true, nil }
	svc := newSvc(r)
	tid := uint64(1)
	req := empleados.CrearEmpleadoRequest{
		Nombre:   "Ana",
		Apellido: "Gómez",
		Usuario:  "ana.gomez",
		Rol:      "barista",
		TiendaID: &tid,
	}
	_, err := svc.CrearEmpleado(context.Background(), 1, req)
	if !errors.Is(err, empleados.ErrUsuarioDuplicado) {
		t.Fatalf("error esperado ErrUsuarioDuplicado, obtenido: %v", err)
	}
}

func TestInactivarUltimoAdmin(t *testing.T) {
	r := repoBase()
	r.obtenerPorIDFn = func(_ context.Context, _ uint64) (*empleados.Empleado, error) { return empleadoAdmin(), nil }
	r.contarAdminsActivosExcluyendoFn = func(_ context.Context, _ *sql.Tx, _ uint64) (int, error) { return 0, nil }
	svc := newSvc(r)
	_, err := svc.CambiarEstado(context.Background(), 1, 1, false)
	if !errors.Is(err, empleados.ErrUltimoAdminActivo) {
		t.Fatalf("error esperado ErrUltimoAdminActivo, obtenido: %v", err)
	}
}

func TestResetContrasenaGeneraHashDistinto(t *testing.T) {
	r := repoBase()
	r.obtenerPorIDFn = func(_ context.Context, _ uint64) (*empleados.Empleado, error) { return empleadoAdmin(), nil }
	var hashes []string
	r.actualizarContrasenaFn = func(_ context.Context, _ uint64, hash string) error {
		hashes = append(hashes, hash)
		return nil
	}
	svc := newSvc(r)
	c1, err := svc.ResetearContrasena(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("primer reset falló: %v", err)
	}
	c2, err := svc.ResetearContrasena(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("segundo reset falló: %v", err)
	}
	if c1 == c2 {
		t.Fatal("dos resets consecutivos generaron la misma contraseña temporal")
	}
	if len(hashes) < 2 || hashes[0] == hashes[1] {
		t.Fatal("los hashes bcrypt almacenados deben ser distintos")
	}
}

func TestEditarEmpleadoTiendaInactiva(t *testing.T) {
	r := repoBase()
	r.obtenerTiendaActivaPorIDFn = func(_ context.Context, _ uint64) error { return empleados.ErrTiendaInactiva }
	svc := newSvc(r)
	nuevoRol := "lider_tienda"
	tid := uint64(99)
	req := empleados.EditarEmpleadoRequest{Rol: &nuevoRol, TiendaID: &tid}
	_, err := svc.EditarEmpleado(context.Background(), 1, 10, req)
	var valErr *empleados.ValidationError
	if !errors.As(err, &valErr) || valErr.Codigo != "tienda_no_existe" {
		t.Fatalf("error esperado tienda_no_existe, obtenido: %v", err)
	}
}

func TestCambiarContrasenaMinimo4Chars(t *testing.T) {
	svc := newSvc(repoBase())
	err := svc.CambiarContrasena(context.Background(), 10, "abc")
	var valErr *empleados.ValidationError
	if !errors.As(err, &valErr) || valErr.Codigo != "contrasena_muy_corta" {
		t.Fatalf("error esperado contrasena_muy_corta, obtenido: %v", err)
	}
}

// --- Casos positivos ---

func TestCrearEmpleadoExitoso(t *testing.T) {
	r := repoBase()
	tid := uint64(1)
	req := empleados.CrearEmpleadoRequest{
		Nombre:   "Ana",
		Apellido: "Gómez",
		Usuario:  "ana.gomez",
		Rol:      "barista",
		TiendaID: &tid,
	}
	svc := newSvc(r)
	resp, err := svc.CrearEmpleado(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("creación exitosa esperada, error: %v", err)
	}
	if resp.ContrasenaTemporal == "" {
		t.Fatal("contrasena_temporal debe ser no vacía")
	}
	if resp.ID == 0 {
		t.Fatal("el id del empleado creado debe ser > 0")
	}
	if resp.Activo != true {
		t.Fatal("empleado recién creado debe estar activo")
	}
}

func TestEditarEmpleadoExitoso(t *testing.T) {
	r := repoBase()
	logCalled := false
	r.registrarLogFn = func(_ context.Context, _ *sql.Tx, _, _ uint64, accion string, detalle map[string]any) error {
		if accion == "EDITAR" {
			logCalled = true
			if _, ok := detalle["campos_anteriores"]; !ok {
				return errors.New("detalle debe tener campos_anteriores")
			}
			if _, ok := detalle["campos_nuevos"]; !ok {
				return errors.New("detalle debe tener campos_nuevos")
			}
		}
		return nil
	}
	svc := newSvc(r)
	nuevoNombre := "Ana María"
	resp, err := svc.EditarEmpleado(context.Background(), 1, 10, empleados.EditarEmpleadoRequest{Nombre: &nuevoNombre})
	if err != nil {
		t.Fatalf("edición exitosa esperada, error: %v", err)
	}
	if resp == nil {
		t.Fatal("respuesta no debe ser nil")
	}
	if !logCalled {
		t.Fatal("audit log EDITAR debe registrarse")
	}
}

func TestReactivarEmpleadoExitoso(t *testing.T) {
	r := repoBase()
	emp := empleadoBarista()
	emp.Activo = false
	r.obtenerPorIDFn = func(_ context.Context, _ uint64) (*empleados.Empleado, error) { return emp, nil }
	svc := newSvc(r)
	resp, err := svc.CambiarEstado(context.Background(), 1, 10, true)
	if err != nil {
		t.Fatalf("reactivación exitosa esperada, error: %v", err)
	}
	if resp == nil {
		t.Fatal("respuesta no debe ser nil")
	}
}

func TestListarEmpleadosConFiltros(t *testing.T) {
	r := repoBase()
	r.listarEmpleadosFn = func(_ context.Context, p empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
		if p.Q != "ana" {
			return nil, 0, errors.New("q debe ser 'ana'")
		}
		return []empleados.Empleado{*empleadoBarista()}, 1, nil
	}
	svc := newSvc(r)
	resp, err := svc.ListarEmpleados(context.Background(), empleados.ListarEmpleadosParams{Q: "ana", Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("listar exitoso esperado, error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("total esperado 1, obtenido %d", resp.Total)
	}
	if len(resp.Empleados) != 1 {
		t.Fatalf("esperados 1 empleados, obtenidos %d", len(resp.Empleados))
	}
}

// --- Tests de generador de contraseña ---

func TestGenerarContrasenaTempUnicidad(t *testing.T) {
	// Verificar que 10 resets generan 10 contraseñas distintas.
	r := repoBase()
	r.obtenerPorIDFn = func(_ context.Context, _ uint64) (*empleados.Empleado, error) { return empleadoAdmin(), nil }
	svc := newSvc(r)
	seen := make(map[string]bool)
	for i := range 10 {
		c, err := svc.ResetearContrasena(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("reset %d falló: %v", i, err)
		}
		if len(c) < 12 {
			t.Fatalf("contraseña temporal debe tener al menos 12 chars, tiene %d", len(c))
		}
		if seen[c] {
			t.Fatalf("contraseña duplicada en iteración %d: %s", i, c)
		}
		seen[c] = true
	}
}
