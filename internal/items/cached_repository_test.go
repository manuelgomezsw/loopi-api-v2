package items_test

import (
	"testing"
	"time"

	it "github.com/manuelgomezsw/loopi-api-v2/internal/items"
)

func newCachedRepo(t *testing.T, inner it.Repository) it.Repository {
	t.Helper()
	repo, err := it.NewCachedRepository(inner, time.Hour)
	if err != nil {
		t.Fatalf("error creando cached repository: %v", err)
	}
	return repo
}

// TestCacheInvalidacionEnCreate (quickstart.md §5): tras Crear, el listado cacheado se invalida.
func TestCacheInvalidacionEnCreate(t *testing.T) {
	llamadasListar := 0
	inner := baseMockRepo()
	inner.listarFunc = func(*it.FiltrosListado) (*it.ListarItemsResponse, error) {
		llamadasListar++
		return &it.ListarItemsResponse{Items: []it.ItemConNombres{}, Total: 0}, nil
	}
	repo := newCachedRepo(t, inner)

	filtros := &it.FiltrosListado{Pagina: 1, PorPagina: 50}
	if _, err := repo.Listar(filtros); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.Listar(filtros); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if llamadasListar != 1 {
		t.Fatalf("esperaba 1 llamada al inner tras 2 lecturas cacheadas, obtuvo %d", llamadasListar)
	}

	if _, err := repo.Crear(crearReqValido(), 1); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}

	if _, err := repo.Listar(filtros); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if llamadasListar != 2 {
		t.Fatalf("esperaba que Crear invalidara la caché del listado; llamadas=%d", llamadasListar)
	}
}

// TestCacheInvalidacionEnInactivar (quickstart.md §5): tras CambiarEstado, la caché por ID
// y de detalle del item se invalidan.
func TestCacheInvalidacionEnInactivar(t *testing.T) {
	llamadasObtener := 0
	inner := baseMockRepo()
	inner.obtenerPorIDFunc = func(uint64) (*it.Item, error) {
		llamadasObtener++
		return itemEjemplo(), nil
	}
	repo := newCachedRepo(t, inner)

	if _, err := repo.ObtenerPorID(1); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := repo.ObtenerPorID(1); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if llamadasObtener != 1 {
		t.Fatalf("esperaba 1 llamada al inner tras 2 lecturas cacheadas, obtuvo %d", llamadasObtener)
	}

	if _, err := repo.CambiarEstado(1, false, 1); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}

	if _, err := repo.ObtenerPorID(1); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if llamadasObtener != 2 {
		t.Fatalf("esperaba que CambiarEstado invalidara la caché por ID; llamadas=%d", llamadasObtener)
	}
}
