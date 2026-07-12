package items_test

import (
	"context"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	it "github.com/manuelgomezsw/loopi-api-v2/internal/items"
)

// --- Mock Repository ---

type mockRepo struct {
	crearFunc                 func(req *it.CrearItemRequest, userID uint64) (*it.Item, error)
	existeCodigoFunc          func(codigo string) (bool, error)
	existeNombreFunc          func(nombre string, excludeID *uint64) (bool, error)
	obtenerPorIDFunc          func(id uint64) (*it.Item, error)
	obtenerDetallePorIDFunc   func(id uint64) (*it.ItemConNombres, error)
	listarFunc                func(filtros *it.FiltrosListado) (*it.ListarItemsResponse, error)
	actualizarFunc            func(id uint64, req *it.EditarItemRequest, userID uint64) (*it.Item, error)
	cambiarEstadoFunc         func(id uint64, activo bool, userID uint64) (*it.Item, error)
	estaEnUsoFunc             func(itemID uint64) (bool, error)
	verificarSubcategoriaFunc func(id uint64) (bool, bool, error)
	verificarProveedorFunc    func(id uint64) (bool, bool, error)
	verificarUnidadMedidaFunc func(id uint64) (bool, bool, error)
	verificarTiendaFunc       func(id uint64) (bool, bool, error)
	insertarCostoTiendaFunc   func(itemID uint64, req *it.RegistrarCostoTiendaRequest, userID uint64) (*it.ItemCostoTienda, error)
	listarCostosTiendaFunc    func(itemID uint64) ([]it.CostoPorTienda, error)
}

func (m *mockRepo) Crear(req *it.CrearItemRequest, userID uint64) (*it.Item, error) {
	return m.crearFunc(req, userID)
}
func (m *mockRepo) ExisteCodigo(codigo string) (bool, error) { return m.existeCodigoFunc(codigo) }
func (m *mockRepo) ExisteNombre(nombre string, excludeID *uint64) (bool, error) {
	return m.existeNombreFunc(nombre, excludeID)
}
func (m *mockRepo) ObtenerPorID(id uint64) (*it.Item, error) { return m.obtenerPorIDFunc(id) }
func (m *mockRepo) ObtenerDetallePorID(id uint64) (*it.ItemConNombres, error) {
	return m.obtenerDetallePorIDFunc(id)
}
func (m *mockRepo) Listar(filtros *it.FiltrosListado) (*it.ListarItemsResponse, error) {
	return m.listarFunc(filtros)
}
func (m *mockRepo) Actualizar(id uint64, req *it.EditarItemRequest, userID uint64) (*it.Item, error) {
	return m.actualizarFunc(id, req, userID)
}
func (m *mockRepo) CambiarEstado(id uint64, activo bool, userID uint64) (*it.Item, error) {
	return m.cambiarEstadoFunc(id, activo, userID)
}
func (m *mockRepo) EstaEnUso(itemID uint64) (bool, error) { return m.estaEnUsoFunc(itemID) }
func (m *mockRepo) VerificarSubcategoria(id uint64) (existe, activo bool, err error) {
	return m.verificarSubcategoriaFunc(id)
}
func (m *mockRepo) VerificarProveedor(id uint64) (existe, activo bool, err error) {
	return m.verificarProveedorFunc(id)
}
func (m *mockRepo) VerificarUnidadMedida(id uint64) (existe, activo bool, err error) {
	return m.verificarUnidadMedidaFunc(id)
}
func (m *mockRepo) VerificarTienda(id uint64) (existe, activo bool, err error) {
	return m.verificarTiendaFunc(id)
}
func (m *mockRepo) InsertarCostoTienda(itemID uint64, req *it.RegistrarCostoTiendaRequest, userID uint64) (*it.ItemCostoTienda, error) {
	return m.insertarCostoTiendaFunc(itemID, req, userID)
}
func (m *mockRepo) ListarCostosTienda(itemID uint64) ([]it.CostoPorTienda, error) {
	return m.listarCostosTiendaFunc(itemID)
}

// --- Helpers ---

// itemEjemplo retorna un item activo de referencia, con todas sus referencias activas.
func itemEjemplo() *it.Item {
	return &it.Item{
		ID: 1, Codigo: "LEC-001", Nombre: "Leche Entera", Tipo: "insumo",
		SubcategoriaID: 3, UnidadMedidaID: 5, FrecuenciaInventario: "diario",
		StockSeguridad: "10000.0000", Activo: true, CreadoPor: 1, ActualizadoPor: 1,
	}
}

