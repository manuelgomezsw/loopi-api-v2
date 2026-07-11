package proveedores_test

import (
	"context"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	pv "github.com/manuelgomezsw/loopi-api-v2/internal/proveedores"
)

// --- Mock Repository ---

type mockRepo struct {
	crearFunc                func(req *pv.CrearProveedorRequest) (*pv.Proveedor, error)
	existeNITFunc            func(nit string, excludeID *uint64) (bool, error)
	obtenerPorIDFunc         func(id uint64) (*pv.Proveedor, error)
	obtenerPorIDConItemsFunc func(id uint64) (*pv.ProveedorDetalleResponse, error)
	listarFunc               func(filtros *pv.FiltrosListado) (*pv.ListarProveedoresResponse, error)
	actualizarFunc           func(id uint64, req *pv.EditarProveedorRequest) (*pv.Proveedor, error)
	cambiarEstadoFunc        func(id uint64, activo bool) error
	contarItemsAsignadosFunc func(id uint64) (int, error)
}

func (m *mockRepo) Crear(req *pv.CrearProveedorRequest) (*pv.Proveedor, error) {
	return m.crearFunc(req)
}
func (m *mockRepo) ExisteNIT(nit string, excludeID *uint64) (bool, error) {
	return m.existeNITFunc(nit, excludeID)
}
func (m *mockRepo) ObtenerPorID(id uint64) (*pv.Proveedor, error) {
	return m.obtenerPorIDFunc(id)
}
func (m *mockRepo) ObtenerPorIDConItems(id uint64) (*pv.ProveedorDetalleResponse, error) {
	return m.obtenerPorIDConItemsFunc(id)
}
func (m *mockRepo) Listar(filtros *pv.FiltrosListado) (*pv.ListarProveedoresResponse, error) {
	return m.listarFunc(filtros)
}
func (m *mockRepo) Actualizar(id uint64, req *pv.EditarProveedorRequest) (*pv.Proveedor, error) {
	return m.actualizarFunc(id, req)
}
func (m *mockRepo) CambiarEstado(id uint64, activo bool) error {
	return m.cambiarEstadoFunc(id, activo)
}
func (m *mockRepo) ContarItemsAsignados(id uint64) (int, error) {
	return m.contarItemsAsignadosFunc(id)
}

// --- Mock Service ---

type mockSvc struct {
	crearFunc        func(req *pv.CrearProveedorRequest, userID uint64, rol string) (*pv.Proveedor, error)
	listarFunc       func(filtros *pv.FiltrosListado) (*pv.ListarProveedoresResponse, error)
	obtenerPorIDFunc func(id uint64) (*pv.ProveedorDetalleResponse, error)
	editarFunc       func(id uint64, req *pv.EditarProveedorRequest, userID uint64, rol string) (*pv.Proveedor, error)
	inactivarFunc    func(id, userID uint64, rol string) (*pv.CambiarEstadoResponse, error)
	activarFunc      func(id, userID uint64, rol string) (*pv.CambiarEstadoResponse, error)
}

func (m *mockSvc) Crear(req *pv.CrearProveedorRequest, userID uint64, rol string) (*pv.Proveedor, error) {
	return m.crearFunc(req, userID, rol)
}
func (m *mockSvc) Listar(filtros *pv.FiltrosListado) (*pv.ListarProveedoresResponse, error) {
	return m.listarFunc(filtros)
}
func (m *mockSvc) ObtenerPorID(id uint64) (*pv.ProveedorDetalleResponse, error) {
	return m.obtenerPorIDFunc(id)
}
func (m *mockSvc) Editar(id uint64, req *pv.EditarProveedorRequest, userID uint64, rol string) (*pv.Proveedor, error) {
	return m.editarFunc(id, req, userID, rol)
}
func (m *mockSvc) Inactivar(id, userID uint64, rol string) (*pv.CambiarEstadoResponse, error) {
	return m.inactivarFunc(id, userID, rol)
}
func (m *mockSvc) Activar(id, userID uint64, rol string) (*pv.CambiarEstadoResponse, error) {
	return m.activarFunc(id, userID, rol)
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

func proveedorEjemplo() *pv.Proveedor {
	return &pv.Proveedor{
		ID:               1,
		RazonSocial:      "Distribuidora La Cosecha S.A.S",
		NIT:              "900123456-7",
		NombreContacto:   "Carlos Rodríguez",
		TelefonoContacto: "3001234567",
		Activo:           true,
	}
}

func strPtr(s string) *string { return &s }
