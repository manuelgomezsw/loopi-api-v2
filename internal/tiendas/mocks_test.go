package tiendas_test

import (
	"context"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"github.com/manuelgomezsw/loopi-api-v2/internal/tiendas"
)

// --- Mock TiendaRepository ---

type mockRepo struct {
	crearFunc         func(t tiendas.Tienda) (tiendas.Tienda, error)
	obtenerFunc       func(id uint64) (tiendas.Tienda, error)
	listarFunc        func(estado string, pagina, limite int) ([]tiendas.Tienda, int, error)
	actualizarFunc    func(id uint64, req tiendas.TiendaUpdateRequest, adminID uint64, ahora time.Time) (tiendas.Tienda, error)
	cambiarActivoFunc func(id uint64, activo bool, adminID uint64, ahora time.Time) (tiendas.Tienda, error)
}

func (m *mockRepo) Crear(t tiendas.Tienda) (tiendas.Tienda, error) {
	return m.crearFunc(t)
}

func (m *mockRepo) ObtenerPorID(id uint64) (tiendas.Tienda, error) {
	return m.obtenerFunc(id)
}

func (m *mockRepo) Listar(estado string, pagina, limite int) ([]tiendas.Tienda, int, error) {
	return m.listarFunc(estado, pagina, limite)
}

func (m *mockRepo) Actualizar(id uint64, req tiendas.TiendaUpdateRequest, adminID uint64, ahora time.Time) (tiendas.Tienda, error) {
	return m.actualizarFunc(id, req, adminID, ahora)
}

func (m *mockRepo) CambiarActivo(id uint64, activo bool, adminID uint64, ahora time.Time) (tiendas.Tienda, error) {
	return m.cambiarActivoFunc(id, activo, adminID, ahora)
}

// --- Mock TiendaService ---

type mockService struct {
	listarFunc    func(estado string, pagina, limite int) (tiendas.ListaTiendasResponse, error)
	crearFunc     func(req tiendas.TiendaRequest, adminID uint64) (tiendas.TiendaResponse, error)
	obtenerFunc   func(id uint64) (tiendas.TiendaResponse, error)
	actualizarFunc func(id uint64, req tiendas.TiendaUpdateRequest, adminID uint64) (tiendas.TiendaResponse, error)
	inactivarFunc func(id, adminID uint64) (tiendas.TiendaResponse, error)
	reactivarFunc func(id, adminID uint64) (tiendas.TiendaResponse, error)
}

func (m *mockService) Listar(estado string, pagina, limite int) (tiendas.ListaTiendasResponse, error) {
	return m.listarFunc(estado, pagina, limite)
}

func (m *mockService) Crear(req tiendas.TiendaRequest, adminID uint64) (tiendas.TiendaResponse, error) {
	return m.crearFunc(req, adminID)
}

func (m *mockService) ObtenerPorID(id uint64) (tiendas.TiendaResponse, error) {
	return m.obtenerFunc(id)
}

func (m *mockService) Actualizar(id uint64, req tiendas.TiendaUpdateRequest, adminID uint64) (tiendas.TiendaResponse, error) {
	return m.actualizarFunc(id, req, adminID)
}

func (m *mockService) Inactivar(id, adminID uint64) (tiendas.TiendaResponse, error) {
	return m.inactivarFunc(id, adminID)
}

func (m *mockService) Reactivar(id, adminID uint64) (tiendas.TiendaResponse, error) {
	return m.reactivarFunc(id, adminID)
}

// --- Helpers de autenticación ---

// ctxAdmin retorna un context con claims de admin para tests de handlers.
func ctxAdmin() context.Context {
	claims := &auth.Claims{}
	claims.Subject = "42"
	claims.Rol = "admin"
	return auth.ContextWithClaims(context.Background(), claims)
}

// ctxRol retorna un context con el rol especificado.
func ctxRol(rol string) context.Context {
	claims := &auth.Claims{}
	claims.Subject = "99"
	claims.Rol = rol
	return auth.ContextWithClaims(context.Background(), claims)
}

// tiendaEjemplo retorna una Tienda de prueba con todos los campos completos.
func tiendaEjemplo() tiendas.Tienda {
	return tiendas.Tienda{
		ID:        1,
		Codigo:    "TDA-001",
		Nombre:    "Tienda Norte",
		Direccion: "Calle 100 #20-30",
		Ciudad:    "Bogotá",
		Telefono:  "3001234567",
		Activo:    true,
		CreadoPor: 42,
		CreadoEn:  time.Now(),
	}
}
