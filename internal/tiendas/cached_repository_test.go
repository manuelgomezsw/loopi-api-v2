package tiendas_test

import (
	"errors"
	"testing"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/tiendas"
)

func newCachedTiendaRepo(t *testing.T, inner tiendas.TiendaRepository) tiendas.TiendaRepository {
	t.Helper()
	repo, err := tiendas.NewCachedRepository(inner, time.Minute)
	if err != nil {
		t.Fatalf("NewCachedRepository: %v", err)
	}
	return repo
}

// --- ObtenerPorID ---

func TestCachedTiendaObtenerPorIDMissLlamaInner(t *testing.T) {
	llamadas := 0
	expected := tiendas.Tienda{ID: 1, Nombre: "Central"}
	inner := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			llamadas++
			return expected, nil
		},
	}
	repo := newCachedTiendaRepo(t, inner)
	got, err := repo.ObtenerPorID(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Nombre != "Central" {
		t.Errorf("esperaba 'Central', obtuvo '%s'", got.Nombre)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada, obtuvo %d", llamadas)
	}
}

func TestCachedTiendaObtenerPorIDHitNoLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		obtenerFunc: func(uint64) (tiendas.Tienda, error) {
			llamadas++
			return tiendas.Tienda{ID: 2, Nombre: "Norte"}, nil
		},
	}
	repo := newCachedTiendaRepo(t, inner)
	if _, err := repo.ObtenerPorID(2); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.ObtenerPorID(2); err != nil {
		t.Fatal(err)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada total (hit en segunda), obtuvo %d", llamadas)
	}
}

func TestCachedTiendaObtenerPorIDErrorNoCachea(t *testing.T) {
	errDB := errors.New("not found")
	inner := &mockRepo{
		obtenerFunc: func(uint64) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, errDB
		},
	}
	repo := newCachedTiendaRepo(t, inner)
	_, err := repo.ObtenerPorID(99)
	if !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
}

// --- Listar ---

func TestCachedTiendaListarMissLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			llamadas++
			return []tiendas.Tienda{{ID: 1}}, 1, nil
		},
	}
	repo := newCachedTiendaRepo(t, inner)
	items, total, err := repo.Listar("", 1, 10)
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

func TestCachedTiendaListarHitNoLlamaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			llamadas++
			return []tiendas.Tienda{{ID: 1}}, 1, nil
		},
	}
	repo := newCachedTiendaRepo(t, inner)
	if _, _, err := repo.Listar("activo", 1, 10); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, _, err := repo.Listar("activo", 1, 10); err != nil {
		t.Fatal(err)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada total, obtuvo %d", llamadas)
	}
}

// --- Crear invalida lista ---

func TestCachedTiendaCrearInvalidaListaCache(t *testing.T) {
	listarLlamadas := 0
	inner := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			listarLlamadas++
			return []tiendas.Tienda{}, 0, nil
		},
		crearFunc: func(t tiendas.Tienda) (tiendas.Tienda, error) {
			return tiendas.Tienda{ID: 1}, nil
		},
	}
	repo := newCachedTiendaRepo(t, inner)

	if _, _, err := repo.Listar("", 1, 10); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if _, err := repo.Crear(tiendas.Tienda{Nombre: "Nueva"}); err != nil {
		t.Fatal(err)
	}

	if _, _, err := repo.Listar("", 1, 10); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas (caché invalidada por Crear), obtuvo %d", listarLlamadas)
	}
}

// --- Actualizar invalida byID y lista ---

func TestCachedTiendaActualizarInvalidaCache(t *testing.T) {
	obtenerLlamadas := 0
	inner := &mockRepo{
		obtenerFunc: func(uint64) (tiendas.Tienda, error) {
			obtenerLlamadas++
			return tiendas.Tienda{ID: 3, Nombre: "Sur"}, nil
		},
		actualizarFunc: func(id uint64, req tiendas.TiendaUpdateRequest, adminID uint64, ahora time.Time) (tiendas.Tienda, error) {
			return tiendas.Tienda{ID: id, Nombre: req.Nombre}, nil
		},
	}
	repo := newCachedTiendaRepo(t, inner)

	if _, err := repo.ObtenerPorID(3); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if _, err := repo.Actualizar(3, tiendas.TiendaUpdateRequest{Nombre: "Sur Actualizada"}, 1, time.Now()); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.ObtenerPorID(3); err != nil {
		t.Fatal(err)
	}
	if obtenerLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas (caché invalidada por Actualizar), obtuvo %d", obtenerLlamadas)
	}
}

