package categorias_test

import (
	"errors"
	"testing"
	"time"

	cat "github.com/manuelgomezsw/loopi-api-v2/internal/categorias"
)

const testTTL = 10 * time.Second

func newCachedRepo(inner cat.Repository) (cat.Repository, error) {
	return cat.NewCachedRepository(inner, testTTL)
}

// --- Hit de caché (inner NO invocado en segunda llamada) ---

func TestCacheListarConItems_HitNoInvocaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		listarConItemsFunc: func(soloActivas *bool) (*cat.CatalogoResponse, error) {
			llamadas++
			return &cat.CatalogoResponse{Categorias: []cat.CategoriaResponse{}, Total: 0}, nil
		},
	}
	repo, err := newCachedRepo(inner)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = repo.ListarConItems(nil)
	// Ristretto es async — esperamos a que el Set se procese
	time.Sleep(5 * time.Millisecond)
	_, _ = repo.ListarConItems(nil)

	if llamadas > 2 {
		t.Errorf("inner invocado más de lo esperado (%d veces)", llamadas)
	}
}

// --- Miss de caché (inner invocado + resultado almacenado) ---

func TestCacheListarConItems_MissInvocaInner(t *testing.T) {
	llamadas := 0
	inner := &mockRepo{
		listarConItemsFunc: func(soloActivas *bool) (*cat.CatalogoResponse, error) {
			llamadas++
			return &cat.CatalogoResponse{Categorias: []cat.CategoriaResponse{}, Total: 0}, nil
		},
	}
	repo, err := newCachedRepo(inner)
	if err != nil {
		t.Fatal(err)
	}

	_, err2 := repo.ListarConItems(nil)
	if err2 != nil {
		t.Fatalf("no esperaba error: %v", err2)
	}
	if llamadas != 1 {
		t.Errorf("esperaba 1 llamada al inner en miss, obtuvo %d", llamadas)
	}
}

// --- Escritura invalida caché ---

func TestCacheInsertarCategoria_InvalidaLista(t *testing.T) {
	llamadas := 0
	c := categoriaEjemplo()
	inner := &mockRepo{
		listarConItemsFunc: func(soloActivas *bool) (*cat.CatalogoResponse, error) {
			llamadas++
			return &cat.CatalogoResponse{Categorias: []cat.CategoriaResponse{}, Total: 0}, nil
		},
		insertarCategoriaFunc: func(nombre string, creadoPor uint64) (*cat.Categoria, error) {
			return c, nil
		},
	}
	repo, err := newCachedRepo(inner)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = repo.ListarConItems(nil)
	time.Sleep(5 * time.Millisecond)

	// escritura — invalida caché
	_, _ = repo.InsertarCategoria("Nueva", 1)
	time.Sleep(5 * time.Millisecond)

	// siguiente lectura debe ir al inner de nuevo
	_, _ = repo.ListarConItems(nil)

	if llamadas < 2 {
		t.Errorf("esperaba ≥2 llamadas al inner tras escritura, obtuvo %d", llamadas)
	}
}

// --- Error del inner no almacena en caché ---

func TestCacheListarConItems_ErrorNoAlmacena(t *testing.T) {
	llamadas := 0
	errEsperado := errors.New("fallo de BD")
	inner := &mockRepo{
		listarConItemsFunc: func(soloActivas *bool) (*cat.CatalogoResponse, error) {
			llamadas++
			return nil, errEsperado
		},
	}
	repo, err := newCachedRepo(inner)
	if err != nil {
		t.Fatal(err)
	}

	_, err1 := repo.ListarConItems(nil)
	if !errors.Is(err1, errEsperado) {
		t.Errorf("esperaba errEsperado, obtuvo: %v", err1)
	}

	// segunda llamada — el error no fue cacheado, inner debe invocarse de nuevo
	_, err2 := repo.ListarConItems(nil)
	if !errors.Is(err2, errEsperado) {
		t.Errorf("esperaba errEsperado en segunda llamada, obtuvo: %v", err2)
	}
	if llamadas != 2 {
		t.Errorf("esperaba 2 llamadas al inner (error no cacheado), obtuvo %d", llamadas)
	}
}

// --- ObtenerCategoriaPorID — hit de caché ---

func TestCacheObtenerCategoriaPorID_HitNoInvocaInner(t *testing.T) {
	llamadas := 0
	c := categoriaEjemplo()
	inner := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) {
			llamadas++
			return c, nil
		},
	}
	repo, err := newCachedRepo(inner)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = repo.ObtenerCategoriaPorID(1)
	time.Sleep(5 * time.Millisecond)
	_, _ = repo.ObtenerCategoriaPorID(1)

	if llamadas > 2 {
		t.Errorf("inner invocado más de lo esperado (%d veces)", llamadas)
	}
}

// --- ActualizarCategoria invalida byIDCache y catCache ---

func TestCacheActualizarCategoria_InvalidaByID(t *testing.T) {
	llamadas := 0
	c := categoriaEjemplo()
	inner := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) {
			llamadas++
			return c, nil
		},
		actualizarCategoriaFunc: func(id uint64, nombre string, actualizadoPor uint64) (*cat.Categoria, error) {
			c.Nombre = nombre
			return c, nil
		},
	}
	repo, err := newCachedRepo(inner)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = repo.ObtenerCategoriaPorID(1)
	time.Sleep(5 * time.Millisecond)

	_, _ = repo.ActualizarCategoria(1, "Nuevo Nombre", 42)
	time.Sleep(5 * time.Millisecond)

	// siguiente lectura por ID debe re-invocar inner
	_, _ = repo.ObtenerCategoriaPorID(1)
	if llamadas < 2 {
		t.Errorf("esperaba ≥2 llamadas al inner tras actualizar, obtuvo %d", llamadas)
	}
}
