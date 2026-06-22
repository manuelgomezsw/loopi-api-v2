package unidades_medida_test

import (
	"errors"
	"testing"
	"time"

	um "github.com/manuelgomezsw/loopi-api-v2/internal/unidades_medida"
)

// --- Passthrough methods ---

func TestCachedExisteConCodigoDelegaInner(t *testing.T) {
	inner := &mockRepo{
		existeConCodigoFunc: func(codigo string) (bool, error) {
			return codigo == "kg", nil
		},
	}
	repo := newCachedRepo(t, inner)
	ok, err := repo.ExisteConCodigo("kg")
	if err != nil || !ok {
		t.Errorf("esperaba (true, nil), obtuvo (%v, %v)", ok, err)
	}
}

func TestCachedContarItemsDelegaInner(t *testing.T) {
	inner := &mockRepo{
		contarItemsFunc: func(uint64) (int, error) { return 5, nil },
	}
	repo := newCachedRepo(t, inner)
	n, err := repo.ContarItemsConUnidadCanonica(1)
	if err != nil || n != 5 {
		t.Errorf("esperaba (5, nil), obtuvo (%d, %v)", n, err)
	}
}

func TestCachedContarUnidadesActivasPorTipoDelegaInner(t *testing.T) {
	inner := &mockRepo{
		contarUnidadesActivasPorTipoFunc: func(string, uint64) (int, error) { return 3, nil },
	}
	repo := newCachedRepo(t, inner)
	n, err := repo.ContarUnidadesActivasPorTipo("peso", 2)
	if err != nil || n != 3 {
		t.Errorf("esperaba (3, nil), obtuvo (%d, %v)", n, err)
	}
}

// --- Rutas de error en lecturas ---

func TestCachedObtenerPorIDErrorPropaga(t *testing.T) {
	errDB := errors.New("not found")
	inner := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) {
			return nil, errDB
		},
	}
	repo := newCachedRepo(t, inner)
	_, err := repo.ObtenerPorID(1)
	if !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
}

func TestCachedObtenerPorIDConItemsErrorPropaga(t *testing.T) {
	errDB := errors.New("not found")
	inner := &mockRepo{
		obtenerPorIDConItemsFunc: func(uint64) (*um.DetalleUMResponse, error) {
			return nil, errDB
		},
	}
	repo := newCachedRepo(t, inner)
	_, err := repo.ObtenerPorIDConItems(1)
	if !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
}

// --- listarKey con activo filtrado ---

func TestCachedListarConActivoFiltraDistinto(t *testing.T) {
	llamadas := 0
	activo := true
	inner := &mockRepo{
		listarFunc: func(p *um.ListarUMParams) (*um.ListarUMResponse, error) {
			llamadas++
			return &um.ListarUMResponse{Total: 1}, nil
		},
	}
	repo := newCachedRepo(t, inner)
	p1 := &um.ListarUMParams{Page: 1, Limit: 10, Activo: &activo}
	p2 := &um.ListarUMParams{Page: 1, Limit: 10}
	if _, err := repo.Listar(p1); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.Listar(p2); err != nil {
		t.Fatal(err)
	}
	if llamadas != 2 {
		t.Errorf("params distintos deben generar claves distintas; llamadas: %d", llamadas)
	}
}

// --- Crear error no invalida caché ---

func TestCachedCrearErrorNoClearLista(t *testing.T) {
	listarLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		listarFunc: func(*um.ListarUMParams) (*um.ListarUMResponse, error) {
			listarLlamadas++
			return &um.ListarUMResponse{Total: 1}, nil
		},
		crearFunc: func(*um.CrearUMRequest) (*um.UnidadMedida, error) {
			return nil, errDB
		},
	}
	repo := newCachedRepo(t, inner)
	p := &um.ListarUMParams{Page: 1, Limit: 10}
	if _, err := repo.Listar(p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.Crear(&um.CrearUMRequest{}); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}
	if _, err := repo.Listar(p); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si Crear falló; llamadas: %d", listarLlamadas)
	}
}

