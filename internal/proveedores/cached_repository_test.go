package proveedores_test

import (
	"errors"
	"testing"
	"time"

	pv "github.com/manuelgomezsw/loopi-api-v2/internal/proveedores"
)

func newCachedRepo(t *testing.T, inner pv.Repository) pv.Repository {
	t.Helper()
	repo, err := pv.NewCachedRepository(inner, time.Hour)
	if err != nil {
		t.Fatalf("error creando cached repository: %v", err)
	}
	return repo
}

// --- Passthrough methods ---

func TestCachedExisteNITDelegaInner(t *testing.T) {
	inner := &mockRepo{
		existeNITFunc: func(nit string, _ *uint64) (bool, error) { return nit == "900123456-7", nil },
	}
	repo := newCachedRepo(t, inner)
	ok, err := repo.ExisteNIT("900123456-7", nil)
	if err != nil || !ok {
		t.Errorf("esperaba (true, nil), obtuvo (%v, %v)", ok, err)
	}
}

func TestCachedContarItemsAsignadosDelegaInner(t *testing.T) {
	inner := &mockRepo{
		contarItemsAsignadosFunc: func(uint64) (int, error) { return 5, nil },
	}
	repo := newCachedRepo(t, inner)
	n, err := repo.ContarItemsAsignados(1)
	if err != nil || n != 5 {
		t.Errorf("esperaba (5, nil), obtuvo (%d, %v)", n, err)
	}
}

// --- Lecturas: hit / miss ---

func TestCachedObtenerPorID_Miss_InvocaInnerYCachea(t *testing.T) {
	llamadas := 0
	p := proveedorEjemplo()
	inner := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*pv.Proveedor, error) { llamadas++; return p, nil },
	}
	repo := newCachedRepo(t, inner)

	if _, err := repo.ObtenerPorID(p.ID); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if llamadas != 1 {
		t.Fatalf("esperaba 1 llamada al inner tras el miss, obtuvo %d", llamadas)
	}

	// Ristretto escribe de forma asíncrona; se da un margen breve para que el
	// Set del miss anterior quede visible antes de verificar el hit.
	time.Sleep(10 * time.Millisecond)

	if _, err := repo.ObtenerPorID(p.ID); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if llamadas != 1 {
		t.Errorf("esperaba que el hit de caché NO invoque al inner de nuevo; llamadas=%d", llamadas)
	}
}

func TestCachedObtenerPorIDErrorPropagaYNoCachea(t *testing.T) {
	errDB := errors.New("not found")
	llamadas := 0
	inner := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*pv.Proveedor, error) { llamadas++; return nil, errDB },
	}
	repo := newCachedRepo(t, inner)

	_, err := repo.ObtenerPorID(1)
	if !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
	// Segunda llamada también debe invocar el inner (el error no se cacheó).
	_, _ = repo.ObtenerPorID(1)
	if llamadas != 2 {
		t.Errorf("esperaba 2 llamadas al inner (sin cachear el error), obtuvo %d", llamadas)
	}
}

// --- Escrituras: invalidación ---

func TestCachedCrear_InvalidaListCache(t *testing.T) {
	p := proveedorEjemplo()
	listarLlamadas := 0
	inner := &mockRepo{
		crearFunc: func(*pv.CrearProveedorRequest) (*pv.Proveedor, error) { return p, nil },
		listarFunc: func(*pv.FiltrosListado) (*pv.ListarProveedoresResponse, error) {
			listarLlamadas++
			return &pv.ListarProveedoresResponse{Proveedores: []pv.Proveedor{*p}, Total: 1}, nil
		},
	}
	repo := newCachedRepo(t, inner)
	filtros := &pv.FiltrosListado{Page: 1, Limit: 50}

	if _, err := repo.Listar(filtros); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if _, err := repo.Crear(&pv.CrearProveedorRequest{RazonSocial: p.RazonSocial, NIT: p.NIT}); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if _, err := repo.Listar(filtros); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if listarLlamadas != 2 {
		t.Errorf("esperaba que Crear invalide la caché de listado (2 llamadas a Listar), obtuvo %d", listarLlamadas)
	}
}

func TestCachedCambiarEstado_InvalidaByIDCache(t *testing.T) {
	p := proveedorEjemplo()
	obtenerLlamadas := 0
	inner := &mockRepo{
		obtenerPorIDFunc:  func(uint64) (*pv.Proveedor, error) { obtenerLlamadas++; return p, nil },
		cambiarEstadoFunc: func(uint64, bool) error { return nil },
	}
	repo := newCachedRepo(t, inner)

	if _, err := repo.ObtenerPorID(p.ID); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if err := repo.CambiarEstado(p.ID, false); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if _, err := repo.ObtenerPorID(p.ID); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if obtenerLlamadas != 2 {
		t.Errorf("esperaba que CambiarEstado invalide la caché por ID (2 llamadas), obtuvo %d", obtenerLlamadas)
	}
}

func TestCachedCambiarEstado_ErrorInnerPropaga(t *testing.T) {
	errDB := errors.New("db error")
	inner := &mockRepo{
		cambiarEstadoFunc: func(uint64, bool) error { return errDB },
	}
	repo := newCachedRepo(t, inner)
	if err := repo.CambiarEstado(1, false); !errors.Is(err, errDB) {
		t.Errorf("esperaba errDB, obtuvo: %v", err)
	}
}
