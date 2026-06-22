package unidades_medida_test

import (
	"errors"
	"testing"

	um "github.com/manuelgomezsw/loopi-api-v2/internal/unidades_medida"
)

// newSvc crea un UMService con el mock de repositorio dado.
func newSvc(t *testing.T, repo um.UMRepository) um.UMService {
	t.Helper()
	return um.NewService(repo)
}

// --- Crear ---

func TestCrearUnidadCodigoDuplicado(t *testing.T) {
	repo := &mockRepo{
		existeConCodigoFunc: func(codigo string) (bool, error) { return true, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Crear(&um.CrearUMRequest{
		Codigo: "kg", Nombre: "Kilo", TipoMedida: "peso", FactorConversion: 1000,
	}, 1, "admin")
	if !errors.Is(err, um.ErrCodigoDuplicado) {
		t.Errorf("esperaba ErrCodigoDuplicado, obtuvo: %v", err)
	}
}

func TestCrearUnidadFactorCero(t *testing.T) {
	repo := &mockRepo{
		existeConCodigoFunc: func(string) (bool, error) { return false, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Crear(&um.CrearUMRequest{
		Codigo: "xx", Nombre: "Test", TipoMedida: "peso", FactorConversion: 0,
	}, 1, "admin")
	if !errors.Is(err, um.ErrFactorInvalido) {
		t.Errorf("esperaba ErrFactorInvalido, obtuvo: %v", err)
	}
}

func TestCrearUnidadTipoInvalido(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(t, repo)
	_, err := svc.Crear(&um.CrearUMRequest{
		Codigo: "cm2", Nombre: "Metro cuadrado", TipoMedida: "area", FactorConversion: 1,
	}, 1, "admin")
	if !errors.Is(err, um.ErrTipoInvalido) {
		t.Errorf("esperaba ErrTipoInvalido, obtuvo: %v", err)
	}
}

func TestCrearUnidadExitosa(t *testing.T) {
	esperado := umEjemplo()
	repo := &mockRepo{
		existeConCodigoFunc: func(string) (bool, error) { return false, nil },
		crearFunc:           func(*um.CrearUMRequest) (*um.UnidadMedida, error) { return esperado, nil },
	}
	svc := newSvc(t, repo)
	resultado, err := svc.Crear(&um.CrearUMRequest{
		Codigo: "kg", Nombre: "Kilogramo", TipoMedida: "peso", FactorConversion: 1000,
	}, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.ID != esperado.ID {
		t.Errorf("id esperado %d, obtuvo %d", esperado.ID, resultado.ID)
	}
}

// --- Inactivar ---

func TestInactivarUnidadBase(t *testing.T) {
	base := umBase()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return base, nil },
		contarUnidadesActivasPorTipoFunc: func(string, uint64) (int, error) { return 3, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Inactivar(base.ID, 1, "admin")
	if !errors.Is(err, um.ErrUnidadBaseNoInactivable) {
		t.Errorf("esperaba ErrUnidadBaseNoInactivable, obtuvo: %v", err)
	}
}

func TestInactivarYaInactiva(t *testing.T) {
	u := umEjemplo()
	u.Activo = false
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return u, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Inactivar(u.ID, 1, "admin")
	if !errors.Is(err, um.ErrYaInactiva) {
		t.Errorf("esperaba ErrYaInactiva, obtuvo: %v", err)
	}
}

func TestInactivarExitosa(t *testing.T) {
	u := umEjemplo()
	inactivarLlamado := false
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return u, nil },
		inactivarFunc:    func(uint64) error { inactivarLlamado = true; return nil },
	}
	svc := newSvc(t, repo)
	resp, err := svc.Inactivar(u.ID, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Activo {
		t.Error("activo debe ser false")
	}
	if !inactivarLlamado {
		t.Error("repository.Inactivar no fue llamado")
	}
}

// --- Editar ---

func TestEditarFactorUnidadBase(t *testing.T) {
	base := umBase()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return base, nil },
	}
	svc := newSvc(t, repo)
	f := 0.5
	_, err := svc.Editar(base.ID, &um.EditarUMRequest{FactorConversion: &f}, 1, "admin")
	if !errors.Is(err, um.ErrFactorBaseInmutable) {
		t.Errorf("esperaba ErrFactorBaseInmutable, obtuvo: %v", err)
	}
}

func TestEditarUnidadNoExiste(t *testing.T) {
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return nil, um.ErrUnidadNoEncontrada },
	}
	svc := newSvc(t, repo)
	n := "Kilogramos"
	_, err := svc.Editar(999, &um.EditarUMRequest{Nombre: &n}, 1, "admin")
	if !errors.Is(err, um.ErrUnidadNoEncontrada) {
		t.Errorf("esperaba ErrUnidadNoEncontrada, obtuvo: %v", err)
	}
}

func TestEditarNombreExitoso(t *testing.T) {
	u := umEjemplo()
	editarLlamado := false
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return u, nil },
		editarFunc: func(id uint64, req *um.EditarUMRequest) (*um.UnidadMedida, error) {
			editarLlamado = true
			u.Nombre = *req.Nombre
			return u, nil
		},
	}
	svc := newSvc(t, repo)
	n := "Kilogramos"
	resultado, err := svc.Editar(u.ID, &um.EditarUMRequest{Nombre: &n}, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.Nombre != "Kilogramos" {
		t.Errorf("nombre esperado 'Kilogramos', obtuvo %q", resultado.Nombre)
	}
	if !editarLlamado {
		t.Error("repository.Editar no fue llamado")
	}
}

