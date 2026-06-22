package unidades_medida_test

import (
	"context"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	um "github.com/manuelgomezsw/loopi-api-v2/internal/unidades_medida"
)

// --- Mock UMRepository ---

type mockRepo struct {
	crearFunc                       func(req *um.CrearUMRequest) (*um.UnidadMedida, error)
	existeConCodigoFunc             func(codigo string) (bool, error)
	obtenerPorIDFunc                func(id uint64) (*um.UnidadMedida, error)
	obtenerPorIDConItemsFunc        func(id uint64) (*um.DetalleUMResponse, error)
	listarFunc                      func(params *um.ListarUMParams) (*um.ListarUMResponse, error)
	editarFunc                      func(id uint64, req *um.EditarUMRequest) (*um.UnidadMedida, error)
	inactivarFunc                   func(id uint64) error
	contarItemsFunc                 func(id uint64) (int, error)
	contarUnidadesActivasPorTipoFunc func(tipo string, excludeID uint64) (int, error)
}

func (m *mockRepo) Crear(req *um.CrearUMRequest) (*um.UnidadMedida, error) {
	return m.crearFunc(req)
}
func (m *mockRepo) ExisteConCodigo(codigo string) (bool, error) {
	return m.existeConCodigoFunc(codigo)
}
func (m *mockRepo) ObtenerPorID(id uint64) (*um.UnidadMedida, error) {
	return m.obtenerPorIDFunc(id)
}
func (m *mockRepo) ObtenerPorIDConItems(id uint64) (*um.DetalleUMResponse, error) {
	return m.obtenerPorIDConItemsFunc(id)
}
func (m *mockRepo) Listar(params *um.ListarUMParams) (*um.ListarUMResponse, error) {
	return m.listarFunc(params)
}
func (m *mockRepo) Editar(id uint64, req *um.EditarUMRequest) (*um.UnidadMedida, error) {
	return m.editarFunc(id, req)
}
func (m *mockRepo) Inactivar(id uint64) error {
	return m.inactivarFunc(id)
}
func (m *mockRepo) ContarItemsConUnidadCanonica(id uint64) (int, error) {
	return m.contarItemsFunc(id)
}
func (m *mockRepo) ContarUnidadesActivasPorTipo(tipo string, excludeID uint64) (int, error) {
	return m.contarUnidadesActivasPorTipoFunc(tipo, excludeID)
}

// --- Mock UMService ---

type mockSvc struct {
	crearFunc          func(req *um.CrearUMRequest, userID uint64, rol string) (*um.UnidadMedida, error)
	inactivarFunc      func(id, userID uint64, rol string) (*um.InactivarResponse, error)
	obtenerImpactoFunc func(id uint64) (*um.ImpactoResponse, error)
	listarFunc         func(params *um.ListarUMParams) (*um.ListarUMResponse, error)
	obtenerPorIDFunc   func(id uint64) (*um.DetalleUMResponse, error)
	editarFunc         func(id uint64, req *um.EditarUMRequest, userID uint64, rol string) (*um.UnidadMedida, error)
}

func (m *mockSvc) Crear(req *um.CrearUMRequest, userID uint64, rol string) (*um.UnidadMedida, error) {
	return m.crearFunc(req, userID, rol)
}
func (m *mockSvc) Inactivar(id, userID uint64, rol string) (*um.InactivarResponse, error) {
	return m.inactivarFunc(id, userID, rol)
}
func (m *mockSvc) ObtenerImpacto(id uint64) (*um.ImpactoResponse, error) {
	return m.obtenerImpactoFunc(id)
}
func (m *mockSvc) Listar(params *um.ListarUMParams) (*um.ListarUMResponse, error) {
	return m.listarFunc(params)
}
func (m *mockSvc) ObtenerPorID(id uint64) (*um.DetalleUMResponse, error) {
	return m.obtenerPorIDFunc(id)
}
func (m *mockSvc) Editar(id uint64, req *um.EditarUMRequest, userID uint64, rol string) (*um.UnidadMedida, error) {
	return m.editarFunc(id, req, userID, rol)
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

func umEjemplo() *um.UnidadMedida {
	return &um.UnidadMedida{
		ID:               4,
		Codigo:           "kg",
		Nombre:           "Kilogramo",
		TipoMedida:       "peso",
		FactorConversion: 1000.0,
		UnidadBase:       false,
		Activo:           true,
	}
}

func umBase() *um.UnidadMedida {
	return &um.UnidadMedida{
		ID:               1,
		Codigo:           "g",
		Nombre:           "Gramo",
		TipoMedida:       "peso",
		FactorConversion: 1.0,
		UnidadBase:       true,
		Activo:           true,
	}
}