// baseMockRepo retorna un mockRepo con comportamiento por defecto "todo válido y activo",
// para que cada test sólo tenga que sobreescribir los funcs relevantes al caso bajo prueba.
func baseMockRepo() *mockRepo {
	item := itemEjemplo()
	return &mockRepo{
		crearFunc: func(req *it.CrearItemRequest, userID uint64) (*it.Item, error) {
			return item, nil
		},
		existeCodigoFunc: func(string) (bool, error) { return false, nil },
		existeNombreFunc: func(string, *uint64) (bool, error) { return false, nil },
		obtenerPorIDFunc: func(uint64) (*it.Item, error) { return item, nil },
		obtenerDetallePorIDFunc: func(uint64) (*it.ItemConNombres, error) {
			return &it.ItemConNombres{Item: *item, SubcategoriaNombre: "Lácteos > Líquidos", UnidadMedidaSimbolo: "ml"}, nil
		},
		listarFunc: func(*it.FiltrosListado) (*it.ListarItemsResponse, error) {
			return &it.ListarItemsResponse{Items: []it.ItemConNombres{}, Total: 0, Pagina: 1, TotalPaginas: 0}, nil
		},
		actualizarFunc: func(id uint64, req *it.EditarItemRequest, userID uint64) (*it.Item, error) {
			return item, nil
		},
		cambiarEstadoFunc: func(id uint64, activo bool, userID uint64) (*it.Item, error) {
			actualizado := *item
			actualizado.Activo = activo
			return &actualizado, nil
		},
		estaEnUsoFunc:             func(uint64) (bool, error) { return false, nil },
		verificarSubcategoriaFunc: func(uint64) (bool, bool, error) { return true, true, nil },
		verificarProveedorFunc:    func(uint64) (bool, bool, error) { return true, true, nil },
		verificarUnidadMedidaFunc: func(uint64) (bool, bool, error) { return true, true, nil },
		verificarTiendaFunc:       func(uint64) (bool, bool, error) { return true, true, nil },
		insertarCostoTiendaFunc: func(itemID uint64, req *it.RegistrarCostoTiendaRequest, userID uint64) (*it.ItemCostoTienda, error) {
			return &it.ItemCostoTienda{ID: 1, ItemID: itemID, TiendaID: req.TiendaID, CostoUnitario: req.CostoUnitario}, nil
		},
		listarCostosTiendaFunc: func(uint64) ([]it.CostoPorTienda, error) { return []it.CostoPorTienda{}, nil },
	}
}

func crearReqValido() *it.CrearItemRequest {
	return &it.CrearItemRequest{
		Codigo: "LEC-001", Nombre: "Leche Entera", Tipo: "insumo",
		SubcategoriaID: 3, UnidadMedidaID: 5, FrecuenciaInventario: "diario",
		StockSeguridad: "10000.0000",
	}
}

func editarReqValido() *it.EditarItemRequest {
	return &it.EditarItemRequest{
		Nombre: "Leche Entera", SubcategoriaID: 3, UnidadMedidaID: 5,
		FrecuenciaInventario: "diario", StockSeguridad: "10000.0000",
	}
}

// --- Mock Service ---

type mockSvc struct {
	crearFunc                  func(req *it.CrearItemRequest, userID uint64, rol string) (*it.ItemDetalleResponse, error)
	listarFunc                 func(filtros *it.FiltrosListado) (*it.ListarItemsResponse, error)
	obtenerPorIDFunc           func(id uint64) (*it.ItemDetalleResponse, error)
	editarFunc                 func(id uint64, req *it.EditarItemRequest, userID uint64, rol string) (*it.ItemDetalleResponse, error)
	inactivarFunc              func(id, userID uint64, rol string) (*it.CambiarEstadoResponse, error)
	reactivarFunc              func(id, userID uint64, rol string) (*it.CambiarEstadoResponse, error)
	registrarCostoTiendaFunc   func(itemID uint64, req *it.RegistrarCostoTiendaRequest, userID uint64, rol string) (*it.ItemCostoTienda, error)
	obtenerHistorialCostosFunc func(itemID uint64) (*it.HistorialCostosResponse, error)
}

func (m *mockSvc) Crear(req *it.CrearItemRequest, userID uint64, rol string) (*it.ItemDetalleResponse, error) {
	return m.crearFunc(req, userID, rol)
}
func (m *mockSvc) Listar(filtros *it.FiltrosListado) (*it.ListarItemsResponse, error) {
	return m.listarFunc(filtros)
}
func (m *mockSvc) ObtenerPorID(id uint64) (*it.ItemDetalleResponse, error) {
	return m.obtenerPorIDFunc(id)
}
func (m *mockSvc) Editar(id uint64, req *it.EditarItemRequest, userID uint64, rol string) (*it.ItemDetalleResponse, error) {
	return m.editarFunc(id, req, userID, rol)
}
func (m *mockSvc) Inactivar(id, userID uint64, rol string) (*it.CambiarEstadoResponse, error) {
	return m.inactivarFunc(id, userID, rol)
}
func (m *mockSvc) Reactivar(id, userID uint64, rol string) (*it.CambiarEstadoResponse, error) {
	return m.reactivarFunc(id, userID, rol)
}
func (m *mockSvc) RegistrarCostoTienda(itemID uint64, req *it.RegistrarCostoTiendaRequest, userID uint64, rol string) (*it.ItemCostoTienda, error) {
	return m.registrarCostoTiendaFunc(itemID, req, userID, rol)
}
func (m *mockSvc) ObtenerHistorialCostos(itemID uint64) (*it.HistorialCostosResponse, error) {
	return m.obtenerHistorialCostosFunc(itemID)
}

// --- Helpers de autenticación ---

func ctxAdmin() context.Context {
	claims := &auth.Claims{}
	claims.Subject = "42"
	claims.Rol = "admin"
	return auth.ContextWithClaims(context.Background(), claims)
}

func ctxRol(rol string) context.Context {
	claims := &auth.Claims{}
	claims.Subject = "99"
	claims.Rol = rol
	return auth.ContextWithClaims(context.Background(), claims)
}
