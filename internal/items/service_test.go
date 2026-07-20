package items_test

import (
	"errors"
	"testing"

	it "github.com/manuelgomezsw/loopi-api-v2/internal/items"
)

func newSvc(t *testing.T, repo it.Repository) it.Service {
	t.Helper()
	return it.NewService(repo)
}

// --- Crear ---

func TestCrearItemCodigoDuplicado(t *testing.T) {
	repo := baseMockRepo()
	repo.existeCodigoFunc = func(string) (bool, error) { return true, nil }
	svc := newSvc(t, repo)

	_, err := svc.Crear(crearReqValido(), 1, "admin")
	if !errors.Is(err, it.ErrCodigoDuplicado) {
		t.Errorf("esperaba ErrCodigoDuplicado, obtuvo: %v", err)
	}
}

func TestCrearItemNombreDuplicadoCaseInsensitive(t *testing.T) {
	repo := baseMockRepo()
	repo.existeNombreFunc = func(string, *uint64) (bool, error) { return true, nil }
	svc := newSvc(t, repo)

	req := crearReqValido()
	req.Nombre = "leche entera"
	_, err := svc.Crear(req, 1, "admin")
	if !errors.Is(err, it.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

func TestCrearItemSinCamposObligatorios(t *testing.T) {
	repo := baseMockRepo()
	svc := newSvc(t, repo)

	_, err := svc.Crear(&it.CrearItemRequest{}, 1, "admin")
	var valErr *it.ValidationError
	if !errors.As(err, &valErr) || valErr.Codigo != "codigo_requerido" {
		t.Errorf("esperaba ValidationError codigo_requerido, obtuvo: %v", err)
	}
}

func TestCrearItemSubcategoriaInactiva(t *testing.T) {
	repo := baseMockRepo()
	repo.verificarSubcategoriaFunc = func(uint64) (bool, bool, error) { return true, false, nil }
	svc := newSvc(t, repo)

	_, err := svc.Crear(crearReqValido(), 1, "admin")
	if !errors.Is(err, it.ErrSubcategoriaInactiva) {
		t.Errorf("esperaba ErrSubcategoriaInactiva, obtuvo: %v", err)
	}
}

// --- Editar ---

func TestEditarCodigoAntesDeUso(t *testing.T) {
	repo := baseMockRepo()
	repo.estaEnUsoFunc = func(uint64) (bool, error) { return false, nil }
	svc := newSvc(t, repo)

	req := editarReqValido()
	req.Codigo = "LEC-001-V2"
	resultado, err := svc.Editar(1, req, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado == nil {
		t.Fatal("esperaba respuesta no nula")
	}
}

func TestEditarCodigoDespuesDeUso(t *testing.T) {
	repo := baseMockRepo()
	repo.estaEnUsoFunc = func(uint64) (bool, error) { return true, nil }
	svc := newSvc(t, repo)

	req := editarReqValido()
	req.Codigo = "LEC-001-V2"
	_, err := svc.Editar(1, req, 1, "admin")
	if !errors.Is(err, it.ErrCodigoEnUso) {
		t.Errorf("esperaba ErrCodigoEnUso, obtuvo: %v", err)
	}
}

func TestEditarNombreDuplicadoCaseInsensitive(t *testing.T) {
	repo := baseMockRepo()
	repo.existeNombreFunc = func(string, *uint64) (bool, error) { return true, nil }
	svc := newSvc(t, repo)

	req := editarReqValido()
	req.Nombre = "leche entera premium"
	_, err := svc.Editar(1, req, 1, "admin")
	if !errors.Is(err, it.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

func TestCambiarFrecuenciaNoAfectaHistorialPrevio(t *testing.T) {
	repo := baseMockRepo()
	var idActualizado uint64
	var reqRecibido *it.EditarItemRequest
	llamadasCostos := 0
	repo.actualizarFunc = func(id uint64, req *it.EditarItemRequest, userID uint64) (*it.Item, error) {
		idActualizado = id
		reqRecibido = req
		item := *itemEjemplo()
		item.FrecuenciaInventario = req.FrecuenciaInventario
		return &item, nil
	}
	repo.obtenerDetallePorIDFunc = func(uint64) (*it.ItemConNombres, error) {
		item := *itemEjemplo()
		item.FrecuenciaInventario = "semanal"
		return &it.ItemConNombres{Item: item, SubcategoriaNombre: "Lácteos > Líquidos", UnidadMedidaSimbolo: "ml"}, nil
	}
	repo.listarCostosTiendaFunc = func(uint64) ([]it.CostoPorTienda, error) {
		llamadasCostos++
		return []it.CostoPorTienda{}, nil
	}

	svc := newSvc(t, repo)

	req := editarReqValido()
	req.FrecuenciaInventario = "semanal"
	resultado, err := svc.Editar(1, req, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.FrecuenciaInventario != "semanal" {
		t.Errorf("esperaba frecuencia semanal, obtuvo: %s", resultado.FrecuenciaInventario)
	}
	if idActualizado != 1 || reqRecibido.FrecuenciaInventario != "semanal" {
		t.Errorf("esperaba UPDATE dirigido solo al item 1 con nueva frecuencia")
	}
	if llamadasCostos != 0 {
		t.Errorf("editar la frecuencia no debe tocar el historial de costos por tienda; llamadas=%d", llamadasCostos)
	}
}

// --- Inactivar / Reactivar ---

func TestInactivarItem(t *testing.T) {
	repo := baseMockRepo()
	svc := newSvc(t, repo)

	resultado, err := svc.Inactivar(1, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.Activo {
		t.Errorf("esperaba activo=false")
	}
}

func TestInactivarItemYaInactivo(t *testing.T) {
	repo := baseMockRepo()
	inactivo := *itemEjemplo()
	inactivo.Activo = false
	repo.obtenerPorIDFunc = func(uint64) (*it.Item, error) { return &inactivo, nil }
	svc := newSvc(t, repo)

	_, err := svc.Inactivar(1, 1, "admin")
	if !errors.Is(err, it.ErrItemYaInactivo) {
		t.Errorf("esperaba ErrItemYaInactivo, obtuvo: %v", err)
	}
}

func TestReactivarItem(t *testing.T) {
	repo := baseMockRepo()
	inactivo := *itemEjemplo()
	inactivo.Activo = false
	repo.obtenerPorIDFunc = func(uint64) (*it.Item, error) { return &inactivo, nil }
	svc := newSvc(t, repo)

	resultado, err := svc.Reactivar(1, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !resultado.Activo {
		t.Errorf("esperaba activo=true")
	}
}

// --- Costos por tienda ---

func TestRegistrarCostoTienda(t *testing.T) {
	repo := baseMockRepo()
	svc := newSvc(t, repo)

	resultado, err := svc.RegistrarCostoTienda(1, &it.RegistrarCostoTiendaRequest{TiendaID: 1, CostoUnitario: 3400}, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.CostoUnitario != 3400 {
		t.Errorf("esperaba costo_unitario 3400, obtuvo: %d", resultado.CostoUnitario)
	}
}

func TestCostoVigenteTiendaEsElUltimo(t *testing.T) {
	repo := baseMockRepo()
	repo.listarCostosTiendaFunc = func(uint64) ([]it.CostoPorTienda, error) {
		return []it.CostoPorTienda{
			{
				TiendaID: 1, TiendaNombre: "Sede Norte", CostoVigente: 3600,
				Historial: []it.HistorialEntry{
					{ID: 5, CostoUnitario: 3600},
					{ID: 2, CostoUnitario: 3200},
				},
			},
		}, nil
	}
	svc := newSvc(t, repo)

	resultado, err := svc.ObtenerHistorialCostos(1)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if len(resultado.CostosPorTienda) != 1 || len(resultado.CostosPorTienda[0].Historial) != 2 {
		t.Fatalf("esperaba 1 tienda con historial de 2 entradas, obtuvo: %+v", resultado.CostosPorTienda)
	}
	if resultado.CostosPorTienda[0].CostoVigente != 3600 {
		t.Errorf("esperaba costo_vigente 3600, obtuvo: %d", resultado.CostosPorTienda[0].CostoVigente)
	}
}

// --- Listado ---

func TestListadoPaginadoFiltros(t *testing.T) {
	repo := baseMockRepo()
	var filtrosRecibidos *it.FiltrosListado
	repo.listarFunc = func(filtros *it.FiltrosListado) (*it.ListarItemsResponse, error) {
		filtrosRecibidos = filtros
		return &it.ListarItemsResponse{
			Items:        []it.ItemConNombres{{Item: *itemEjemplo()}},
			Total:        1,
			Pagina:       filtros.Pagina,
			TotalPaginas: 1,
		}, nil
	}
	svc := newSvc(t, repo)

	activo := true
	resultado, err := svc.Listar(&it.FiltrosListado{Tipo: "insumo", Activo: &activo, Pagina: 1, PorPagina: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.Total != 1 || len(resultado.Items) != 1 {
		t.Fatalf("esperaba 1 item en el resultado, obtuvo: %+v", resultado)
	}
	if filtrosRecibidos.Tipo != "insumo" {
		t.Errorf("esperaba filtro tipo=insumo propagado al repositorio, obtuvo: %q", filtrosRecibidos.Tipo)
	}
}
