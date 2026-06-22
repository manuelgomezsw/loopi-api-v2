package empleados_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/empleados"
)

func newCachedEmpRepo(t *testing.T, inner empleados.EmpleadoRepository) empleados.EmpleadoRepository {
	t.Helper()
	repo, err := empleados.NewCachedRepository(inner, time.Minute)
	if err != nil {
		t.Fatalf("NewCachedRepository: %v", err)
	}
	return repo
}

// --- ObtenerPorID ---

func TestCachedEmpObtenerPorIDMissLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			llamadas++
			return &empleados.Empleado{ID: id, Nombre: "Ana"}, nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	got, err := repo.ObtenerPorID(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Nombre != "Ana" {
		t.Errorf("esperaba 'Ana', obtuvo '%s'", got.Nombre)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada, obtuvo %d", llamadas)
	}
}

func TestCachedEmpObtenerPorIDHitNoLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			llamadas++
			return &empleados.Empleado{ID: id}, nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	if _, err := repo.ObtenerPorID(ctx, 5); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.ObtenerPorID(ctx, 5); err != nil {
		t.Fatal(err)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada total (hit en segunda), obtuvo %d", llamadas)
	}
}

func TestCachedEmpObtenerPorIDErrorNoCachea(t *testing.T) {
	errDB := errors.New("not found")
	inner := &mockRepo{
		obtenerPorIDFn: func(context.Context, uint64) (*empleados.Empleado, error) {
			return nil, errDB
		},
	}
	repo := newCachedEmpRepo(t, inner)
	_, err := repo.ObtenerPorID(context.Background(), 99)
	if !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
}

// --- ListarEmpleados ---

func TestCachedEmpListarMissLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		listarEmpleadosFn: func(_ context.Context, _ empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
			llamadas++
			return []empleados.Empleado{{ID: 1}}, 1, nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	p := empleados.ListarEmpleadosParams{Page: 1, Limit: 10}
	items, total, err := repo.ListarEmpleados(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 {
		t.Errorf("resultado inesperado: items=%d, total=%d", len(items), total)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada, obtuvo %d", llamadas)
	}
}

func TestCachedEmpListarHitNoLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		listarEmpleadosFn: func(_ context.Context, _ empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
			llamadas++
			return []empleados.Empleado{{ID: 1}}, 1, nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	p := empleados.ListarEmpleadosParams{Page: 1, Limit: 10}
	if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
		t.Fatal(err)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada total (hit en segunda), obtuvo %d", llamadas)
	}
}

// --- InsertarEmpleado invalida lista ---

func TestCachedEmpInsertarInvalidaListaCache(t *testing.T) {
	listarLlamadas := 0
	inner := &mockRepo{
		listarEmpleadosFn: func(_ context.Context, _ empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
			listarLlamadas++
			return []empleados.Empleado{}, 0, nil
		},
		insertarEmpleadoFn: func(_ context.Context, _ *sql.Tx, emp empleados.Empleado, _ string) (uint64, error) {
			return 42, nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	p := empleados.ListarEmpleadosParams{Page: 1, Limit: 10}

	if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if _, err := repo.InsertarEmpleado(ctx, nil, empleados.Empleado{Nombre: "Luis"}, "hash"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas (caché invalidada por Insertar), obtuvo %d", listarLlamadas)
	}
}

// --- ActualizarEmpleado invalida byID y lista ---

func TestCachedEmpActualizarInvalidaCache(t *testing.T) {
	byIDLlamadas := 0
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			byIDLlamadas++
			return &empleados.Empleado{ID: id}, nil
		},
		actualizarEmpleadoFn: func(_ context.Context, _ *sql.Tx, id uint64, _ empleados.Empleado) error {
			return nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()

	if _, err := repo.ObtenerPorID(ctx, 7); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := repo.ActualizarEmpleado(ctx, nil, 7, empleados.Empleado{Nombre: "Nuevo"}); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.ObtenerPorID(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas (caché invalidada por Actualizar), obtuvo %d", byIDLlamadas)
	}
}

// --- Error en escritura no invalida caché ---

func TestCachedEmpActualizarErrorNoInvalidaCache(t *testing.T) {
	byIDLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			byIDLlamadas++
			return &empleados.Empleado{ID: id}, nil
		},
		actualizarEmpleadoFn: func(context.Context, *sql.Tx, uint64, empleados.Empleado) error {
			return errDB
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()

	if _, err := repo.ObtenerPorID(ctx, 8); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := repo.ActualizarEmpleado(ctx, nil, 8, empleados.Empleado{}); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}

	if _, err := repo.ObtenerPorID(ctx, 8); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si Actualizar falló; llamadas: %d", byIDLlamadas)
	}
}