// --- Inactivar error no invalida caché ---

func TestCachedInactivarErrorNoInvalidaCache(t *testing.T) {
	byIDLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) {
			byIDLlamadas++
			return &um.UnidadMedida{ID: 6}, nil
		},
		inactivarFunc: func(uint64) error { return errDB },
	}
	repo := newCachedRepo(t, inner)
	if _, err := repo.ObtenerPorID(6); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := repo.Inactivar(6); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}
	if _, err := repo.ObtenerPorID(6); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si Inactivar falló; llamadas: %d", byIDLlamadas)
	}
}

func newCachedRepo(t *testing.T, inner um.UMRepository) um.UMRepository {
	t.Helper()
	repo, err := um.NewCachedRepository(inner, time.Minute)
	if err != nil {
		t.Fatalf("NewCachedRepository: %v", err)
	}
	return repo
}

// --- Listar ---

func TestCachedListarMissLlamaInner(t *testing.T) {
	llamadas := 0
	expected := &um.ListarUMResponse{Total: 2}
	inner := &mockRepo{
		listarFunc: func(*um.ListarUMParams) (*um.ListarUMResponse, error) {
			llamadas++
			return expected, nil
		},
	}
	repo := newCachedRepo(t, inner)
	p := &um.ListarUMParams{Page: 1, Limit: 10}
	got, err := repo.Listar(p)
	if err != nil {
		t.Fatalf("Listar error: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("Total esperado 2, obtuvo %d", got.Total)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada a inner.Listar, obtuvo %d", llamadas)
	}
}

func TestCachedListarHitNoLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		listarFunc: func(*um.ListarUMParams) (*um.ListarUMResponse, error) {
			llamadas++
			return &um.ListarUMResponse{Total: 1}, nil
		},
	}
	repo := newCachedRepo(t, inner)
	p := &um.ListarUMParams{Page: 1, Limit: 10}
	if _, err := repo.Listar(p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.Listar(p); err != nil {
		t.Fatal(err)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada total (hit en segunda), obtuvo %d", llamadas)
	}
}

func TestCachedListarErrorNoModificaCache(t *testing.T) {
	errDB := errors.New("db error")
	inner := &mockRepo{
		listarFunc: func(*um.ListarUMParams) (*um.ListarUMResponse, error) {
			return nil, errDB
		},
	}
	repo := newCachedRepo(t, inner)
	_, err := repo.Listar(&um.ListarUMParams{Page: 1, Limit: 10})
	if !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
}

// --- ObtenerPorID ---

func TestCachedObtenerPorIDMissLlamaInner(t *testing.T) {
	llamadas := 0
	expected := &um.UnidadMedida{ID: 7, Codigo: "kg"}
	inner := &mockRepo{
		obtenerPorIDFunc: func(id uint64) (*um.UnidadMedida, error) {
			llamadas++
			return expected, nil
		},
	}
	repo := newCachedRepo(t, inner)
	got, err := repo.ObtenerPorID(7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Codigo != "kg" {
		t.Errorf("esperaba 'kg', obtuvo '%s'", got.Codigo)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada, obtuvo %d", llamadas)
	}
}

func TestCachedObtenerPorIDHitNoLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) {
			llamadas++
			return &um.UnidadMedida{ID: 3}, nil
		},
	}
	repo := newCachedRepo(t, inner)
	if _, err := repo.ObtenerPorID(3); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.ObtenerPorID(3); err != nil {
		t.Fatal(err)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada total, obtuvo %d", llamadas)
	}
}

// --- ObtenerPorIDConItems ---

func TestCachedObtenerPorIDConItemsHitNoLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		obtenerPorIDConItemsFunc: func(uint64) (*um.DetalleUMResponse, error) {
			llamadas++
			return &um.DetalleUMResponse{UnidadMedida: um.UnidadMedida{ID: 5}}, nil
		},
	}
	repo := newCachedRepo(t, inner)
	if _, err := repo.ObtenerPorIDConItems(5); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.ObtenerPorIDConItems(5); err != nil {
		t.Fatal(err)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada total, obtuvo %d", llamadas)
	}
}