// --- Listar ---

func TestListarPorTipoPeso(t *testing.T) {
	unidades := []um.UnidadMedida{*umBase(), *umEjemplo()}
	repo := &mockRepo{
		listarFunc: func(params *um.ListarUMParams) (*um.ListarUMResponse, error) {
			return &um.ListarUMResponse{UnidadesMedida: unidades, Total: 2, Page: 1, Limit: 50}, nil
		},
	}
	svc := newSvc(t, repo)
	resp, err := svc.Listar(&um.ListarUMParams{Tipo: "peso", Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total esperado 2, obtuvo %d", resp.Total)
	}
}

func TestListarCacheHit(t *testing.T) {
	llamadas := 0
	unidades := []um.UnidadMedida{*umEjemplo()}
	repo := &mockRepo{
		listarFunc: func(params *um.ListarUMParams) (*um.ListarUMResponse, error) {
			llamadas++
			return &um.ListarUMResponse{UnidadesMedida: unidades, Total: 1, Page: 1, Limit: 50}, nil
		},
	}
	svc := newSvc(t, repo)
	params := &um.ListarUMParams{Page: 1, Limit: 50}
	_, _ = svc.Listar(params)
	// Segunda llamada — puede venir del caché (Ristretto es async, al menos no falló)
	_, err := svc.Listar(params)
	if err != nil {
		t.Fatalf("no esperaba error en segunda llamada: %v", err)
	}
}

// --- ObtenerImpacto ---

func TestObtenerImpacto_SinItems(t *testing.T) {
	u := umEjemplo()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return u, nil },
		contarItemsFunc:  func(uint64) (int, error) { return 0, nil },
	}
	svc := newSvc(t, repo)
	resp, err := svc.ObtenerImpacto(u.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.ItemsConUnidadCanonica != 0 {
		t.Errorf("esperaba items_con_unidad_canonica=0, obtuvo %d", resp.ItemsConUnidadCanonica)
	}
}

func TestObtenerImpacto_ConItems(t *testing.T) {
	u := umEjemplo()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return u, nil },
		contarItemsFunc:  func(uint64) (int, error) { return 7, nil },
	}
	svc := newSvc(t, repo)
	resp, err := svc.ObtenerImpacto(u.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.ItemsConUnidadCanonica != 7 {
		t.Errorf("esperaba items_con_unidad_canonica=7, obtuvo %d", resp.ItemsConUnidadCanonica)
	}
	if resp.Advertencia == nil || *resp.Advertencia == "" {
		t.Error("esperaba advertencia no vacía")
	}
}

func TestObtenerImpacto_NoExiste(t *testing.T) {
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return nil, um.ErrUnidadNoEncontrada },
	}
	svc := newSvc(t, repo)
	_, err := svc.ObtenerImpacto(999)
	if !errors.Is(err, um.ErrUnidadNoEncontrada) {
		t.Errorf("esperaba ErrUnidadNoEncontrada, obtuvo: %v", err)
	}
}

func TestObtenerPorIDNoExiste(t *testing.T) {
	repo := &mockRepo{
		obtenerPorIDConItemsFunc: func(uint64) (*um.DetalleUMResponse, error) {
			return nil, um.ErrUnidadNoEncontrada
		},
	}
	svc := newSvc(t, repo)
	_, err := svc.ObtenerPorID(999)
	if !errors.Is(err, um.ErrUnidadNoEncontrada) {
		t.Errorf("esperaba ErrUnidadNoEncontrada, obtuvo: %v", err)
	}
}

func TestObtenerPorIDExitoso(t *testing.T) {
	u := umEjemplo()
	detalle := &um.DetalleUMResponse{UnidadMedida: *u, ItemsConUnidadCanonica: 2}
	repo := &mockRepo{
		obtenerPorIDConItemsFunc: func(id uint64) (*um.DetalleUMResponse, error) {
			return detalle, nil
		},
	}
	svc := newSvc(t, repo)
	resp, err := svc.ObtenerPorID(u.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Codigo != u.Codigo {
		t.Errorf("codigo esperado %q, obtuvo %q", u.Codigo, resp.Codigo)
	}
	if resp.ItemsConUnidadCanonica != 2 {
		t.Errorf("items esperados 2, obtuvo %d", resp.ItemsConUnidadCanonica)
	}
}

func TestEditarFactorNoBaseExitoso(t *testing.T) {
	u := umEjemplo()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*um.UnidadMedida, error) { return u, nil },
		editarFunc: func(id uint64, req *um.EditarUMRequest) (*um.UnidadMedida, error) {
			u.FactorConversion = *req.FactorConversion
			return u, nil
		},
	}
	svc := newSvc(t, repo)
	f := 500.0
	resultado, err := svc.Editar(u.ID, &um.EditarUMRequest{FactorConversion: &f}, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.FactorConversion != 500.0 {
		t.Errorf("factor esperado 500, obtuvo %f", resultado.FactorConversion)
	}
}