// --- ActualizarActivo invalida byID y lista ---

func TestCachedEmpActualizarActivoInvalidaCache(t *testing.T) {
	listarLlamadas := 0
	inner := &mockRepo{
		listarEmpleadosFn: func(_ context.Context, _ empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
			listarLlamadas++
			return []empleados.Empleado{{ID: 9}}, 1, nil
		},
		actualizarActivoFn: func(context.Context, *sql.Tx, uint64, bool) error {
			return nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	p := empleados.ListarEmpleadosParams{Page: 1, Limit: 10}

	if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := repo.ActualizarActivo(ctx, nil, 9, false); err != nil {
		t.Fatal(err)
	}

	if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas (caché invalidada por ActualizarActivo), obtuvo %d", listarLlamadas)
	}
}

// --- ActualizarContrasena invalida byID ---

func TestCachedEmpActualizarContrasenaInvalidaByID(t *testing.T) {
	byIDLlamadas := 0
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			byIDLlamadas++
			return &empleados.Empleado{ID: id}, nil
		},
		actualizarContrasenaFn: func(context.Context, uint64, string) error {
			return nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()

	if _, err := repo.ObtenerPorID(ctx, 11); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := repo.ActualizarContrasena(ctx, 11, "newhash"); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.ObtenerPorID(ctx, 11); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas (byID invalidado por ActualizarContrasena), obtuvo %d", byIDLlamadas)
	}
}

// --- Passthrough methods ---

func TestCachedEmpExistePorUsuarioDelegaInner(t *testing.T) {
	inner := &mockRepo{
		existePorUsuarioFn: func(_ context.Context, usuario string) (bool, error) {
			return usuario == "admin", nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ok, err := repo.ExistePorUsuario(context.Background(), "admin")
	if err != nil || !ok {
		t.Errorf("esperaba (true, nil), obtuvo (%v, %v)", ok, err)
	}
}

func TestCachedEmpContarAdminsDelegaInner(t *testing.T) {
	inner := &mockRepo{
		contarAdminsActivosExcluyendoFn: func(_ context.Context, _ *sql.Tx, id uint64) (int, error) {
			return 2, nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	n, err := repo.ContarAdminsActivosExcluyendo(context.Background(), nil, 1)
	if err != nil || n != 2 {
		t.Errorf("esperaba (2, nil), obtuvo (%d, %v)", n, err)
	}
}

func TestCachedEmpObtenerTiendaActivaDelegaInner(t *testing.T) {
	inner := &mockRepo{
		obtenerTiendaActivaPorIDFn: func(context.Context, uint64) error { return nil },
	}
	repo := newCachedEmpRepo(t, inner)
	if err := repo.ObtenerTiendaActivaPorID(context.Background(), 1); err != nil {
		t.Errorf("esperaba nil, obtuvo: %v", err)
	}
}

func TestCachedEmpMarcarCambioCompletadoInvalidaByID(t *testing.T) {
	byIDLlamadas := 0
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			byIDLlamadas++
			return &empleados.Empleado{ID: id}, nil
		},
		marcarCambioCompletadoFn: func(context.Context, uint64) error { return nil },
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	if _, err := repo.ObtenerPorID(ctx, 12); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := repo.MarcarCambioCompletado(ctx, 12); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ObtenerPorID(ctx, 12); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas (byID invalidado por MarcarCambioCompletado), obtuvo %d", byIDLlamadas)
	}
}

func TestCachedEmpBeginTxDelegaInner(t *testing.T) {
	inner := &mockRepo{
		beginTxFn: func(context.Context) (*sql.Tx, error) { return nil, nil },
	}
	repo := newCachedEmpRepo(t, inner)
	tx, err := repo.BeginTx(context.Background())
	if err != nil || tx != nil {
		t.Errorf("esperaba (nil, nil), obtuvo (%v, %v)", tx, err)
	}
}

func TestCachedEmpRegistrarLogDelegaInner(t *testing.T) {
	llamado := false
	inner := &mockRepo{
		registrarLogFn: func(_ context.Context, _ *sql.Tx, _, _ uint64, _ string, _ map[string]any) error {
			llamado = true
			return nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	if err := repo.RegistrarLog(context.Background(), nil, 1, 2, "crear", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if !llamado {
		t.Error("esperaba que RegistrarLog delegara a inner")
	}
}

// --- Rutas de error ---

func TestCachedEmpListarErrorPropaga(t *testing.T) {
	errDB := errors.New("db error")
	inner := &mockRepo{
		listarEmpleadosFn: func(context.Context, empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
			return nil, 0, errDB
		},
	}
	repo := newCachedEmpRepo(t, inner)
	_, _, err := repo.ListarEmpleados(context.Background(), empleados.ListarEmpleadosParams{Page: 1})
	if !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
}

func TestCachedEmpActualizarActivoErrorNoInvalidaCache(t *testing.T) {
	byIDLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			byIDLlamadas++
			return &empleados.Empleado{ID: id}, nil
		},
		actualizarActivoFn: func(context.Context, *sql.Tx, uint64, bool) error { return errDB },
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	if _, err := repo.ObtenerPorID(ctx, 13); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := repo.ActualizarActivo(ctx, nil, 13, false); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}
	if _, err := repo.ObtenerPorID(ctx, 13); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si ActualizarActivo falló; llamadas: %d", byIDLlamadas)
	}
}

func TestCachedEmpActualizarContrasenaErrorNoClear(t *testing.T) {
	byIDLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			byIDLlamadas++
			return &empleados.Empleado{ID: id}, nil
		},
		actualizarContrasenaFn: func(context.Context, uint64, string) error { return errDB },
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	if _, err := repo.ObtenerPorID(ctx, 14); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := repo.ActualizarContrasena(ctx, 14, "x"); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}
	if _, err := repo.ObtenerPorID(ctx, 14); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si ActualizarContrasena falló; llamadas: %d", byIDLlamadas)
	}
}