// --- Crear invalida listCache ---

func TestCachedCrearInvalidaListaCache(t *testing.T) {
	listarLlamadas := 0
	inner := &mockRepo{
		listarFunc: func(*um.ListarUMParams) (*um.ListarUMResponse, error) {
			listarLlamadas++
			return &um.ListarUMResponse{Total: listarLlamadas}, nil
		},
		crearFunc: func(*um.CrearUMRequest) (*um.UnidadMedida, error) {
			return &um.UnidadMedida{ID: 10}, nil
		},
	}
	repo := newCachedRepo(t, inner)
	p := &um.ListarUMParams{Page: 1, Limit: 10}

	// Primer Listar — miss, popula caché.
	if _, err := repo.Listar(p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	// Crear — debe invalidar caché.
	if _, err := repo.Crear(&um.CrearUMRequest{Codigo: "x", Nombre: "X", TipoMedida: "peso", FactorConversion: 1}); err != nil {
		t.Fatal(err)
	}

	// Segundo Listar — debe llamar inner de nuevo (caché limpia).
	if _, err := repo.Listar(p); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas a Listar (caché invalidada por Crear), obtuvo %d", listarLlamadas)
	}
}

// --- Editar invalida byID y lista ---

func TestCachedEditarInvalidaCache(t *testing.T) {
	byIDLlamadas := 0
	inner := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) {
			byIDLlamadas++
			return &um.UnidadMedida{ID: 2}, nil
		},
		editarFunc: func(id uint64, req *um.EditarUMRequest) (*um.UnidadMedida, error) {
			return &um.UnidadMedida{ID: id}, nil
		},
	}
	repo := newCachedRepo(t, inner)

	// Cachea ObtenerPorID.
	if _, err := repo.ObtenerPorID(2); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	// Editar invalida byIDCache.
	nombre := "nuevo"
	if _, err := repo.Editar(2, &um.EditarUMRequest{Nombre: &nombre}); err != nil {
		t.Fatal(err)
	}

	// ObtenerPorID debe llamar inner otra vez.
	if _, err := repo.ObtenerPorID(2); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas a ObtenerPorID (invalidada por Editar), obtuvo %d", byIDLlamadas)
	}
}

// --- Error en escritura no invalida caché ---

func TestCachedEditarErrorNoInvalidaCache(t *testing.T) {
	byIDLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) {
			byIDLlamadas++
			return &um.UnidadMedida{ID: 1}, nil
		},
		editarFunc: func(uint64, *um.EditarUMRequest) (*um.UnidadMedida, error) {
			return nil, errDB
		},
	}
	repo := newCachedRepo(t, inner)

	if _, err := repo.ObtenerPorID(1); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	nombre := "x"
	if _, err := repo.Editar(1, &um.EditarUMRequest{Nombre: &nombre}); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}

	// Caché no fue invalidada; inner NO debe ser llamado en la siguiente lectura.
	if _, err := repo.ObtenerPorID(1); err != nil {
		t.Fatal(err)
	}
	if byIDLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si Editar falló; llamadas: %d", byIDLlamadas)
	}
}

// --- Inactivar invalida byID y lista ---

func TestCachedInactivarInvalidaCache(t *testing.T) {
	listarLlamadas := 0
	inner := &mockRepo{
		listarFunc: func(*um.ListarUMParams) (*um.ListarUMResponse, error) {
			listarLlamadas++
			return &um.ListarUMResponse{Total: 1}, nil
		},
		inactivarFunc: func(uint64) error { return nil },
	}
	repo := newCachedRepo(t, inner)
	p := &um.ListarUMParams{Page: 1, Limit: 10}

	if _, err := repo.Listar(p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := repo.Inactivar(1); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.Listar(p); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas a Listar (caché invalidada por Inactivar), obtuvo %d", listarLlamadas)
	}
}