// --- Error en escritura no invalida caché ---

func TestCachedTiendaActualizarErrorNoInvalidaCache(t *testing.T) {
	obtenerLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		obtenerFunc: func(uint64) (tiendas.Tienda, error) {
			obtenerLlamadas++
			return tiendas.Tienda{ID: 4}, nil
		},
		actualizarFunc: func(uint64, tiendas.TiendaUpdateRequest, uint64, time.Time) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, errDB
		},
	}
	repo := newCachedTiendaRepo(t, inner)

	if _, err := repo.ObtenerPorID(4); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if _, err := repo.Actualizar(4, tiendas.TiendaUpdateRequest{}, 1, time.Now()); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}

	// Caché intacta; inner NO debe ser llamado.
	if _, err := repo.ObtenerPorID(4); err != nil {
		t.Fatal(err)
	}
	if obtenerLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si Actualizar falló; llamadas: %d", obtenerLlamadas)
	}
}

// --- CambiarActivo invalida byID y lista ---

func TestCachedTiendaCambiarActivoInvalidaCache(t *testing.T) {
	listarLlamadas := 0
	inner := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			listarLlamadas++
			return []tiendas.Tienda{{ID: 5}}, 1, nil
		},
		cambiarActivoFunc: func(id uint64, activo bool, adminID uint64, ahora time.Time) (tiendas.Tienda, error) {
			return tiendas.Tienda{ID: id, Activo: activo}, nil
		},
	}
	repo := newCachedTiendaRepo(t, inner)

	if _, _, err := repo.Listar("", 1, 10); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if _, err := repo.CambiarActivo(5, false, 1, time.Now()); err != nil {
		t.Fatal(err)
	}

	if _, _, err := repo.Listar("", 1, 10); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 2 {
		t.Errorf("esperaba 2 llamadas (caché invalidada por CambiarActivo), obtuvo %d", listarLlamadas)
	}
}

// --- Rutas de error en lecturas ---

func TestCachedTiendaListarErrorPropaga(t *testing.T) {
	errDB := errors.New("db error")
	inner := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			return nil, 0, errDB
		},
	}
	repo := newCachedTiendaRepo(t, inner)
	_, _, err := repo.Listar("", 1, 10)
	if !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
}

func TestCachedTiendaCrearErrorNoClearLista(t *testing.T) {
	listarLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			listarLlamadas++
			return []tiendas.Tienda{{ID: 1}}, 1, nil
		},
		crearFunc: func(tiendas.Tienda) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, errDB
		},
	}
	repo := newCachedTiendaRepo(t, inner)
	if _, _, err := repo.Listar("", 1, 10); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.Crear(tiendas.Tienda{Nombre: "X"}); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}
	if _, _, err := repo.Listar("", 1, 10); err != nil {
		t.Fatal(err)
	}
	if listarLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si Crear falló; llamadas: %d", listarLlamadas)
	}
}

func TestCachedTiendaCambiarActivoErrorNoInvalidaCache(t *testing.T) {
	obtenerLlamadas := 0
	errDB := errors.New("db error")
	inner := &mockRepo{
		obtenerFunc: func(uint64) (tiendas.Tienda, error) {
			obtenerLlamadas++
			return tiendas.Tienda{ID: 6, Activo: true}, nil
		},
		cambiarActivoFunc: func(uint64, bool, uint64, time.Time) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, errDB
		},
	}
	repo := newCachedTiendaRepo(t, inner)
	if _, err := repo.ObtenerPorID(6); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.CambiarActivo(6, false, 1, time.Now()); !errors.Is(err, errDB) {
		t.Fatalf("esperaba errDB, obtuvo: %v", err)
	}
	if _, err := repo.ObtenerPorID(6); err != nil {
		t.Fatal(err)
	}
	if obtenerLlamadas != 1 {
		t.Errorf("caché no debió invalidarse si CambiarActivo falló; llamadas: %d", obtenerLlamadas)
	}
}