// --- Error en InsertarEmpleado no invalida caché ---

func TestCachedEmpInsertarErrorNoClearLista(t *testing.T) {
	listarLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		listarEmpleadosFn: func(_ context.Context, _ empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
			listarLlamadas++
			return []empleados.Empleado{}, 0, nil
		},
		insertarEmpleadoFn: func(context.Context, *sql.Tx, empleados.Empleado, string) (uint64, error) {
			return 0, errDB
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	p := empleados.ListarEmpleadosParams{Page: 1, Limit: 10}
	if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.InsertarEmpleado(ctx, nil, empleados.Empleado{}, "x"); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}
	if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si InsertarEmpleado falló; llamadas: %d", listarLlamadas)
	}
}

// --- Error en MarcarCambioCompletado no invalida caché ---

func TestCachedEmpMarcarCambioErrorNoClear(t *testing.T) {
	byIDLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		obtenerPorIDFn: func(_ context.Context, id uint64) (*empleados.Empleado, error) {
			byIDLlamadas++
			return &empleados.Empleado{ID: id}, nil
		},
		marcarCambioCompletadoFn: func(context.Context, uint64) error { return errDB },
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	if _, err := repo.ObtenerPorID(ctx, 15); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := repo.MarcarCambioCompletado(ctx, 15); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}
	if _, err := repo.ObtenerPorID(ctx, 15); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si MarcarCambioCompletado falló; llamadas: %d", byIDLlamadas)
	}
}

// --- listarEmpleadosKey con punteros no nil ---

func TestCachedEmpListarKeyConTiendaYActivoDistinto(t *testing.T) {
	llamadas := 0
	tiendaID := uint64(3)
	activo := true
	inner := &mockRepo{
		listarEmpleadosFn: func(_ context.Context, _ empleados.ListarEmpleadosParams) ([]empleados.Empleado, int, error) {
			llamadas++
			return []empleados.Empleado{}, 0, nil
		},
	}
	repo := newCachedEmpRepo(t, inner)
	ctx := context.Background()
	p1 := empleados.ListarEmpleadosParams{Page: 1, Limit: 10, TiendaID: &tiendaID}
	p2 := empleados.ListarEmpleadosParams{Page: 1, Limit: 10, Activo: &activo}
	p3 := empleados.ListarEmpleadosParams{Page: 1, Limit: 10}
	for _, p := range []empleados.ListarEmpleadosParams{p1, p2, p3} {
		if _, _, err := repo.ListarEmpleados(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if llamadas != 3 {
		t.Errorf("3 params distintos deben generar claves distintas; llamadas: %d", llamadas)
	}
}
